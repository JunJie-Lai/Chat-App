package main

import (
	"errors"
	"github.com/JunJie-Lai/Chat-App/internal/data"
	"github.com/JunJie-Lai/Chat-App/internal/validator"
	"net/http"
	"time"
)

func (app *application) getAllChannelsHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	channels, err := app.models.Channel.GetAllChannel(user.ID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	if err := app.writeJSON(w, http.StatusOK, envelope{"channels": channels}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) createChannelHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	var input struct {
		Name string `json:"channel_name"`
	}

	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	channel := &data.Channel{
		Name: input.Name,
	}

	v := validator.New()
	if data.ValidateChannel(v, channel); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	if err := app.models.Channel.CreateChannel(user.ID, channel); err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateChannel):
			v.AddError("channel", "the user with this channel already exists")
			app.failedValidationResponse(w, r, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	if err := app.writeJSON(w, http.StatusCreated, envelope{"channel": channel}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) editChannelHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	var input struct {
		ID   int64  `json:"channel_id"`
		Name string `json:"channel_name"`
	}

	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	channel, err := app.models.Channel.GetChannel(user.ID, input.ID)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	channel.Name = input.Name

	v := validator.New()
	if data.ValidateChannel(v, channel); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	if err := app.models.Channel.UpdateChannelName(user.ID, channel); err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateChannel):
			v.AddError("channel", "the user with this channel name already exists")
			app.failedValidationResponse(w, r, v.Errors)
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	if err := app.writeJSON(w, http.StatusOK, envelope{"channel": channel}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) deleteChannelHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	var input struct {
		ID   int64  `json:"channel_id"`
		Name string `json:"channel_name"`
	}

	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	channel, err := app.models.Channel.GetChannel(user.ID, input.ID)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	v := validator.New()
	if data.ValidateChannel(v, channel); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	if err := app.models.Channel.DeleteChannel(user.ID, channel.ID); err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	if err := app.writeJSON(w, http.StatusOK, envelope{"message": "channel deleted"}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) getChannelHandler(w http.ResponseWriter, r *http.Request) {
	roomID, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	channel, err := app.models.Channel.GetExistingChannel(roomID)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	var websocketToken *data.SessionToken
	if user := app.contextGetUser(r); !user.IsAnonymous() {
		token, err := app.models.SessionToken.New(user.ID, 3*time.Second)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}
		websocketToken = token
	}

	if err := app.writeJSON(w, http.StatusOK, envelope{"channel": channel, "websocket_token": websocketToken}, nil); err != nil {
	}
}
