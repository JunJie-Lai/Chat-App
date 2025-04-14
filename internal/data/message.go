package data

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/redis/go-redis/v9"
	"strconv"
	"time"
)

type MessageInterface interface {
	Set(*Message) error
	Get(int64) ([]*Message, error)
}

type Message struct {
	Username  string    `json:"username"`
	Message   []byte    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	SuperChat bool      `json:"super_chat"`
	RoomID    int64     `json:"-"`
}

type MessageModel struct {
	db      *sql.DB
	redisDB *redis.Client
}

func (m *MessageModel) Set(message *Message) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	key := "room:" + strconv.FormatInt(message.RoomID, 10) + ":messages"
	msg, err := json.Marshal(message)
	if err != nil {
		return err
	}
	m.redisDB.RPush(ctx, key, msg)
	m.redisDB.ExpireXX(context.Background(), key, 24*time.Hour)
	return nil
}

func (m *MessageModel) Get(roomID int64) ([]*Message, error) {
	key := "room:" + strconv.FormatInt(roomID, 10) + ":messages"
	result, err := m.redisDB.LRange(context.Background(), key, 0, -1).Result()
	if err != nil {
		return nil, err
	}

	var messages []*Message
	for _, message := range result {
		var msg Message
		if err := json.Unmarshal([]byte(message), &msg); err != nil {
			return nil, err
		}
		messages = append(messages, &msg)
	}
	return messages, nil
}
