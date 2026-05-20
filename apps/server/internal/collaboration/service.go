package collaboration

import (
	"log"
	"time"

	"github.com/bin-ke/my-notion/pkg/db"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Service handles persistence of Yjs document snapshots to PostgreSQL.
type Service struct {
	DB *gorm.DB
}

func NewService(database *gorm.DB) *Service {
	return &Service{DB: database}
}

// SaveSnapshot upserts the page snapshot (page_id unique).
func (s *Service) SaveSnapshot(pageID uint, docBinary []byte) error {
	snap := db.PageSnapshot{
		PageID:    pageID,
		DocBinary: docBinary,
		Version:   0,
	}
	// Upsert: update DocBinary and Version if page_id already exists
	return s.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "page_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"doc_binary", "version", "updated_at"}),
	}).Create(&snap).Error
}

// LoadSnapshot loads the snapshot for a page from the database.
func (s *Service) LoadSnapshot(pageID uint) ([]byte, error) {
	var snap db.PageSnapshot
	err := s.DB.Where("page_id = ?", pageID).First(&snap).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return snap.DocBinary, nil
}

// LoadAllSnapshots loads all snapshots and populates the docstore.
func (s *Service) LoadAllSnapshots(docStore *DocStore) error {
	var snaps []db.PageSnapshot
	if err := s.DB.Find(&snaps).Error; err != nil {
		return err
	}
	for _, snap := range snaps {
		if len(snap.DocBinary) > 0 {
			docStore.LoadFromDB(snap.PageID, snap.DocBinary)
		}
	}
	log.Printf("collab: loaded %d page snapshots from DB", len(snaps))
	return nil
}

// StartFlushLoop runs a periodic goroutine that flushes dirty documents to DB.
func StartFlushLoop(docStore *DocStore, service *Service, hub *Hub, interval time.Duration) {
	go func() {
		for {
			time.Sleep(interval)
			dirty := docStore.DirtyPages()
			for _, pageID := range dirty {
				snapshot := docStore.Snapshot(pageID)
				if len(snapshot) == 0 {
					continue
				}
				if err := service.SaveSnapshot(pageID, snapshot); err != nil {
					log.Printf("collab: failed to flush page %d: %v", pageID, err)
					continue
				}
				docStore.MarkClean(pageID)
			}
			if len(dirty) > 0 {
				log.Printf("collab: flushed %d pages", len(dirty))
			}
		}
	}()

	// Stale eviction loop — evict pages idle for 5 minutes after flushing
	go func() {
		for {
			time.Sleep(1 * time.Minute)
			docStore.EvictStale(5 * time.Minute)
		}
	}()
}

// FlushOnEmpty flushes a page when its last client disconnects.
func FlushOnEmpty(docStore *DocStore, service *Service, pageID uint) {
	snapshot := docStore.Snapshot(pageID)
	if len(snapshot) == 0 {
		return
	}
	if err := service.SaveSnapshot(pageID, snapshot); err != nil {
		log.Printf("collab: failed to flush page %d on disconnect: %v", pageID, err)
		return
	}
	docStore.MarkClean(pageID)
	log.Printf("collab: flushed page %d (last client disconnected)", pageID)
}
