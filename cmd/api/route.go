package main

import (
	"net/http"
)

func (app *application) route() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /v1/user/register", app.requireNonAuthenticatedUser(app.registerUserHandler))
	mux.HandleFunc("POST /v1/user/login", app.requireNonAuthenticatedUser(app.loginUserHandler))
	mux.HandleFunc("POST /v1/user/logout", app.requireAuthenticatedUser(app.logoutUserHandler))

	mux.HandleFunc("GET /v1/channel", app.requireAuthenticatedUser(app.getAllChannelsHandler))
	mux.HandleFunc("POST /v1/channel", app.requireAuthenticatedUser(app.createChannelHandler))
	mux.HandleFunc("PUT /v1/channel", app.requireAuthenticatedUser(app.editChannelHandler))
	mux.HandleFunc("DELETE v1/channel", app.requireAuthenticatedUser(app.deleteChannelHandler))

	mux.HandleFunc("GET /v1/channel/{id}", app.getChannelHandler)
	mux.HandleFunc("GET /{$}", app.websocketHandler)

	return app.recoverPanic(app.enableCORS(app.rateLimit(app.authenticate(mux))))
}
