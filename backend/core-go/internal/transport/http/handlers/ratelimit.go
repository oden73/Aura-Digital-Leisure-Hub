package handlers

import (
	"net"
	"net/http"
	"strings"

	"aura/backend/core-go/internal/pkg/ratelimit"
)

// RateLimitConfig wires a Limiter into the HTTP chain. SkipPaths exempts
// readiness/health/metrics endpoints so that probes and scrapers cannot
// be limited out of action; SkipFn is an additional escape hatch (useful
// in tests).
type RateLimitConfig struct {
	Limiter   *ratelimit.Limiter
	SkipPaths []string
	SkipFn    func(*http.Request) bool
}

// RateLimit returns a middleware that rejects requests with HTTP 429 when
// the per-identity bucket is empty. Identity is the authenticated user
// id when present, falling back to the client IP — that ordering is
// intentional, otherwise legitimate users behind shared NAT would
// throttle each other.
func RateLimit(cfg RateLimitConfig) func(http.Handler) http.Handler {
	skip := make(map[string]struct{}, len(cfg.SkipPaths))
	for _, p := range cfg.SkipPaths {
		skip[p] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		if cfg.Limiter == nil {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := skip[r.URL.Path]; ok {
				next.ServeHTTP(w, r)
				return
			}
			if cfg.SkipFn != nil && cfg.SkipFn(r) {
				next.ServeHTTP(w, r)
				return
			}

			key := rateLimitKey(r)
			if !cfg.Limiter.Allow(key) {
				w.Header().Set("Retry-After", "1")
				writeError(w, http.StatusTooManyRequests, "rate_limited", "Too many requests")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// rateLimitKey picks the strongest identity available for a request:
// authenticated user id wins over IP. The IP fallback strips the port so
// a client opening multiple connections counts as one bucket.
func rateLimitKey(r *http.Request) string {
	if uid, ok := userIDFromContext(r.Context()); ok {
		return "user:" + uid
	}
	return "ip:" + clientIPOnly(r)
}

func clientIPOnly(r *http.Request) string {
	if v := r.Header.Get("X-Forwarded-For"); v != "" {
		// The header may carry several IPs (proxy chain); the first one
		// is the original client.
		if comma := strings.IndexByte(v, ','); comma >= 0 {
			return strings.TrimSpace(v[:comma])
		}
		return strings.TrimSpace(v)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
