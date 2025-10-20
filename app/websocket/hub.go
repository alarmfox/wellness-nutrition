package websocket

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// NotificationType represents the type of notification
type NotificationType string

const (
	NotificationBookingCreated NotificationType = "booking_created"
	NotificationBookingDeleted NotificationType = "booking_deleted"
)

// Notification represents a WebSocket notification message
type Notification struct {
	Type      NotificationType `json:"type"`
	Message   string           `json:"message"`
	UserName  string           `json:"userName"`
	SlotTime  string           `json:"slotTime"`
	Timestamp time.Time        `json:"timestamp"`
}

// Hub maintains active WebSocket connections and broadcasts messages
type Hub struct {
	// Registered clients
	clients map[*websocket.Conn]bool
	
	// Mutex for thread-safe operations
	mu sync.RWMutex
	
	// Channel for broadcasting notifications
	broadcast chan *Notification
	
	// Channel to register new clients
	register chan *websocket.Conn
	
	// Channel to unregister clients
	unregister chan *websocket.Conn
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan *Notification, 256),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case conn := <-h.register:
			h.mu.Lock()
			h.clients[conn] = true
			h.mu.Unlock()
			log.Printf("WebSocket client connected. Total clients: %d", len(h.clients))
			
		case conn := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[conn]; ok {
				delete(h.clients, conn)
				conn.Close()
			}
			h.mu.Unlock()
			log.Printf("WebSocket client disconnected. Total clients: %d", len(h.clients))
			
		case notification := <-h.broadcast:
			h.mu.RLock()
			for conn := range h.clients {
				err := conn.WriteJSON(notification)
				if err != nil {
					log.Printf("Error writing to WebSocket: %v", err)
					// Unregister client on write error
					h.mu.RUnlock()
					h.Unregister(conn)
					h.mu.RLock()
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Register adds a new client connection
func (h *Hub) Register(conn *websocket.Conn) {
	h.register <- conn
}

// Unregister removes a client connection
func (h *Hub) Unregister(conn *websocket.Conn) {
	h.unregister <- conn
}

// Broadcast sends a notification to all connected clients
func (h *Hub) Broadcast(notification *Notification) {
	notification.Timestamp = time.Now()
	select {
	case h.broadcast <- notification:
	default:
		log.Printf("Broadcast channel full, dropping notification")
	}
}

// BroadcastJSON sends a JSON notification to all connected clients
func (h *Hub) BroadcastJSON(notificationType NotificationType, message, userName, slotTime string) {
	notification := &Notification{
		Type:     notificationType,
		Message:  message,
		UserName: userName,
		SlotTime: slotTime,
	}
	h.Broadcast(notification)
}

// GetClientCount returns the number of connected clients
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// MarshalJSON custom marshaler for debugging
func (n *Notification) MarshalJSON() ([]byte, error) {
	type Alias Notification
	return json.Marshal(&struct {
		*Alias
		Timestamp string `json:"timestamp"`
	}{
		Alias:     (*Alias)(n),
		Timestamp: n.Timestamp.Format(time.RFC3339),
	})
}
