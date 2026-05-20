package collaboration

import (
	"encoding/binary"
	"sync"

	"github.com/gorilla/websocket"
)

// Hub manages all collaboration rooms. One room per page.
type Hub struct {
	mu    sync.RWMutex
	rooms map[uint]*Room
}

// Room holds connected clients and the document update log for a single page.
type Room struct {
	mu        sync.RWMutex
	clients   map[*Client]struct{}
	updateLog [][]byte
}

// Client represents a single WebSocket connection.
type Client struct {
	hub      *Hub
	room     *Room
	conn     *websocket.Conn
	pageID   uint
	userID   uint
	userName string

	send     chan []byte // binary messages (Yjs updates)
	sendJSON chan any    // JSON messages (awareness)
	done     chan struct{}
}

func NewHub() *Hub {
	return &Hub{
		rooms: make(map[uint]*Room),
	}
}

func (h *Hub) getOrCreateRoom(pageID uint) *Room {
	h.mu.Lock()
	defer h.mu.Unlock()
	if room, ok := h.rooms[pageID]; ok {
		return room
	}
	room := &Room{
		clients:   make(map[*Client]struct{}),
		updateLog: make([][]byte, 0),
	}
	h.rooms[pageID] = room
	return room
}

func (h *Hub) Register(client *Client) {
	room := h.getOrCreateRoom(client.pageID)
	client.room = room
	room.mu.Lock()
	room.clients[client] = struct{}{}
	room.mu.Unlock()
}

func (h *Hub) Unregister(client *Client) {
	if client.room == nil {
		return
	}
	room := client.room
	room.mu.Lock()
	delete(room.clients, client)
	remaining := len(room.clients)
	room.mu.Unlock()

	// Clean up empty rooms after a delay (handled by caller via docstore)
	if remaining == 0 {
		h.mu.Lock()
		// Keep room in map — docstore will flush and evict later
		h.mu.Unlock()
	}
}

// AppendUpdate stores an update and broadcasts to all other clients in the room.
func (h *Hub) AppendUpdate(pageID uint, sender *Client, data []byte) {
	room := sender.room
	if room == nil {
		return
	}
	room.mu.Lock()
	room.updateLog = append(room.updateLog, data)
	room.mu.Unlock()
	h.BroadcastUpdate(pageID, sender, data)
}

// BroadcastUpdate sends a binary update to all clients in the room except the sender.
func (h *Hub) BroadcastUpdate(pageID uint, sender *Client, data []byte) {
	if sender.room == nil {
		return
	}
	// Prepend message type 1 (sync_update)
	msg := make([]byte, 1+len(data))
	msg[0] = 1
	copy(msg[1:], data)

	room := sender.room
	room.mu.RLock()
	defer room.mu.RUnlock()
	for client := range room.clients {
		if client != sender {
			select {
			case client.send <- msg:
			default:
				// client's send buffer is full, skip
			}
		}
	}
}

// BroadcastAwareness sends an awareness message to all clients in the room except the sender.
func (h *Hub) BroadcastAwareness(pageID uint, sender *Client, msg any) {
	if sender.room == nil {
		return
	}
	room := sender.room
	room.mu.RLock()
	defer room.mu.RUnlock()
	for client := range room.clients {
		if client != sender {
			select {
			case client.sendJSON <- msg:
			default:
			}
		}
	}
}

// GetUpdateLog returns a copy of the full update log for a page.
func (h *Hub) GetUpdateLog(pageID uint) [][]byte {
	room := h.getOrCreateRoom(pageID)
	room.mu.RLock()
	defer room.mu.RUnlock()
	log := make([][]byte, len(room.updateLog))
	copy(log, room.updateLog)
	return log
}

// EncodeFullDocument encodes the full update log as a sync_init message:
// concatenated updates, each prefixed with a varint length.
func EncodeFullDocument(updates [][]byte) []byte {
	// Calculate total size: 1 byte type + for each update: varint length + data
	size := 1 // message type 0
	for _, u := range updates {
		size += binary.MaxVarintLen64 + len(u)
	}
	buf := make([]byte, 0, size)
	buf = append(buf, 0) // sync_init message type
	for _, u := range updates {
		lenBuf := make([]byte, binary.MaxVarintLen64)
		n := binary.PutUvarint(lenBuf, uint64(len(u)))
		buf = append(buf, lenBuf[:n]...)
		buf = append(buf, u...)
	}
	return buf
}

// RoomClientCount returns the number of connected clients in a room.
func (h *Hub) RoomClientCount(pageID uint) int {
	h.mu.RLock()
	room, ok := h.rooms[pageID]
	h.mu.RUnlock()
	if !ok {
		return 0
	}
	room.mu.RLock()
	defer room.mu.RUnlock()
	return len(room.clients)
}
