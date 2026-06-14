package websocket

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
	maxMsgSize = 512
)

// SeatEvent is sent to all WebSocket clients watching a showtime.
type SeatEvent struct {
	SeatID     string `json:"seat_id"`
	ShowtimeID string `json:"showtime_id"`
	Status     string `json:"status"` // AVAILABLE | LOCKED | BOOKED
}

// Client is a single WebSocket connection.
type Client struct {
	hub        *Hub
	showtimeID string
	conn       *websocket.Conn
	send       chan []byte
}

// Hub manages all WebSocket clients, grouped by showtime.
// All room mutations run inside the single Run() goroutine.
type Hub struct {
	rooms      map[string]map[*Client]struct{} // showtimeID → set of clients
	register   chan *Client
	unregister chan *Client
	broadcast  chan roomMsg
}

type roomMsg struct {
	showtimeID string
	payload    []byte
}

// NewHub creates an initialised Hub ready to be started.
func NewHub() *Hub {
	return &Hub{
		rooms:      make(map[string]map[*Client]struct{}),
		register:   make(chan *Client, 64),
		unregister: make(chan *Client, 64),
		broadcast:  make(chan roomMsg, 256),
	}
}

// Run is the Hub's event loop — start it in a goroutine.
func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			if h.rooms[c.showtimeID] == nil {
				h.rooms[c.showtimeID] = make(map[*Client]struct{})
			}
			h.rooms[c.showtimeID][c] = struct{}{}
			log.Printf("[Hub] joined showtime=%s clients=%d", c.showtimeID, len(h.rooms[c.showtimeID]))

		case c := <-h.unregister:
			if room, ok := h.rooms[c.showtimeID]; ok {
				delete(room, c)
				close(c.send)
				if len(room) == 0 {
					delete(h.rooms, c.showtimeID)
				}
			}
			log.Printf("[Hub] left showtime=%s", c.showtimeID)

		case msg := <-h.broadcast:
			for c := range h.rooms[msg.showtimeID] {
				select {
				case c.send <- msg.payload:
				default:
					// Slow client — evict
					delete(h.rooms[msg.showtimeID], c)
					close(c.send)
				}
			}
		}
	}
}

// BroadcastSeatEvent sends a seat-status update to all clients for a showtime.
func (h *Hub) BroadcastSeatEvent(event SeatEvent) {
	payload, err := json.Marshal(event)
	if err != nil {
		log.Printf("[Hub] marshal error: %v", err)
		return
	}
	h.broadcast <- roomMsg{showtimeID: event.ShowtimeID, payload: payload}
}

// ServeClient registers a connection and blocks until it disconnects.
func (h *Hub) ServeClient(conn *websocket.Conn, showtimeID string) {
	c := &Client{
		hub:        h,
		showtimeID: showtimeID,
		conn:       conn,
		send:       make(chan []byte, 128),
	}
	h.register <- c
	go c.writePump()
	c.readPump()
}

// readPump drains incoming messages (client → server) and detects disconnection.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMsgSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			break
		}
	}
}

// writePump flushes messages from the send channel and sends periodic pings.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}