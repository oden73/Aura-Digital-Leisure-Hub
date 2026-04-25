package http

import (
	"net/http"

	"aura/backend/core-go/internal/transport/http/handlers"
)

// NewRouter builds the HTTP router with all public endpoints.
func NewRouter(h *handlers.Handlers) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", handlers.Health)
	mux.HandleFunc("POST /v1/auth/register", h.Auth.HandleRegister)
	mux.HandleFunc("POST /v1/auth/login", h.Auth.HandleLogin)
	mux.HandleFunc("POST /v1/recommendations", handlers.Auth(h.Auth.Auth.Tokens, h.HandleGetRecommendations))
	mux.HandleFunc("GET /v1/search", h.HandleSearch)
	mux.HandleFunc("PUT /v1/interactions", handlers.Auth(h.Auth.Auth.Tokens, h.HandleUpdateInteraction))
	mux.HandleFunc("POST /v1/sync/external", handlers.Auth(h.Auth.Auth.Tokens, h.HandleSyncExternal))
	mux.HandleFunc("GET /v1/profile", handlers.Auth(h.Auth.Auth.Tokens, h.HandleGetProfile))

	return mux
}
