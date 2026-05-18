package websocket

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/google/uuid"
)

// Hub maintains the set of active clients and broadcasts messages to the clients.
type Hub struct {
	// Registered clients by User ID.
	clients map[uuid.UUID]map[*Client]bool

	// Inbound messages from the clients.
	broadcast chan []byte

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	mu sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[uuid.UUID]map[*Client]bool),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.clients[client.UserID] == nil {
				h.clients[client.UserID] = make(map[*Client]bool)
			}
			h.clients[client.UserID][client] = true
			h.mu.Unlock()
			log.Printf("User %s connected to WebSocket", client.UserID)

		case client := <-h.unregister:
			h.mu.Lock()
			if userClients, ok := h.clients[client.UserID]; ok {
				if _, ok := userClients[client]; ok {
					delete(userClients, client)
					close(client.send)
				}
				if len(userClients) == 0 {
					delete(h.clients, client.UserID)
				}
				log.Printf("User %s disconnected from WebSocket", client.UserID)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			// Handle direct message routing
			var msg struct {
				ReceiverID uuid.UUID `json:"receiver_id"`
			}
			if err := json.Unmarshal(message, &msg); err == nil {
				h.mu.RLock()
				for client := range h.clients[msg.ReceiverID] {
					select {
					case client.send <- message:
					default:
						log.Printf("WebSocket send buffer full for user %s", msg.ReceiverID)
					}
				}
				h.mu.RUnlock()
			}
		}
	}
}

func (h *Hub) BroadcastToUser(userID uuid.UUID, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for client := range h.clients[userID] {
		select {
		case client.send <- message:
		default:
			log.Printf("WebSocket send buffer full for user %s", userID)
		}
	}
}
