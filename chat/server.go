package chat

import (
	"context"
	"github.com/redis/go-redis/v9"
	"strconv"
	"sync"
)

type room struct {
	mu       sync.RWMutex
	clients  map[*Client]struct{}
	messages []*Message
}

type Server struct {
	mu         sync.RWMutex
	redisDB    *redis.Client
	rooms      map[int64]*room
	Register   chan *Client
	Unregister chan *Client
	Broadcast  chan *Message
}

func NewServer(redisDB *redis.Client) *Server {
	return &Server{
		redisDB:    redisDB,
		rooms:      make(map[int64]*room),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Broadcast:  make(chan *Message),
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

			room.mu.RLock()
			// Send message history
			for _, message := range room.messages {
				select {
				case client.Message <- message:
				default:
					server.Unregister <- client
					go client.CloseSlow()
				}
			}
			room.mu.RUnlock()
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
				//room.mu.Lock()
				//// Add message to history
				//room.messages = append(room.messages, message)
				//room.mu.Unlock()

				roomMessagesKey := "room:" + strconv.FormatInt(message.RoomID, 10) + ":messages"
				server.redisDB.RPush(context.Background(), roomMessagesKey, message)

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
