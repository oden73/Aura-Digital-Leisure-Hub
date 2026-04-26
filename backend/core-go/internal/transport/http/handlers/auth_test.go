package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aura/backend/core-go/internal/domain/entities"
	"aura/backend/core-go/internal/pkg/auth"
)

type fakeTokens struct {
	generateIn string
	validateIn string
	refreshIn  string
	out        auth.Token
	err        error
	validateID string
	validateEr error
}

func (f *fakeTokens) Generate(uid string) (auth.Token, error) {
	f.generateIn = uid
	return f.out, f.err
}

func (f *fakeTokens) Validate(token string) (string, error) {
	f.validateIn = token
	return f.validateID, f.validateEr
}

func (f *fakeTokens) Refresh(refresh string) (auth.Token, error) {
	f.refreshIn = refresh
	if f.err != nil {
		return auth.Token{}, f.err
	}
	return f.out, nil
}

type fakeAuthUsers struct {
	create   entities.User
	createEr error
	byEmail  entities.User
	byEmailE error
	byID     entities.User
	byIDE    error
}

func (f *fakeAuthUsers) Create(u entities.User) (entities.User, error) {
	if f.createEr != nil {
		return entities.User{}, f.createEr
	}
	if f.create.ID == "" {
		f.create = u
		f.create.ID = "u-new"
	}
	return f.create, nil
}
func (f *fakeAuthUsers) GetByEmail(_ string) (entities.User, error) { return f.byEmail, f.byEmailE }
func (f *fakeAuthUsers) GetByID(_ string) (entities.User, error)    { return f.byID, f.byIDE }

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

// ---------- register --------------------------------------------------------

func TestHandleRegister_Validation(t *testing.T) {
	cases := []struct {
		name, body string
		want       int
	}{
		{"bad-json", `{`, http.StatusBadRequest},
		{"missing-email", `{"username":"a","password":"p"}`, http.StatusBadRequest},
		{"missing-password", `{"username":"a","email":"a@b.c"}`, http.StatusBadRequest},
		{"missing-username", `{"email":"a@b.c","password":"p"}`, http.StatusBadRequest},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a := &AuthHandlers{
				Auth:  &auth.Service{Tokens: &fakeTokens{}},
				Users: &fakeAuthUsers{},
			}
			r := httptest.NewRequest(http.MethodPost, "/v1/auth/register", strings.NewReader(c.body))
			w := httptest.NewRecorder()
			a.HandleRegister(w, r)
			if w.Code != c.want {
				t.Fatalf("expected %d, got %d (body=%s)", c.want, w.Code, w.Body.String())
			}
		})
	}
}

func TestHandleRegister_HashesPasswordAndIssuesTokens(t *testing.T) {
	users := &fakeAuthUsers{}
	tokens := &fakeTokens{out: auth.Token{Access: "a", Refresh: "r"}}
	a := &AuthHandlers{Auth: &auth.Service{Tokens: tokens}, Users: users}

	body := `{"username":"alice","email":"a@b.c","password":"hunter2"}`
	r := httptest.NewRequest(http.MethodPost, "/v1/auth/register", strings.NewReader(body))
	w := httptest.NewRecorder()
	a.HandleRegister(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body=%s)", w.Code, w.Body.String())
	}
	if users.create.PasswordHash == "" || users.create.PasswordHash == "hunter2" {
		t.Fatalf("password must be hashed before persistence, got %q", users.create.PasswordHash)
	}
	if err := auth.ComparePassword(users.create.PasswordHash, "hunter2"); err != nil {
		t.Fatalf("stored hash must verify against original password: %v", err)
	}
	if tokens.generateIn != users.create.ID {
		t.Fatalf("token must be minted for the new user id; got %q for user %q",
			tokens.generateIn, users.create.ID)
	}
	var resp tokenResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Access != "a" || resp.Refresh != "r" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestHandleRegister_DuplicateEmailReturns400(t *testing.T) {
	users := &fakeAuthUsers{createEr: errors.New("unique violation")}
	a := &AuthHandlers{Auth: &auth.Service{Tokens: &fakeTokens{}}, Users: users}

	body := `{"username":"alice","email":"a@b.c","password":"x"}`
	r := httptest.NewRequest(http.MethodPost, "/v1/auth/register", strings.NewReader(body))
	w := httptest.NewRecorder()
	a.HandleRegister(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on user creation failure, got %d", w.Code)
	}
}

// ---------- login -----------------------------------------------------------

func TestHandleLogin_InvalidCredentialsReturns401(t *testing.T) {
	users := &fakeAuthUsers{byEmailE: auth.ErrInvalidCredentials}
	a := &AuthHandlers{Auth: &auth.Service{Tokens: &fakeTokens{}, Users: users}, Users: users}

	r := httptest.NewRequest(http.MethodPost, "/v1/auth/login",
		strings.NewReader(`{"email":"x","password":"y"}`))
	w := httptest.NewRecorder()
	a.HandleLogin(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestHandleLogin_OK(t *testing.T) {
	hash, _ := auth.HashPassword("hunter2")
	users := &fakeAuthUsers{byEmail: entities.User{ID: "u-1", PasswordHash: hash}}
	tokens := &fakeTokens{out: auth.Token{Access: "a", Refresh: "r"}}
	a := &AuthHandlers{Auth: &auth.Service{Tokens: tokens, Users: users}, Users: users}

	r := httptest.NewRequest(http.MethodPost, "/v1/auth/login",
		strings.NewReader(`{"email":"x","password":"hunter2"}`))
	w := httptest.NewRecorder()
	a.HandleLogin(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", w.Code, w.Body.String())
	}
	if tokens.generateIn != "u-1" {
		t.Fatalf("expected token for u-1, got %q", tokens.generateIn)
	}
}

func TestHandleLogin_Validation(t *testing.T) {
	a := &AuthHandlers{
		Auth:  &auth.Service{Tokens: &fakeTokens{}, Users: &fakeAuthUsers{}},
		Users: &fakeAuthUsers{},
	}
	r := httptest.NewRequest(http.MethodPost, "/v1/auth/login", strings.NewReader(`{"email":""}`))
	w := httptest.NewRecorder()
	a.HandleLogin(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on missing fields, got %d", w.Code)
	}
}
