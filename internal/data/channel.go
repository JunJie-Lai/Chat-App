package data

import (
	"context"
	"database/sql"
	"errors"
	"github.com/JunJie-Lai/Chat-App/internal/validator"
	"time"
)

var ErrDuplicateChannel = errors.New("duplicate channel")

type ChannelInterface interface {
	GetAllChannel(int64) ([]*Channel, error)
	GetChannel(int64, int64) (*Channel, error)
	CreateChannel(int64, *Channel) error
	UpdateChannelName(int64, *Channel) error
	DeleteChannel(int64, int64) error
	GetExistingChannel(int64) (*Channel, error)
}

type Channel struct {
	ID        int64     `json:"channel_id"`
	Name      string    `json:"channel_name"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

type ChannelModel struct {
	db *sql.DB
}

func (m *ChannelModel) GetAllChannel(userID int64) ([]*Channel, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := m.db.QueryContext(ctx, "SELECT id, name, created_at FROM channel WHERE user_id = $1 ORDER BY created_at DESC", userID)
	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) {
		if err := rows.Close(); err != nil {
			return
		}
	}(rows)

	var channels []*Channel
	for rows.Next() {
		var channel Channel
		if err := rows.Scan(&channel.ID, &channel.Name, &channel.CreatedAt); err != nil {
			return nil, err
		}
		channels = append(channels, &channel)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return channels, nil
}

func (m *ChannelModel) GetChannel(userID, channelID int64) (*Channel, error) {
	if channelID < 1 {
		return nil, ErrRecordNotFound
	}

	var channel Channel

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := m.db.QueryRowContext(ctx, "SELECT id, name, created_at FROM channel WHERE user_id = $1 AND id = $2", userID, channelID).
		Scan(&channel.ID, &channel.Name, &channel.CreatedAt); err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &channel, nil
}

func (m *ChannelModel) CreateChannel(userID int64, channel *Channel) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := m.db.QueryRowContext(ctx, "INSERT INTO channel (user_id, name) VALUES ($1, $2) RETURNING id, name, created_at", userID, channel.Name).
		Scan(&channel.ID, &channel.Name, &channel.CreatedAt); err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "channel_user_id_name_key"`:
			return ErrDuplicateChannel
		default:
			return err
		}
	}

	return nil
}

func (m *ChannelModel) UpdateChannelName(userID int64, channel *Channel) error {
	if channel.ID < 1 {
		return ErrRecordNotFound
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := m.db.ExecContext(ctx, "UPDATE channel SET name = $1 WHERE id = $2 AND user_id = $3", channel.Name, channel.ID, userID)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "channel_user_id_name_key"`:
			return ErrDuplicateChannel
		default:
			return err
		}
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrRecordNotFound
	}

	return nil
}

func (m *ChannelModel) DeleteChannel(userID, channelID int64) error {
	if channelID < 1 {
		return ErrRecordNotFound
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := m.db.ExecContext(ctx, "DELETE FROM channel WHERE user_id = $1 AND id = $2", userID, channelID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrRecordNotFound
	}

	return nil
}

func (m *ChannelModel) GetExistingChannel(channelID int64) (*Channel, error) {
	if channelID < 1 {
		return nil, ErrRecordNotFound
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var channel Channel
	if err := m.db.QueryRowContext(ctx, "SELECT id, name FROM channel WHERE id = $1", channelID).
		Scan(&channel.ID, &channel.Name); err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return &channel, nil
}

func ValidateChannel(v *validator.Validator, channel *Channel) {
	v.Check(channel.Name != "", "channel_name", "must be provided")
	v.Check(len(channel.Name) <= 32, "channel_name", "must not be more than 32 characters")
}
