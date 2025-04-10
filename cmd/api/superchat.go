package main

import (
	"errors"
	"github.com/JunJie-Lai/Chat-App/chat"
	"github.com/JunJie-Lai/Chat-App/internal/data"
	"net/http"
	"time"
)

func (app *application) superChatHandler(w http.ResponseWriter, r *http.Request) {
	channelID, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	channel, err := app.models.Channel.GetExistingChannel(channelID)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	var input struct {
		Message string `json:"message"`
	}
	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	app.chatServer.Broadcast <- &chat.Message{
		Username:  app.contextGetUser(r).Name,
		Message:   []byte(input.Message),
		Timestamp: time.Now(),
		RoomID:    channel.ID,
		SuperChat: true,
	}

	if err := app.writeJSON(w, http.StatusAccepted, envelope{"message": "message sent"}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
