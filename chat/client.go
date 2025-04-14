package chat

import (
	"context"
	"github.com/JunJie-Lai/Chat-App/internal/data"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"log/slog"
	"time"
)

type Client struct {
	Conn      *websocket.Conn
	Logger    *slog.Logger
	User      *data.User
	Message   chan *data.Message
	Server    *Server
	RoomID    int64
	CloseSlow func()
}

type Message struct {
	Username  string    `json:"username"`
	Message   []byte    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	SuperChat bool      `json:"super_chat"`
	RoomID    int64     `json:"-"`
}

func (client *Client) ReadMessage() {
	defer func(Conn *websocket.Conn) {
		client.Server.Unregister <- client
		if err := Conn.Close(websocket.StatusNormalClosure, "Disconnected."); err != nil {
			client.Logger.Error(err.Error())
			return
		}
	}(client.Conn)

	for {
		_, message, err := client.Conn.Read(context.Background())
		if err != nil {
			if websocket.CloseStatus(err) != websocket.StatusGoingAway {
				client.Logger.Error("Read error: ", err.Error())
			}
			break
		}
		client.Server.Broadcast <- &data.Message{
			Username:  client.User.Name,
			Message:   message,
			Timestamp: time.Now(),
			RoomID:    client.RoomID,
			SuperChat: false,
		}
	}
}

func (client *Client) WriteMessage() {
	defer func(Conn *websocket.Conn) {
		if err := Conn.Close(websocket.StatusNormalClosure, "Disconnected."); err != nil {
			return
		}
	}(client.Conn)

	for {
		if err := wsjson.Write(context.Background(), client.Conn, <-client.Message); err != nil {
			if websocket.CloseStatus(err) != -1 {
				client.Logger.Error("Write error: ", err.Error())
			}
			break
		}

		for msg := range client.Message {
			if err := wsjson.Write(context.Background(), client.Conn, msg); err != nil {
				if websocket.CloseStatus(err) != -1 {
					client.Logger.Error("Write message history error: ", err.Error())
				}
				break
			}
		}
	}
}
