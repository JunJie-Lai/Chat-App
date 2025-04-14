package main

import (
	"context"
	"errors"
	"github.com/JunJie-Lai/Chat-App/chat"
	"github.com/JunJie-Lai/Chat-App/internal/data"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"net/http"
)

func (app *application) websocketHandler(w http.ResponseWriter, r *http.Request) {
	ws, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	ws.SetReadLimit(768)

	var input struct {
		SessionToken *string `json:"session_token"`
		RoomID       int     `json:"room_id"`
	}
	if err := wsjson.Read(context.Background(), ws, &input); err != nil {
		if err := ws.Close(websocket.StatusPolicyViolation, "Requires room_id"); err != nil {
			return
		}
		return
	}

	if input.SessionToken != nil {
		user, err := app.models.User.GetFromToken(*input.SessionToken)
		if err != nil {
			var tokenErr string
			switch {
			case errors.Is(err, data.ErrRecordNotFound):
				tokenErr = "Invalid session_token"
			default:
				tokenErr = "Server Error"
			}
			if err := ws.Close(websocket.StatusInternalError, tokenErr); err != nil {
				return
			}
			return
		}
		r = app.contextSetUser(r, user)
	}

	channel, err := app.models.Channel.GetExistingChannel(int64(input.RoomID))
	if err != nil {
		var roomErr string
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			roomErr = "Invalid room_id"
		default:
			roomErr = "Server Error"
		}
		if err := ws.Close(websocket.StatusInternalError, roomErr); err != nil {
			return
		}
		return
	}

	client := &chat.Client{
		Conn:    ws,
		Logger:  app.logger,
		User:    app.contextGetUser(r),
		Message: make(chan *data.Message, 128),
		Server:  app.chatServer,
		RoomID:  channel.ID,
		CloseSlow: func() {
			if err := ws.Close(websocket.StatusPolicyViolation, "connection too slow to keep up with messages"); err != nil {
				return
			}
		},
	}
	client.Server.Register <- client

	if !client.User.IsAnonymous() {
		go client.ReadMessage()
	}
	go client.WriteMessage()
}
