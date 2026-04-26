package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newCORSChain(t *testing.T, cfg CORSConfig) http.Handler {
	t.Helper()
	return CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("body"))
	}))
}

func TestCORS_NoOriginPassesThrough(t *testing.T) {
	chain := newCORSChain(t, CORSConfig{Origins: []string{"https://web.aura"}})

	w := httptest.NewRecorder()
	chain.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no Allow-Origin header without an Origin, got %q", got)
	}
}

func TestCORS_AllowedOriginEchoed(t *testing.T) {
	chain := newCORSChain(t, CORSConfig{
		Origins:          []string{"https://web.aura"},
		AllowCredentials: true,
		ExposeHeaders:    []string{"X-Request-ID"},
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Origin", "https://web.aura")
	w := httptest.NewRecorder()
	chain.ServeHTTP(w, r)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://web.aura" {
		t.Fatalf("Allow-Origin = %q, want exact echo", got)
	}
	if w.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatal("expected credentials flag for allowlisted origin")
	}
	if !strings.Contains(w.Header().Get("Vary"), "Origin") {
		t.Fatal("expected Vary: Origin so caches don't poison each other")
	}
	if w.Header().Get("Access-Control-Expose-Headers") != "X-Request-ID" {
		t.Fatal("expected Expose-Headers to surface request id")
	}
}

func TestCORS_ForbiddenOriginGetsNoAllowHeader(t *testing.T) {
	chain := newCORSChain(t, CORSConfig{Origins: []string{"https://web.aura"}})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Origin", "https://evil.example")
	w := httptest.NewRecorder()
	chain.ServeHTTP(w, r)

	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("Allow-Origin must not appear for unlisted origins")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("non-preflight requests should still be served, got %d", w.Code)
	}
}

func TestCORS_PreflightShortCircuits(t *testing.T) {
	called := false
	chain := CORS(CORSConfig{Origins: []string{"https://web.aura"}})(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))

	r := httptest.NewRequest(http.MethodOptions, "/", nil)
	r.Header.Set("Origin", "https://web.aura")
	r.Header.Set("Access-Control-Request-Method", "POST")
	r.Header.Set("Access-Control-Request-Headers", "Authorization")
	w := httptest.NewRecorder()
	chain.ServeHTTP(w, r)

	if called {
		t.Fatal("preflight must not invoke the downstream handler")
	}
	if w.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want 204", w.Code)
	}
	if !strings.Contains(w.Header().Get("Access-Control-Allow-Methods"), "POST") {
		t.Fatalf("expected POST in Allow-Methods, got %q", w.Header().Get("Access-Control-Allow-Methods"))
	}
}

func TestCORS_WildcardWithoutCredentials(t *testing.T) {
	chain := newCORSChain(t, CORSConfig{Origins: []string{"*"}})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Origin", "https://anywhere.example")
	w := httptest.NewRecorder()
	chain.ServeHTTP(w, r)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected wildcard, got %q", got)
	}
	if w.Header().Get("Access-Control-Allow-Credentials") != "" {
		t.Fatal("wildcard origin must not advertise credentials support")
	}
}
