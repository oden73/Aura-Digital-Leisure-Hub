package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aura/backend/core-go/internal/pkg/auth"
)

type fakeTokens struct {
	accessIn  string
	refreshIn string
	out       auth.Token
	err       error
}

func (f *fakeTokens) Generate(_ string) (auth.Token, error)        { return f.out, f.err }
func (f *fakeTokens) Validate(_ string) (string, error)            { return "", nil }
func (f *fakeTokens) Refresh(refresh string) (auth.Token, error) {
	f.refreshIn = refresh
	if f.err != nil {
		return auth.Token{}, f.err
	}
	return f.out, nil
}

func TestHandleRefresh_Success(t *testing.T) {
	tokens := &fakeTokens{out: auth.Token{Access: "new-access", Refresh: "new-refresh"}}
	a := &AuthHandlers{Auth: &auth.Service{Tokens: tokens}}

	body := strings.NewReader(`{"refresh":"old-refresh"}`)
	r := httptest.NewRequest(http.MethodPost, "/v1/auth/refresh", body)
	w := httptest.NewRecorder()
	a.HandleRefresh(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if tokens.refreshIn != "old-refresh" {
		t.Fatalf("expected token manager to receive refresh token, got %q", tokens.refreshIn)
	}
	var resp tokenResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Access != "new-access" || resp.Refresh != "new-refresh" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestHandleRefresh_RejectsEmpty(t *testing.T) {
	a := &AuthHandlers{Auth: &auth.Service{Tokens: &fakeTokens{}}}
	r := httptest.NewRequest(http.MethodPost, "/v1/auth/refresh", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	a.HandleRefresh(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on empty refresh, got %d", w.Code)
	}
}

func TestHandleRefresh_InvalidTokenReturns401(t *testing.T) {
	tokens := &fakeTokens{err: errors.New("expired")}
	a := &AuthHandlers{Auth: &auth.Service{Tokens: tokens}}

	r := httptest.NewRequest(http.MethodPost, "/v1/auth/refresh", strings.NewReader(`{"refresh":"x"}`))
	w := httptest.NewRecorder()
	a.HandleRefresh(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 on invalid refresh, got %d", w.Code)
	}
}
