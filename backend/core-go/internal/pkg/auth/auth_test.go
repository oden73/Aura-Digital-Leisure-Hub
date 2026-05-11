package auth

import (
	"errors"
	"testing"
	"time"

	"aura/backend/core-go/internal/domain/entities"
)

type fakeUserStore struct {
	user entities.User
	err  error
}

func (f fakeUserStore) GetByEmail(email string) (entities.User, error) {
	if f.err != nil {
		return entities.User{}, f.err
	}
	if email != f.user.Email {
		return entities.User{}, errors.New("not found")
	}
	return f.user, nil
}

type fakeTokenManager struct {
	userID string
	token  Token
	err    error
}

func (f *fakeTokenManager) Generate(userID string) (Token, error) {
	f.userID = userID
	if f.err != nil {
		return Token{}, f.err
	}
	return f.token, nil
}

func (f *fakeTokenManager) Validate(access string) (string, error) {
	return "", nil
}

func (f *fakeTokenManager) Refresh(refresh string) (Token, error) {
	return Token{}, nil
}

func TestPasswordHashRoundTrip(t *testing.T) {
	hash, err := HashPassword("s3cret")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if hash == "s3cret" {
		t.Fatal("hash must not store the plaintext password")
	}
	if err := ComparePassword(hash, "s3cret"); err != nil {
		t.Fatalf("expected password to match: %v", err)
	}
	if err := ComparePassword(hash, "wrong"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials, got %v", err)
	}
}

func TestComparePasswordRejectsMalformedHash(t *testing.T) {
	for _, stored := range []string{"", "abc", "not-hex$abc"} {
		if err := ComparePassword(stored, "password"); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("stored %q: expected invalid credentials, got %v", stored, err)
		}
	}
}

func TestHMACTokenManagerGenerateValidateRefresh(t *testing.T) {
	manager := HMACTokenManager{
		Secret:     []byte("test-secret"),
		AccessTTL:  time.Minute,
		RefreshTTL: time.Hour,
		Issuer:     "aura-test",
	}

	token, err := manager.Generate("u-1")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	if token.Access == "" || token.Refresh == "" || token.Access == token.Refresh {
		t.Fatalf("unexpected token pair: %+v", token)
	}

	userID, err := manager.Validate(token.Access)
	if err != nil {
		t.Fatalf("validate token: %v", err)
	}
	if userID != "u-1" {
		t.Fatalf("user id = %q, want u-1", userID)
	}

	refreshed, err := manager.Refresh(token.Refresh)
	if err != nil {
		t.Fatalf("refresh token: %v", err)
	}
	if refreshed.Access == "" || refreshed.Refresh == "" {
		t.Fatalf("empty refreshed token: %+v", refreshed)
	}
}

func TestHMACTokenManagerRejectsInvalidTokens(t *testing.T) {
	manager := HMACTokenManager{
		Secret:    []byte("test-secret"),
		AccessTTL: time.Minute,
		Issuer:    "aura-test",
	}
	token, err := manager.Generate("u-1")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	cases := []string{
		"",
		"not-a-token",
		token.Access + "tampered",
	}
	for _, tok := range cases {
		if _, err := manager.Validate(tok); !errors.Is(err, ErrInvalidToken) {
			t.Fatalf("token %q: expected invalid token, got %v", tok, err)
		}
	}

	wrongIssuer := manager
	wrongIssuer.Issuer = "other"
	if _, err := wrongIssuer.Validate(token.Access); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected issuer mismatch to be invalid, got %v", err)
	}
}

func TestHMACTokenManagerRejectsExpiredToken(t *testing.T) {
	manager := HMACTokenManager{
		Secret:    []byte("test-secret"),
		AccessTTL: -time.Second,
		Issuer:    "aura-test",
	}
	token, err := manager.Generate("u-1")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	if _, err := manager.Validate(token.Access); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected expired token to be invalid, got %v", err)
	}
}

func TestServiceAuthenticateIssuesToken(t *testing.T) {
	hash, err := HashPassword("password")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	tokens := &fakeTokenManager{token: Token{Access: "access", Refresh: "refresh"}}
	service := New(tokens, fakeUserStore{user: entities.User{
		ID:           "u-1",
		Email:        "user@example.com",
		PasswordHash: hash,
	}})

	got, err := service.Authenticate(Credentials{Email: "user@example.com", Password: "password"})
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if got.Access != "access" || got.Refresh != "refresh" {
		t.Fatalf("token = %+v", got)
	}
	if tokens.userID != "u-1" {
		t.Fatalf("token generated for %q, want u-1", tokens.userID)
	}
}

func TestServiceAuthenticatePropagatesFailures(t *testing.T) {
	hash, err := HashPassword("password")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	service := New(&fakeTokenManager{}, fakeUserStore{user: entities.User{
		ID:           "u-1",
		Email:        "user@example.com",
		PasswordHash: hash,
	}})
	if _, err := service.Authenticate(Credentials{Email: "user@example.com", Password: "wrong"}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials, got %v", err)
	}

	repoErr := errors.New("db down")
	service = New(&fakeTokenManager{}, fakeUserStore{err: repoErr})
	if _, err := service.Authenticate(Credentials{Email: "user@example.com", Password: "password"}); !errors.Is(err, repoErr) {
		t.Fatalf("expected repo error, got %v", err)
	}
}
