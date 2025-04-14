package chat

import (
	"github.com/JunJie-Lai/Chat-App/internal/data"
	"sync"
)

type room struct {
	mu      sync.RWMutex
	clients map[*Client]struct{}
}

type Server struct {
	Register   chan *Client
	Unregister chan *Client
	Broadcast  chan *data.Message

	mu     sync.RWMutex
	rooms  map[int64]*room
	models data.Models
}

func NewServer(models data.Models) *Server {
	return &Server{
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Broadcast:  make(chan *data.Message),
		rooms:      make(map[int64]*room),
		models:     models,
	}
}

func (server *Server) Run() {
	for {
		select {
		case client := <-server.Register:
			// Handle room creation
			server.mu.Lock()
			if _, ok := server.rooms[client.RoomID]; !ok {
				server.rooms[client.RoomID] = &room{
					clients: make(map[*Client]struct{}),
				}
			}
			server.mu.Unlock()

			room := server.rooms[client.RoomID]
			room.mu.Lock()
			// Add client to room
			room.clients[client] = struct{}{}
			room.mu.Unlock()

			// Get message history
			messages, _ := server.models.Message.Get(client.RoomID)
			// Send message history
			for _, message := range messages {
				select {
				case client.Message <- message:
				default:
					server.Unregister <- client
					go client.CloseSlow()
				}
			}
		case client := <-server.Unregister:
			if room, ok := server.rooms[client.RoomID]; ok {
				room.mu.Lock()
				// Remove client from room
				delete(room.clients, client)
				// Remove empty room
				if len(room.clients) == 0 {
					server.mu.Lock()
					delete(server.rooms, client.RoomID)
					server.mu.Unlock()
				}
				close(client.Message)
				room.mu.Unlock()
			}
		case message := <-server.Broadcast:
			if room, ok := server.rooms[message.RoomID]; ok {
				// Add message to message history
				_ = server.models.Message.Set(message)

				room.mu.RLock()
				// Send message to all clients
				for client := range room.clients {
					select {
					case client.Message <- message:
					default:
						server.Unregister <- client
						go client.CloseSlow()
					}
				}
				room.mu.RUnlock()
			}
		}
	}
}
