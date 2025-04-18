package data

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"github.com/JunJie-Lai/Chat-App/internal/validator"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"time"
)

var (
	ErrDuplicateEmail = errors.New("duplicate email")
	AnonymousUser     = &User{}
)

type UserInterface interface {
	Insert(*User) error
	GetByEmail(string) (*User, error)
	Update(string, *User) error
	GetFromToken(string) (*User, error)
}

type User struct {
	ID       int64    `json:"user_id" redis:"user_id"`
	Name     string   `json:"user_name" redis:"user_name"`
	Email    string   `json:"email" redis:"email"`
	Password password `json:"-" redis:"-"`
}

type password struct {
	plaintext *string
	hash      []byte
}

type UserModel struct {
	db      *sql.DB
	redisDB *redis.Client
}

func (m UserModel) Insert(user *User) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := m.db.QueryRowContext(ctx,
		"INSERT INTO users (name, email, password_hash) VALUES ($1, $2, $3) RETURNING id",
		&user.Name, &user.Email, &user.Password.hash).Scan(&user.ID); err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return ErrDuplicateEmail
		default:
			return err
		}
	}
	return nil
}

func (m UserModel) GetByEmail(email string) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var user User

	if err := m.db.QueryRowContext(ctx, "SELECT * FROM users WHERE email = $1", email).
		Scan(&user.ID, &user.Name, &user.Email, &user.Password.hash); err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return &user, nil
}

func (m UserModel) Update(tokenPlaintext string, user *User) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := m.db.QueryRowContext(ctx,
		"UPDATE users SET name = $1, email = $2, password_hash = $3 WHERE id = $4 RETURNING id",
		user.Name, user.Email, user.Password.hash, user.ID).Scan(&user.ID); err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return ErrDuplicateEmail
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		default:
			return err
		}
	}

	tokenHash := sha256.Sum256([]byte(tokenPlaintext))
	m.redisDB.HSet(ctx, string(tokenHash[:]), user)
	return nil
}

func (m UserModel) GetFromToken(tokenPlaintext string) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	tokenHash := sha256.Sum256([]byte(tokenPlaintext))

	var user User
	exist, err := m.redisDB.Exists(ctx, string(tokenHash[:])).Result()
	if err != nil {
		return nil, err
	}
	if exist == 0 {
		return nil, ErrRecordNotFound
	}

	if err := m.redisDB.HGetAll(ctx, string(tokenHash[:])).Scan(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (u *User) IsAnonymous() bool {
	return u == AnonymousUser
}

func (p *password) Set(plaintextPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintextPassword), 12)
	if err != nil {
		return err
	}

	p.plaintext = &plaintextPassword
	p.hash = hash
	return nil
}

func (p *password) Matches(plaintextPassword string) (bool, error) {
	if err := bcrypt.CompareHashAndPassword(p.hash, []byte(plaintextPassword)); err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil
		default:
			return false, err
		}
	}
	return true, nil
}

func ValidateEmail(v *validator.Validator, email string) {
	v.Check(email != "", "email", "must be provided")
	v.Check(validator.Matches(email, validator.EmailRX), "email", "must be valid email address")
}

func ValidatePasswordPlaintext(v *validator.Validator, password string) {
	v.Check(password != "", "password", "must be provided")
	v.Check(len(password) >= 8, "password", "must be at least 8 bytes long")
	v.Check(len(password) <= 72, "password", "must not be more than 72 bytes long")
}

func ValidateUser(v *validator.Validator, user *User) {
	v.Check(user.Name != "", "name", "must be provided")
	v.Check(len(user.Name) <= 500, "name", "must not be more than 500 bytes long")

	ValidateEmail(v, user.Email)

	if user.Password.plaintext != nil {
		ValidatePasswordPlaintext(v, *user.Password.plaintext)
	}
	if user.Password.hash == nil {
		panic("missing password hash for user")
	}
}
