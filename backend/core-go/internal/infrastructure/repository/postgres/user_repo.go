package postgres

import (
	"context"
	"errors"
	"strings"
	"time"

	"aura/backend/core-go/internal/domain/entities"
	dbpostgres "aura/backend/core-go/internal/infrastructure/db/postgres"

	"github.com/jackc/pgx/v5"
)

var ErrNotFound = errors.New("not found")

type UserRepo struct {
	DB *dbpostgres.Pool
}

func NewUserRepo(db *dbpostgres.Pool) *UserRepo { return &UserRepo{DB: db} }

func (r *UserRepo) Create(user entities.User) (entities.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	email := strings.ToLower(strings.TrimSpace(user.Email))
	username := strings.TrimSpace(user.Username)
	if email == "" || username == "" || user.PasswordHash == "" {
		return entities.User{}, errors.New("invalid user")
	}

	row := r.DB.QueryRow(ctx, `
		INSERT INTO users (username, email, password_hash)
		VALUES ($1, $2, $3)
		RETURNING user_id, created_at
	`, username, email, user.PasswordHash)

	var id string
	var createdAt time.Time
	if err := row.Scan(&id, &createdAt); err != nil {
		return entities.User{}, err
	}
	user.ID = id
	user.Email = email
	user.Username = username
	user.CreatedAt = createdAt
	return user, nil
}

func (r *UserRepo) GetByID(userID string) (entities.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var u entities.User
	err := r.DB.QueryRow(ctx, `
		SELECT user_id, username, email, password_hash, created_at
		FROM users
		WHERE user_id = $1
	`, userID).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return entities.User{}, ErrNotFound
	}
	return u, err
}

func (r *UserRepo) GetByEmail(email string) (entities.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var u entities.User
	err := r.DB.QueryRow(ctx, `
		SELECT user_id, username, email, password_hash, created_at
		FROM users
		WHERE email = $1
	`, strings.ToLower(strings.TrimSpace(email))).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return entities.User{}, ErrNotFound
	}
	return u, err
}

func (r *UserRepo) GetProfile(userID string) (entities.UserProfile, error) {
	// Placeholder until profile aggregation is implemented.
	_, _ = userID, r
	return entities.UserProfile{UserID: userID}, nil
}

func (r *UserRepo) LinkExternalAccount(account entities.ExternalAccount) error {
	// TODO: implement when external sync is wired.
	_ = account
	return nil
}

