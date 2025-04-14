package main

import (
	"context"
	"database/sql"
	"github.com/JunJie-Lai/Chat-App/chat"
	"github.com/JunJie-Lai/Chat-App/internal/data"
	_ "github.com/joho/godotenv/autoload"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"log/slog"
	"os"
	"sync"
	"time"
)

const version = "1.0.0"

type application struct {
	wg         sync.WaitGroup
	logger     *slog.Logger
	chatServer *chat.Server
	models     data.Models
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	db, err := openDB()
	if err != nil {
		logger.Error(err.Error())
	}

	defer func(db *sql.DB) {
		if err := db.Close(); err != nil {
			logger.Error(err.Error())
		}
	}(db)

	redisDB := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	app := &application{
		logger:     logger,
		models:     data.NewModels(db, redisDB),
		chatServer: chat.NewServer(data.NewModels(db, redisDB)),
	}

	go app.chatServer.Run()

	if err := app.serve(); err != nil {
		logger.Error(err.Error())
	}
	os.Exit(1)
}

func openDB() (*sql.DB, error) {
	db, err := sql.Open("postgres", os.Getenv("DB"))
	if err != nil {
		return nil, err
	}

	//db.SetMaxOpenConns()
	//db.SetMaxIdleConns()
	//db.SetConnMaxIdleTime()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}
	return db, nil
}
