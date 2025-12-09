package services

import (
	"encoding/json"
	"log"
	"my-ecomm/config"
	"my-ecomm/models"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512 * 1024 // 512KB
)

// Message represents a websocket message structure
type Message struct {
	Type      string      `json:"type"`
	Content   interface{} `json:"content"`
	Timestamp time.Time   `json:"timestamp,omitempty"`
	UserID    uint        `json:"userId,omitempty"`
	Username  string      `json:"username,omitempty"`
}

// Client represents a websocket client
type Client struct {
	ID       uint
	Username string
	RoomID   uint
	Conn     *websocket.Conn
	Send     chan []byte
	Hub      *Hub
}

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	// Registered clients by room
	Rooms map[uint]map[*Client]bool

	// Inbound messages from clients
	Broadcast chan *BroadcastMessage

	// Register requests from clients
	Register chan *Client

	// Unregister requests from clients
	Unregister chan *Client

	mu sync.RWMutex
}

type BroadcastMessage struct {
	RoomID  uint
	Message []byte
}

var hubInstance *Hub
var once sync.Once

// GetHub returns singleton instance of Hub
func GetHub() *Hub {
	once.Do(func() {
		hubInstance = &Hub{
			Broadcast:  make(chan *BroadcastMessage),
			Register:   make(chan *Client),
			Unregister: make(chan *Client),
			Rooms:      make(map[uint]map[*Client]bool),
		}
		go hubInstance.Run()
	})
	return hubInstance
}

// Run starts the hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			if h.Rooms[client.RoomID] == nil {
				h.Rooms[client.RoomID] = make(map[*Client]bool)
			}
			h.Rooms[client.RoomID][client] = true
			log.Printf("Client %d registered to room %d. Total clients in room: %d",
				client.ID, client.RoomID, len(h.Rooms[client.RoomID]))
			h.mu.Unlock()

			// Notify other clients in the room about new user
			h.broadcastUserJoined(client)

		case client := <-h.Unregister:
			h.mu.Lock()
			if clients, ok := h.Rooms[client.RoomID]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.Send)
					log.Printf("Client %d unregistered from room %d. Remaining clients: %d",
						client.ID, client.RoomID, len(clients))
					if len(clients) == 0 {
						delete(h.Rooms, client.RoomID)
						log.Printf("Room %d deleted (no clients)", client.RoomID)
					}
				}
			}
			h.mu.Unlock()

			// Notify other clients about user leaving
			h.broadcastUserLeft(client)

		case broadcast := <-h.Broadcast:
			h.mu.RLock()
			if clients, ok := h.Rooms[broadcast.RoomID]; ok {
				log.Printf("Broadcasting message to room %d, clients: %d", broadcast.RoomID, len(clients))
				for client := range clients {
					select {
					case client.Send <- broadcast.Message:
					default:
						close(client.Send)
						delete(clients, client)
						log.Printf("Removed client %d from room %d (send failed)", client.ID, broadcast.RoomID)
					}
				}
			}
			h.mu.RUnlock()

		}
	}
}

// broadcastUserJoined notifies room that a user joined
func (h *Hub) broadcastUserJoined(client *Client) {
	msg := Message{
		Type:      "user_joined",
		Content:   client.Username + " joined the room",
		Timestamp: time.Now(),
		UserID:    client.ID,
		Username:  client.Username,
	}

	if data, err := json.Marshal(msg); err == nil {
		h.Broadcast <- &BroadcastMessage{
			RoomID:  client.RoomID,
			Message: data,
		}
	}
}

// broadcastUserLeft notifies room that a user left
func (h *Hub) broadcastUserLeft(client *Client) {
	msg := Message{
		Type:      "user_left",
		Content:   client.Username + " left the room",
		Timestamp: time.Now(),
		UserID:    client.ID,
		Username:  client.Username,
	}

	if data, err := json.Marshal(msg); err == nil {
		h.Broadcast <- &BroadcastMessage{
			RoomID:  client.RoomID,
			Message: data,
		}
	}
}

// ReadPump pumps messages from websocket connection to hub
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error for client %d: %v", c.ID, err)
			}
			break
		}

		// Parse the message
		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Error parsing message from client %d: %v", c.ID, err)
			continue
		}

		// Handle different message types
		switch msg.Type {
		case "connected":
			// Client connection confirmation - just log it
			log.Printf("Client %d confirmed connection to room %d", c.ID, c.RoomID)
			continue

		case "ping":
			// Respond to ping with pong
			pongMsg := Message{
				Type:      "pong",
				Content:   "pong",
				Timestamp: time.Now(),
			}
			if data, err := json.Marshal(pongMsg); err == nil {
				c.Send <- data
			}
			continue

		case "message", "chat":
			msg.UserID = c.ID
			msg.Username = c.Username
			msg.Timestamp = time.Now()

			// Save to DB
			message := models.Message{
				RoomID:   c.RoomID,
				SenderID: c.ID,
				Content:  msg.Content.(string), // type assertion
			}
			if err := config.DB.Create(&message).Error; err != nil {
				log.Printf("Failed to save message: %v", err)
			}

			if data, err := json.Marshal(msg); err == nil {
				c.Hub.Broadcast <- &BroadcastMessage{
					RoomID:  c.RoomID,
					Message: data,
				}
			}

		default:
			// For any other message type, broadcast as-is with metadata
			msg.UserID = c.ID
			msg.Username = c.Username
			msg.Timestamp = time.Now()

			if data, err := json.Marshal(msg); err == nil {
				c.Hub.Broadcast <- &BroadcastMessage{
					RoomID:  c.RoomID,
					Message: data,
				}
			}
		}
	}
}

// WritePump pumps messages from hub to websocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket message
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// GetRoomClients returns all clients in a room
func (h *Hub) GetRoomClients(roomID uint) []*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients := []*Client{}
	if roomClients, ok := h.Rooms[roomID]; ok {
		for client := range roomClients {
			clients = append(clients, client)
		}
	}
	return clients
}

// GetRoomClientCount returns the number of clients in a room
func (h *Hub) GetRoomClientCount(roomID uint) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, ok := h.Rooms[roomID]; ok {
		return len(clients)
	}
	return 0
}
