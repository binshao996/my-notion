package collaboration

import (
	"sync"
	"time"
)

// DocStore holds in-memory Yjs update logs per page.
// Backed by Service for DB persistence.
type DocStore struct {
	mu        sync.RWMutex
	updates   map[uint][][]byte // pageID → ordered update log
	dirty     map[uint]bool     // pages with unpersisted changes
	lastUsed  map[uint]time.Time
	onEvict   func(pageID uint) // called when evicting a page (to flush first)
}

func NewDocStore() *DocStore {
	return &DocStore{
		updates:  make(map[uint][][]byte),
		dirty:    make(map[uint]bool),
		lastUsed: make(map[uint]time.Time),
	}
}

// SetOnEvict sets a callback that runs before a page is evicted from memory.
func (ds *DocStore) SetOnEvict(fn func(pageID uint)) {
	ds.onEvict = fn
}

// AppendUpdate adds a single update to the page's log.
func (ds *DocStore) AppendUpdate(pageID uint, data []byte) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.updates[pageID] = append(ds.updates[pageID], data)
	ds.dirty[pageID] = true
	ds.lastUsed[pageID] = time.Now()
}

// GetUpdates returns a copy of the full update log for a page.
func (ds *DocStore) GetUpdates(pageID uint) [][]byte {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	log := ds.updates[pageID]
	if log == nil {
		return nil
	}
	out := make([][]byte, len(log))
	copy(out, log)
	ds.lastUsed[pageID] = time.Now()
	return out
}

// GetEncodedDocument returns the full document as a sync_init binary message.
func (ds *DocStore) GetEncodedDocument(pageID uint) []byte {
	updates := ds.GetUpdates(pageID)
	return EncodeFullDocument(updates)
}

// Snapshot returns all updates concatenated (raw bytea for DB storage).
func (ds *DocStore) Snapshot(pageID uint) []byte {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	log := ds.updates[pageID]
	if log == nil {
		return nil
	}
	var total int
	for _, u := range log {
		total += len(u)
	}
	buf := make([]byte, 0, total)
	for _, u := range log {
		buf = append(buf, u...)
	}
	return buf
}

// LoadFromDB populates the in-memory store from a DB snapshot.
func (ds *DocStore) LoadFromDB(pageID uint, snapshot []byte) {
	if len(snapshot) == 0 {
		return
	}
	ds.mu.Lock()
	defer ds.mu.Unlock()
	// Store the snapshot as a single update (the full merged state)
	ds.updates[pageID] = [][]byte{snapshot}
	ds.dirty[pageID] = false
	ds.lastUsed[pageID] = time.Now()
}

// MarkClean clears the dirty flag after a successful DB flush.
func (ds *DocStore) MarkClean(pageID uint) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.dirty[pageID] = false
}

// DirtyPages returns page IDs that have unpersisted changes.
func (ds *DocStore) DirtyPages() []uint {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	var ids []uint
	for id, d := range ds.dirty {
		if d {
			ids = append(ids, id)
		}
	}
	return ids
}

// EvictStale removes pages that have been idle for longer than the given duration.
// Calls onEvict for each evicted page (to flush before removal).
func (ds *DocStore) EvictStale(maxAge time.Duration) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	for id, last := range ds.lastUsed {
		if last.Before(cutoff) {
			if ds.onEvict != nil {
				ds.onEvict(id)
			}
			delete(ds.updates, id)
			delete(ds.dirty, id)
			delete(ds.lastUsed, id)
		}
	}
}
