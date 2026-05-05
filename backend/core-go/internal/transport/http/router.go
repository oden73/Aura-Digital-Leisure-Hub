package http

import (
	"log/slog"
	"net/http"

	"aura/backend/core-go/internal/transport/http/handlers"
)

// RouterOptions tunes the cross-cutting middleware chain wrapped around
// the mux. Every field is optional: zero values disable the relevant
// middleware so this package stays useful in tests.
//
// More observability hooks (rate limiting) are added incrementally in
// subsequent changes; this struct grows with them.
type RouterOptions struct {
	Logger *slog.Logger
	// HealthCheck is the content-aware /health endpoint (DB, AI engine,
	// …). Falls back to a static 200 when nil. /livez is always wired
	// to the static handler so liveness probes never depend on data
	// stores.
	HealthCheck     http.HandlerFunc
	MetricsHandler  http.Handler
	MetricsRecorder handlers.MetricsRecorder
	CORS            *handlers.CORSConfig
	RateLimit       *handlers.RateLimitConfig
}

// NewRouter builds the HTTP router with all public endpoints. The
// returned http.Handler already includes the middleware chain
// (request-id → access-log → metrics → cors → rate-limit → recover).
// Order matters: request-id must be first so every other middleware
// can correlate by id; recover is innermost so panics never escape the
// access log without a 500 status.
func NewRouter(h *handlers.Handlers, opts RouterOptions) http.Handler {
	mux := http.NewServeMux()

	healthFn := handlers.Health
	if opts.HealthCheck != nil {
		healthFn = opts.HealthCheck
	}
	mux.HandleFunc("GET /health", healthFn)
	// /livez is unconditional: only for liveness probes. Readiness probes
	// should target /health, which actually exercises dependencies.
	mux.HandleFunc("GET /livez", handlers.Health)
	if opts.MetricsHandler != nil {
		mux.Handle("GET /metrics", opts.MetricsHandler)
	}
	mux.HandleFunc("POST /v1/auth/register", h.Auth.HandleRegister)
	mux.HandleFunc("POST /v1/auth/login", h.Auth.HandleLogin)
	mux.HandleFunc("POST /v1/auth/refresh", h.Auth.HandleRefresh)
	mux.HandleFunc("POST /v1/recommendations", handlers.Auth(h.Auth.Auth.Tokens, h.HandleGetRecommendations))
	mux.HandleFunc("GET /v1/search", h.HandleSearch)
	mux.HandleFunc("GET /v1/content/{id}", h.HandleGetContent)
	mux.HandleFunc("POST /v1/content", handlers.Auth(h.Auth.Auth.Tokens, h.HandleUpsertContent))
	mux.HandleFunc("PUT /v1/interactions", handlers.Auth(h.Auth.Auth.Tokens, h.HandleUpdateInteraction))
	mux.HandleFunc("GET /v1/library", handlers.Auth(h.Auth.Auth.Tokens, h.HandleGetLibrary))
	mux.HandleFunc("GET /v1/library/items", handlers.Auth(h.Auth.Auth.Tokens, h.HandleGetLibraryItems))
	mux.HandleFunc("POST /v1/sync/external", handlers.Auth(h.Auth.Auth.Tokens, h.HandleSyncExternal))
	mux.HandleFunc("POST /v1/external-accounts", handlers.Auth(h.Auth.Auth.Tokens, h.HandleLinkExternalAccount))
	mux.HandleFunc("GET /v1/profile", handlers.Auth(h.Auth.Auth.Tokens, h.HandleGetProfile))
	mux.HandleFunc("POST /v1/assistant", handlers.Auth(h.Auth.Auth.Tokens, h.HandleAssistantChat))

	var chain http.Handler = mux
	chain = handlers.Recover(chain)
	if opts.RateLimit != nil {
		chain = handlers.RateLimit(*opts.RateLimit)(chain)
	}
	if opts.CORS != nil {
		chain = handlers.CORS(*opts.CORS)(chain)
	}
	if opts.MetricsRecorder != nil {
		chain = handlers.Metrics(opts.MetricsRecorder)(chain)
	}
	chain = handlers.AccessLog(opts.Logger)(chain)
	chain = handlers.RequestID(chain)
	return chain
}
