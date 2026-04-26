package handlers

import (
	"log/slog"
	"net/http"
	"runtime/debug"
)

// Recover catches panics from downstream handlers, returns a 500 to the
// client and logs the panic value plus a stack trace. We log via slog so
// the entry lands in the same structured stream as access logs and is
// correlated with a request id when RequestID middleware is upstream.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.ErrorContext(r.Context(), "panic_recovered",
					slog.Any("panic", rec),
					slog.String("path", r.URL.Path),
					slog.String("method", r.Method),
					slog.String("request_id", RequestIDFromContext(r.Context())),
					slog.String("stack", string(debug.Stack())),
				)
				writeError(w, http.StatusInternalServerError, "panic", "Internal error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

