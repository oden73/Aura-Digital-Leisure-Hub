package app

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"

	"aura/backend/core-go/internal/domain/entities"
)

var errNotFound = errors.New("not found")

type memoryUserRepo struct {
	mu       sync.RWMutex
	byID     map[string]entities.User
	byEmail  map[string]string
}

func newMemoryUserRepo() *memoryUserRepo {
	return &memoryUserRepo{
		byID:    map[string]entities.User{},
		byEmail: map[string]string{},
	}
}

func (r *memoryUserRepo) Create(u entities.User) (entities.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	emailKey := strings.ToLower(strings.TrimSpace(u.Email))
	if emailKey == "" || u.Username == "" || u.PasswordHash == "" {
		return entities.User{}, errors.New("invalid user")
	}
	if _, ok := r.byEmail[emailKey]; ok {
		return entities.User{}, errors.New("email already exists")
	}
	u.ID = newID()
	u.Email = emailKey
	u.CreatedAt = time.Now()
	r.byID[u.ID] = u
	r.byEmail[emailKey] = u.ID
	return u, nil
}

func (r *memoryUserRepo) GetByID(userID string) (entities.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	u, ok := r.byID[userID]
	if !ok {
		return entities.User{}, errNotFound
	}
	return u, nil
}

func (r *memoryUserRepo) GetByEmail(email string) (entities.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id, ok := r.byEmail[strings.ToLower(strings.TrimSpace(email))]
	if !ok {
		return entities.User{}, errNotFound
	}
	u, ok := r.byID[id]
	if !ok {
		return entities.User{}, errNotFound
	}
	return u, nil
}

func (r *memoryUserRepo) GetProfile(userID string) (entities.UserProfile, error) {
	_, err := r.GetByID(userID)
	if err != nil {
		return entities.UserProfile{}, err
	}
	return entities.UserProfile{UserID: userID}, nil
}

func (r *memoryUserRepo) LinkExternalAccount(_ entities.ExternalAccount) error { return nil }

func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

