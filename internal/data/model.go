package data

import (
	"database/sql"
	"errors"
	"github.com/redis/go-redis/v9"
)

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
)

type Models struct {
	User         UserInterface
	SessionToken SessionTokenInterface
	Channel      ChannelInterface
	Message      MessageInterface
}

func NewModels(db *sql.DB, redisDB *redis.Client) Models {
	return Models{
		User:         &UserModel{db},
		SessionToken: &SessionTokenModel{db, redisDB},
		Channel:      &ChannelModel{db},
		Message:      &MessageModel{db, redisDB},
	}
}
