package handlers

import (
	"encoding/json"
	"net/http"

	"aura/backend/core-go/internal/domain/entities"
	"aura/backend/core-go/internal/pkg/auth"
)

type AuthHandlers struct {
	Auth  *auth.Service
	Users interface {
		Create(user entities.User) (entities.User, error)
	}
}

type registerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type tokenResponse struct {
	Access  string `json:"access"`
	Refresh string `json:"refresh"`
}

func (a *AuthHandlers) HandleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if req.Email == "" || req.Password == "" || req.Username == "" {
		http.Error(w, "missing fields", http.StatusBadRequest)
		return
	}

	hash, err := authHash(req.Password)
	if err != nil {
		http.Error(w, "failed to hash password", http.StatusInternalServerError)
		return
	}

	u, err := a.Users.Create(entities.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hash,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	t, err := a.Auth.Tokens.Generate(u.ID)
	if err != nil {
		http.Error(w, "failed to issue token", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, tokenResponse{Access: t.Access, Refresh: t.Refresh})
}

func (a *AuthHandlers) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if req.Email == "" || req.Password == "" {
		http.Error(w, "missing fields", http.StatusBadRequest)
		return
	}
	t, err := a.Auth.Authenticate(auth.Credentials{Email: req.Email, Password: req.Password})
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	writeJSON(w, http.StatusOK, tokenResponse{Access: t.Access, Refresh: t.Refresh})
}

func authHash(password string) (string, error) {
	return auth.HashPassword(password)
}

