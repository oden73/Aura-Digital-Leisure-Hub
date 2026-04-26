package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"
)

// ctxRequestID is the context key under which a per-request id is stored.
const ctxRequestID ctxKey = "request_id"

// RequestIDFromContext returns the request id assigned by RequestID
// middleware, or an empty string if the middleware was not in the chain.
func RequestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(ctxRequestID).(string); ok {
		return v
	}
	return ""
}

// RequestID assigns a unique id to every incoming request, exposes it on
// the X-Request-ID response header, and stores it in the context. Clients
// may pre-set the header to propagate ids across services; we trust those
// only when they look like a 128-bit hex string to avoid log injection.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if !isValidRequestID(id) {
			id = newRequestID()
		}
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), ctxRequestID, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// statusRecorder wraps a ResponseWriter to capture the status code we
// eventually write to the client. http.ResponseWriter does not expose
// the status, so we have to track it manually.
type statusRecorder struct {
	http.ResponseWriter
	status int
	size   int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	n, err := s.ResponseWriter.Write(b)
	s.size += n
	return n, err
}

// AccessLog logs one structured line per request with method, path,
// status, duration, request id and (when available) authenticated user
// id. Logs at error for 5xx, warn for 4xx, info otherwise.
func AccessLog(logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w}
			next.ServeHTTP(rec, r)

			status := rec.status
			if status == 0 {
				status = http.StatusOK
			}

			attrs := []any{
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", status),
				slog.Int64("duration_ms", time.Since(start).Milliseconds()),
				slog.Int("response_bytes", rec.size),
				slog.String("remote", clientIP(r)),
				slog.String("request_id", RequestIDFromContext(r.Context())),
			}
			if uid, ok := userIDFromContext(r.Context()); ok {
				attrs = append(attrs, slog.String("user_id", uid))
			}

			level := slog.LevelInfo
			switch {
			case status >= 500:
				level = slog.LevelError
			case status >= 400:
				level = slog.LevelWarn
			}
			logger.LogAttrs(r.Context(), level, "http_request", attrsAsAttr(attrs)...)
		})
	}
}

// attrsAsAttr converts our []any slog field list into []slog.Attr because
// LogAttrs is the typed variant; it skips anything that isn't an Attr.
func attrsAsAttr(in []any) []slog.Attr {
	out := make([]slog.Attr, 0, len(in))
	for _, v := range in {
		if a, ok := v.(slog.Attr); ok {
			out = append(out, a)
		}
	}
	return out
}

// clientIP picks the first plausible client address for logs. We honour
// X-Forwarded-For for deployments behind a trusted proxy and fall back
// to RemoteAddr otherwise.
func clientIP(r *http.Request) string {
	if v := r.Header.Get("X-Forwarded-For"); v != "" {
		return v
	}
	return r.RemoteAddr
}

// isValidRequestID accepts only hex strings between 8 and 64 chars. This
// is deliberately strict so an attacker cannot stuff arbitrary bytes into
// log lines via the X-Request-ID header.
func isValidRequestID(id string) bool {
	if len(id) < 8 || len(id) > 64 {
		return false
	}
	for _, r := range id {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'f':
		case r >= 'A' && r <= 'F':
		case r == '-':
		default:
			return false
		}
	}
	return true
}

func newRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand failing is extraordinary; fall back to a static
		// marker rather than panicking and bringing down the request.
		return "0000000000000000"
	}
	return hex.EncodeToString(b[:])
}
