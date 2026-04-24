package http

import (
	"net/http"

	"aura/backend/core-go/internal/transport/http/handlers"
)

// NewRouter builds the HTTP router with all public endpoints.
func NewRouter(h *handlers.Handlers) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", handlers.Health)
	mux.HandleFunc("POST /v1/recommendations", h.HandleGetRecommendations)
	mux.HandleFunc("GET /v1/search", h.HandleSearch)
	mux.HandleFunc("PUT /v1/interactions", h.HandleUpdateInteraction)
	mux.HandleFunc("POST /v1/sync/external", h.HandleSyncExternal)
	mux.HandleFunc("GET /v1/profile", h.HandleGetProfile)

	return mux
}
