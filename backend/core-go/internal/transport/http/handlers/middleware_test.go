package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"aura/backend/core-go/internal/pkg/auth"
)

type fakeValidator struct {
	wantToken string
	wantUID   string
	err       error
	saw       string
}

func (f *fakeValidator) Generate(_ string) (auth.Token, error) { return auth.Token{}, nil }
func (f *fakeValidator) Refresh(_ string) (auth.Token, error)  { return auth.Token{}, nil }
func (f *fakeValidator) Validate(token string) (string, error) {
	f.saw = token
	if f.err != nil {
		return "", f.err
	}
	return f.wantUID, nil
}

func TestAuthMiddleware_RejectsMissingBearer(t *testing.T) {
	mw := Auth(&fakeValidator{}, func(http.ResponseWriter, *http.Request) {
		t.Fatal("inner handler must not be called without Authorization header")
	})
	r := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	if e := decodeError(t, w.Body.Bytes()); e.Code != "missing_token" {
		t.Fatalf("unexpected error code: %q", e.Code)
	}
}

func TestAuthMiddleware_RejectsNonBearerScheme(t *testing.T) {
	mw := Auth(&fakeValidator{}, func(http.ResponseWriter, *http.Request) {
		t.Fatal("inner handler must not be called for non-bearer auth")
	})
	r := httptest.NewRequest(http.MethodGet, "/protected", nil)
	r.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_RejectsInvalidToken(t *testing.T) {
	mw := Auth(&fakeValidator{err: errors.New("expired")}, func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler must not run on invalid token")
	})
	r := httptest.NewRequest(http.MethodGet, "/protected", nil)
	r.Header.Set("Authorization", "Bearer expired-token")
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	if e := decodeError(t, w.Body.Bytes()); e.Code != "invalid_token" {
		t.Fatalf("unexpected error code: %q", e.Code)
	}
}

func TestAuthMiddleware_PropagatesUserID(t *testing.T) {
	tok := &fakeValidator{wantUID: "u-42"}
	var seenUID string
	var seenOK bool
	mw := Auth(tok, func(_ http.ResponseWriter, r *http.Request) {
		seenUID, seenOK = userIDFromContext(r.Context())
	})

	r := httptest.NewRequest(http.MethodGet, "/protected", nil)
	r.Header.Set("Authorization", "Bearer good-token")
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, r)

	if !seenOK || seenUID != "u-42" {
		t.Fatalf("expected user_id u-42 in context, got %q (ok=%v)", seenUID, seenOK)
	}
	if tok.saw != "good-token" {
		t.Fatalf("token manager saw %q, want %q", tok.saw, "good-token")
	}
}

func TestUserIDFromContext_EmptyValueIsNotOK(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if uid, ok := userIDFromContext(r.Context()); ok || uid != "" {
		t.Fatalf("empty context must return false; got %q ok=%v", uid, ok)
	}
}
