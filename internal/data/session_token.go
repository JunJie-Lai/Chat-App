package data

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"github.com/JunJie-Lai/Chat-App/internal/validator"
	"github.com/redis/go-redis/v9"
	"time"
)

type SessionTokenInterface interface {
	New(*User, time.Duration) (*SessionToken, error)
	Set(*User, *SessionToken) error
	Delete(string) error
}

type SessionToken struct {
	Plaintext string    `json:"token"`
	Hash      []byte    `json:"-"`
	UserID    int64     `json:"-"`
	Expiry    time.Time `json:"expiry"`
}

type SessionTokenModel struct {
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

func (m SessionTokenModel) New(user *User, ttl time.Duration) (*SessionToken, error) {
	token, err := generateToken(user.ID, ttl)
	if err != nil {
		return nil, err
	}

	if err := m.Set(user, token); err != nil {
		return nil, err
	}

	return token, err
}

func (m SessionTokenModel) Set(user *User, token *SessionToken) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	m.redisDB.HSet(ctx, string(token.Hash), user)
	return m.redisDB.ExpireAt(ctx, string(token.Hash), token.Expiry).Err()
}

func (m SessionTokenModel) Delete(tokenPlaintext string) error {
	tokenHash := sha256.Sum256([]byte(tokenPlaintext))
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := m.redisDB.Del(ctx, string(tokenHash[:])).Result()
	if err != nil {
		return err
	}
	if result == 0 {
		return ErrRecordNotFound
	}

	return nil
}
