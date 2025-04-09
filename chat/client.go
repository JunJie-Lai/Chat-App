package chat

import (
	"context"
	"fmt"
	"github.com/JunJie-Lai/Chat-App/internal/data"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"log"
	"time"
)

type Client struct {
	Conn      *websocket.Conn
	User      *data.User
	Message   chan *Message
	Server    *Server
	RoomID    int64
	CloseSlow func()
}

type Message struct {
	Username  string    `json:"username"`
	Message   []byte    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	RoomID    int64     `json:"-"`
}

func (client *Client) ReadMessage() {
	defer func(Conn *websocket.Conn) {
		fmt.Println("READ CLOSE")
		client.Server.Unregister <- client
		if err := Conn.Close(websocket.StatusNormalClosure, client.User.Name+" disconnected."); err != nil {
			return
		}
	}(client.Conn)

	for {
		_, message, err := client.Conn.Read(context.Background())
		if err != nil {
			if websocket.CloseStatus(err) != websocket.StatusGoingAway {
				log.Println(3, err)
			}
			break
		}
		client.Server.Broadcast <- &Message{
			Username:  client.User.Name,
			Message:   message,
			Timestamp: time.Now(),
			RoomID:    client.RoomID,
		}
	}
}

func (client *Client) WriteMessage() {
	defer func(client *Client) {
		fmt.Println("WRITE CLOSE")
		if err := client.Conn.Close(websocket.StatusNormalClosure, client.User.Name+" disconnected."); err != nil {
			return
		}
	}(client)

	for {
		if err := wsjson.Write(context.Background(), client.Conn, <-client.Message); err != nil {
			if websocket.CloseStatus(err) != -1 {
				log.Println(4, err)
			}
			break
		}

		for msg := range client.Message {
			if err := wsjson.Write(context.Background(), client.Conn, msg); err != nil {
				log.Println(4.5, err)
				break
			}
		}
	}
}
