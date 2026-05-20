package collaboration

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512 * 1024 // 512KB
	sendBufSize    = 256
)

// NewClient creates a client and starts its read/write pumps.
func NewClient(hub *Hub, conn *websocket.Conn, pageID, userID uint, userName string) *Client {
	c := &Client{
		hub:      hub,
		conn:     conn,
		pageID:   pageID,
		userID:   userID,
		userName: userName,
		send:     make(chan []byte, sendBufSize),
		sendJSON: make(chan any, sendBufSize),
		done:     make(chan struct{}),
	}
	hub.Register(c)

	go c.writePump()
	go c.readPump()

	return c
}

func (c *Client) readPump() {
	defer func() {
		c.hub.Unregister(c)
		close(c.done)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		msgType, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("collab: ws error for user %d on page %d: %v", c.userID, c.pageID, err)
			}
			return
		}

		switch msgType {
		case websocket.BinaryMessage:
			c.handleBinary(data)
		case websocket.TextMessage:
			c.handleJSON(data)
		}
	}
}

func (c *Client) handleBinary(data []byte) {
	if len(data) == 0 {
		return
	}
	msgType := data[0]
	payload := data[1:]

	switch msgType {
	case 1: // sync_update from client
		c.hub.AppendUpdate(c.pageID, c, payload)
	}
}

func (c *Client) handleJSON(data []byte) {
	var msg map[string]any
	if err := json.Unmarshal(data, &msg); err != nil {
		return
	}
	msgType, _ := msg["type"].(string)
	if msgType == "awareness" {
		c.hub.BroadcastAwareness(c.pageID, c, msg)
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.BinaryMessage, msg); err != nil {
				return
			}

		case msg, ok := <-c.sendJSON:
			if !ok {
				return
			}
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			jsonBytes, err := json.Marshal(msg)
			if err != nil {
				continue
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, jsonBytes); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		case <-c.done:
			return
		}
	}
}

// SendInit sends the initial sync_init message (message type 0) containing the full document.
func (c *Client) SendInit(data []byte) bool {
	select {
	case c.send <- data:
		return true
	default:
		return false
	}
}

// Close cleanly shuts down the client.
func (c *Client) Close() {
	c.hub.Unregister(c)
	select {
	case <-c.done:
		// already closed
	default:
		close(c.done)
	}
	c.conn.Close()
}
