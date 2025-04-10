package main

import (
	"net/http"
)

func (app *application) route() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/", app.notFoundResponse)
	path := [6]string{"/v1/user/register", "/v1/user/login", "/v1/user/logout", " /v1/channel", "/v1/channel/{id}", "/{$}"}
	for _, route := range path {
		mux.HandleFunc(route, app.methodNotAllowedResponse)
	}

	mux.HandleFunc("POST /v1/user/register", app.requireNonAuthenticatedUser(app.registerUserHandler))
	mux.HandleFunc("POST /v1/user/login", app.requireNonAuthenticatedUser(app.loginUserHandler))
	mux.HandleFunc("POST /v1/user/logout", app.requireAuthenticatedUser(app.logoutUserHandler))

	mux.HandleFunc("GET /v1/channel", app.requireAuthenticatedUser(app.getAllChannelsHandler))
	mux.HandleFunc("POST /v1/channel", app.requireAuthenticatedUser(app.createChannelHandler))
	mux.HandleFunc("PUT /v1/channel", app.requireAuthenticatedUser(app.editChannelHandler))
	mux.HandleFunc("DELETE v1/channel", app.requireAuthenticatedUser(app.deleteChannelHandler))

	mux.HandleFunc("GET /v1/channel/{id}", app.getChannelHandler)
	mux.HandleFunc("POST /v1/channel/{id}", app.requireAuthenticatedUser(app.superChatHandler))
	mux.HandleFunc("GET /{$}", app.websocketHandler)

	return app.recoverPanic(app.enableCORS(app.rateLimit(app.authenticate(mux))))
}
