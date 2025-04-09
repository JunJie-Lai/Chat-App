package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func (app *application) serve() error {
	server := http.Server{
		Addr:         os.Getenv("PORT"),
		Handler:      app.route(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  time.Second * 10,
		WriteTimeout: time.Second * 30,
		ErrorLog:     slog.NewLogLogger(app.logger.Handler(), slog.LevelError),
	}

	shutdownError := make(chan error)

	go func() {
		quit := make(chan os.Signal, 1)

		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		s := <-quit

		app.logger.Info("shutting down server", "signal", s.String())

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		app.wg.Wait()
		shutdownError <- server.Shutdown(ctx)
	}()

	app.logger.Info("starting server", "addr", server.Addr, "env", os.Getenv("ENVIRONMENT"))

	err := server.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	if err := <-shutdownError; err != nil {
		return err
	}

	app.logger.Info("stopped server", "addr", server.Addr)
	return nil
}
