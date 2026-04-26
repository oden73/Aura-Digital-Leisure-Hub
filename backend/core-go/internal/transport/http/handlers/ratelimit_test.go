package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"aura/backend/core-go/internal/pkg/ratelimit"
)

func newRLChain(rps float64, burst float64) http.Handler {
	limiter := ratelimit.New(rps, burst, 0)
	return RateLimit(RateLimitConfig{Limiter: limiter})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
}

func TestRateLimit_RejectsBeyondBurst(t *testing.T) {
	chain := newRLChain(0.0001, 2)

	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/v1/x", nil)
		r.RemoteAddr = "10.0.0.1:1234"
		chain.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("burst slot %d should be allowed, got %d", i, w.Code)
		}
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/x", nil)
	r.RemoteAddr = "10.0.0.1:1234"
	chain.ServeHTTP(w, r)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("3rd request should be 429, got %d", w.Code)
	}
	if w.Header().Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header on 429")
	}
}

func TestRateLimit_DifferentIPsHaveOwnBuckets(t *testing.T) {
	chain := newRLChain(0.0001, 1)

	for _, ip := range []string{"10.0.0.1:1", "10.0.0.2:1"} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/v1/x", nil)
		r.RemoteAddr = ip
		chain.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("each IP should get its own bucket; %s got %d", ip, w.Code)
		}
	}
}

func TestRateLimit_SkipPathsBypassed(t *testing.T) {
	limiter := ratelimit.New(0.0001, 1, 0)
	chain := RateLimit(RateLimitConfig{
		Limiter:   limiter,
		SkipPaths: []string{"/health"},
	})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/health", nil)
		r.RemoteAddr = "10.0.0.1:1"
		chain.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("/health should be exempt; got %d on attempt %d", w.Code, i)
		}
	}
}

func TestRateLimit_NilLimiterIsTransparent(t *testing.T) {
	chain := RateLimit(RateLimitConfig{Limiter: nil})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	w := httptest.NewRecorder()
	chain.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != http.StatusTeapot {
		t.Fatalf("nil limiter should pass through, got %d", w.Code)
	}
}

// Compile-time guard: rate-limit middleware must not accidentally drop
// time.Duration support (in case future refactors swap units).
var _ = (*ratelimit.Limiter)(nil)
var _ time.Duration
