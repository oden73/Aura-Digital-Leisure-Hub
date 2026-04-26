package handlers

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

type capturedCall struct {
	method, route string
	status        int
	duration      time.Duration
}

type spyRecorder struct {
	calls atomic.Int64
	last  capturedCall
}

func (s *spyRecorder) ObserveHTTP(method, route string, status int, duration time.Duration) {
	s.calls.Add(1)
	s.last = capturedCall{method, route, status, duration}
}

func TestMetricsMiddleware_RecordsObservation(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/items/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	rec := &spyRecorder{}
	chain := Metrics(rec)(mux)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/items/abc", nil)
	chain.ServeHTTP(w, r)

	if rec.calls.Load() != 1 {
		t.Fatalf("expected 1 metric observation, got %d", rec.calls.Load())
	}
	if rec.last.method != http.MethodGet {
		t.Fatalf("method = %q, want GET", rec.last.method)
	}
	if rec.last.status != http.StatusCreated {
		t.Fatalf("status = %d, want 201", rec.last.status)
	}
	// The route label should be the matched mux pattern, not the raw path,
	// otherwise Prometheus cardinality blows up with every distinct id.
	if rec.last.route != "GET /v1/items/{id}" {
		t.Fatalf("route = %q, want pattern \"GET /v1/items/{id}\"", rec.last.route)
	}
}

func TestMetricsMiddleware_NilRecorderIsTransparent(t *testing.T) {
	called := false
	chain := Metrics(nil)(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		called = true
	}))
	chain.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	if !called {
		t.Fatal("nil recorder must not block the chain")
	}
}
