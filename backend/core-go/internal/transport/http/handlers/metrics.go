package handlers

import (
	"net/http"
	"time"
)

// MetricsRecorder is the slice of metrics.Recorder this transport layer
// actually depends on. Declaring a local interface keeps the transport
// package decoupled from the concrete metrics implementation, which
// matters for tests that mock the recorder.
type MetricsRecorder interface {
	ObserveHTTP(method, route string, status int, duration time.Duration)
}

// Metrics is a middleware that records request count + latency using the
// supplied Recorder. The route label is the matched mux pattern when
// present (Go 1.22+ stores it on the request) and falls back to the raw
// URL path so we never lose data; however, callers should ensure routes
// are mux-bound to keep cardinality bounded.
func Metrics(rec MetricsRecorder) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if rec == nil {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &statusRecorder{ResponseWriter: w}
			next.ServeHTTP(rw, r)

			status := rw.status
			if status == 0 {
				status = http.StatusOK
			}
			route := r.Pattern
			if route == "" {
				// Outside the mux (e.g. notFound). Use the literal path
				// only as a fallback — high-cardinality routes should
				// always have a registered pattern.
				route = r.URL.Path
			}
			rec.ObserveHTTP(r.Method, route, status, time.Since(start))
		})
	}
}
