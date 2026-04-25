package handlers

import (
	"context"
	"net/http"
	"strings"

	"aura/backend/core-go/internal/pkg/auth"
)

type ctxKey string

const ctxUserID ctxKey = "user_id"

func userIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxUserID).(string)
	return v, ok && v != ""
}

func authMiddleware(tokens auth.TokenManager, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}
		uid, err := tokens.Validate(strings.TrimSpace(strings.TrimPrefix(h, "Bearer ")))
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), ctxUserID, uid)
		next(w, r.WithContext(ctx))
	}
}

// Auth wraps a handler with bearer-token auth.
func Auth(tokens auth.TokenManager, next http.HandlerFunc) http.HandlerFunc {
	return authMiddleware(tokens, next)
}

