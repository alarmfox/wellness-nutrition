package websocket

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"
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
	clients map[*Client]bool

	// Mutex for thread-safe operations
	mu sync.RWMutex

	// Channel for broadcasting notifications
	broadcast chan *Notification

	// Channel to register new clients
	register chan *Client

	// Channel to unregister clients
	unregister chan *Client
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan *Notification, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()

		case notification := <-h.broadcast:
			data, err := json.Marshal(notification)
			if err != nil {
				log.Printf("Error marshaling notification: %v", err)
				continue
			}

			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- data:
				default:
					// Client send buffer is full, close it
					h.mu.RUnlock()
					close(client.send)
					delete(h.clients, client)
					h.mu.RLock()
				}
			}
			h.mu.RUnlock()
		case <-ctx.Done():
			return
		}
	}
}

// Register adds a new client connection
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister removes a client connection
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
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
