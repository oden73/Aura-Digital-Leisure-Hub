package handlers

import "net/http"

func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				writeError(w, http.StatusInternalServerError, "panic", "Internal error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

