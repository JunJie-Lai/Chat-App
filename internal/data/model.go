package data

import (
	"database/sql"
	"errors"
)

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
)

type Models struct {
	User         UserInterface
	SessionToken SessionTokenInterface
	Channel      ChannelInterface
}

func NewModels(db *sql.DB) Models {
	return Models{
		User:         &UserModel{db},
		SessionToken: &SessionTokenModel{db},
		Channel:      &ChannelModel{db},
	}
}
