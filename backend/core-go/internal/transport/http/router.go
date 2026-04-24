package http

import (
	"net/http"

	"aura/backend/core-go/internal/transport/http/handlers"
)

func NewRouter() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", handlers.Health)
	mux.HandleFunc("POST /v1/recommendations", handlers.GetRecommendations)
	mux.HandleFunc("GET /v1/search", handlers.SearchContent)
	mux.HandleFunc("PUT /v1/interactions", handlers.UpdateInteraction)
	mux.HandleFunc("POST /v1/sync/external", handlers.SyncExternalContent)
	mux.HandleFunc("GET /v1/profile", handlers.GetProfile)

	return mux
}
