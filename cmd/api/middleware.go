package main

import (
	"errors"
	"expvar"
	"fmt"
	"github.com/JunJie-Lai/Chat-App/internal/data"
	"github.com/JunJie-Lai/Chat-App/internal/validator"
	"github.com/felixge/httpsnoop"
	"github.com/tomasen/realip"
	"golang.org/x/time/rate"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

func (app *application) recoverPanic(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")
				app.serverErrorResponse(w, r, fmt.Errorf("%s", err))
			}
		}()
		next.ServeHTTP(w, r)
	}
}

func (app *application) rateLimit(next http.Handler) http.HandlerFunc {
	type client struct {
		*rate.Limiter
		lastSeen time.Time
	}

	var (
		mu      sync.Mutex
		clients = make(map[string]*client)
	)

	go func() {
		for {
			time.Sleep(time.Minute)
			mu.Lock()

			for ip, client := range clients {
				if time.Since(client.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}

			mu.Unlock()
		}
	}()

	return func(w http.ResponseWriter, r *http.Request) {
		if os.Getenv("ENVIRONMENT") == "production" {
			ip := realip.FromRequest(r)

			mu.Lock()

			if _, found := clients[ip]; !found {
				clients[ip] = &client{
					Limiter: rate.NewLimiter(rate.Limit(20), 100),
				}
			}

			clients[ip].lastSeen = time.Now()

			if !clients[ip].Limiter.Allow() {
				mu.Unlock()
				app.rateLimitExceededResponse(w, r)
				return
			}

			// Very importantly, unlock the mutex before calling the next handler in the
			// chain. Notice that we DON'T use defer to unlock the mutex, as that would mean
			// that the mutex isn't unlocked until all the handlers downstream of this
			// middleware have also returned.
			mu.Unlock()
		}
		next.ServeHTTP(w, r)
	}
}

func (app *application) authenticate(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Authorization")

		authorizationHeader := r.Header.Get("Authorization")
		if authorizationHeader == "" {
			r = app.contextSetUser(r, data.AnonymousUser)
			next.ServeHTTP(w, r)
			return
		}

		headerParts := strings.Split(authorizationHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		token := headerParts[1]

		v := validator.New()

		if data.ValidateTokenPlaintext(v, token); !v.Valid() {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		user, err := app.models.User.GetFromToken(token)
		if err != nil {
			switch {
			case errors.Is(err, data.ErrRecordNotFound):
				app.invalidAuthenticationTokenResponse(w, r)
			default:
				app.serverErrorResponse(w, r, err)
			}
			return
		}

		r = app.contextSetUser(r, user)
		next.ServeHTTP(w, r)
	}
}

func (app *application) requireAuthenticatedUser(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if user := app.contextGetUser(r); user.IsAnonymous() {
			app.authenticationRequiredResponse(w, r)
			return
		}
		next.ServeHTTP(w, r)
	}
}

func (app *application) requireNonAuthenticatedUser(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if user := app.contextGetUser(r); !user.IsAnonymous() {
			if err := app.writeJSON(w, http.StatusOK, envelope{"user": user}, nil); err != nil {
				app.serverErrorResponse(w, r, err)
				return
			}
			return
		}
		next.ServeHTTP(w, r)
	}
}

func (app *application) enableCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Origin")
		w.Header().Add("Vary", "Access-Control-Request-Method")

		origin := r.Header.Get("Origin")
		trustedOriginsEnv := os.Getenv("TRUSTED_ORIGIN")
		trustedOrigins := strings.Split(trustedOriginsEnv, ",")
		if origin != "" {
			for i := range trustedOrigins {
				if origin == trustedOrigins[i] || trustedOrigins[i] == "*" {
					w.Header().Set("Access-Control-Allow-Origin", origin)

					if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
						w.Header().Set("Access-Control-Allow-Methods", "PUT, PATCH, OPTIONS, DELETE")
						w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")

						w.WriteHeader(http.StatusOK)
						return
					}
					break
				}
			}
		}
		next.ServeHTTP(w, r)
	}
}

func (app *application) metrics(next http.HandlerFunc) http.HandlerFunc {
	totalRequestsReceived := expvar.NewInt("total_requests_received")
	totalResponsesSent := expvar.NewInt("total_responses_sent")
	totalProcessingTimeMicroseconds := expvar.NewInt("total_processing_time_μs")
	totalResponsesSentByStatus := expvar.NewMap("total_responses_sent_by_status")

	return func(w http.ResponseWriter, r *http.Request) {
		totalRequestsReceived.Add(1)

		metrics := httpsnoop.CaptureMetrics(next, w, r)

		totalResponsesSent.Add(1)

		totalProcessingTimeMicroseconds.Add(metrics.Duration.Microseconds())
		totalResponsesSentByStatus.Add(strconv.Itoa(metrics.Code), 1)
	}
}
