package data

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base32"
	"github.com/JunJie-Lai/Chat-App/internal/validator"
	"github.com/redis/go-redis/v9"
	"time"
)

type SessionTokenInterface interface {
	New(int64, time.Duration) (*SessionToken, error)
	Insert(*SessionToken) error
	DeleteAllForUser(int64, string) error
}

type SessionToken struct {
	Plaintext string    `json:"token"`
	Hash      []byte    `json:"-"`
	UserID    int64     `json:"-"`
	Expiry    time.Time `json:"expiry"`
}

type SessionTokenModel struct {
	db      *sql.DB
	redisDB *redis.Client
}

func generateToken(userID int64, ttl time.Duration) (*SessionToken, error) {
	// ttl: time-to-live
	token := &SessionToken{
		UserID: userID,
		Expiry: time.Now().Add(ttl),
	}

	randomBytes := make([]byte, 16)

	if _, err := rand.Read(randomBytes); err != nil {
		return nil, err
	}

	token.Plaintext = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)
	hash := sha256.Sum256([]byte(token.Plaintext))
	token.Hash = hash[:]

	return token, nil
}

func ValidateTokenPlaintext(v *validator.Validator, tokenPlaintext string) {
	v.Check(tokenPlaintext != "", "token", "must be provided")
	v.Check(len(tokenPlaintext) == 26, "token", "must be 26 bytes long")
}

func (m SessionTokenModel) New(userID int64, ttl time.Duration) (*SessionToken, error) {
	token, err := generateToken(userID, ttl)
	if err != nil {
		return nil, err
	}

	err = m.Insert(token)
	return token, err
}

func (m SessionTokenModel) Insert(token *SessionToken) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.db.ExecContext(ctx, "INSERT INTO session_token VALUES ($1, $2, $3)", token.Hash, token.UserID, token.Expiry)
	return err
}

func (m SessionTokenModel) DeleteAllForUser(userID int64, tokenPlaintext string) error {
	tokenHash := sha256.Sum256([]byte(tokenPlaintext))
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.db.ExecContext(ctx, "DELETE FROM session_token WHERE user_id = $1 AND (expiry < CURRENT_TIMESTAMP OR hash = $2)", userID, tokenHash[:])
	return err
}
