// Package auth provides the AuthenticationService skeleton from the
// application-layer class diagram.
package auth

import "aura/backend/core-go/internal/domain/entities"

// Credentials is the user-provided login payload.
type Credentials struct {
	Email    string
	Password string
}

// Token is an opaque bearer token with optional refresh companion.
type Token struct {
	Access  string
	Refresh string
}

// TokenManager is responsible for issuing and validating tokens.
type TokenManager interface {
	Generate(userID string) (Token, error)
	Validate(access string) (string, error) // returns user_id
	Refresh(refresh string) (Token, error)
}

// Service authenticates users and issues session tokens.
type Service struct {
	Tokens TokenManager
	Users  interface {
		GetByEmail(email string) (entities.User, error)
	}
}

// New constructs an auth service.
func New(tokens TokenManager, users interface {
	GetByEmail(email string) (entities.User, error)
}) *Service {
	return &Service{Tokens: tokens, Users: users}
}

// Authenticate verifies credentials and returns an access token.
func (s *Service) Authenticate(c Credentials) (Token, error) {
	u, err := s.Users.GetByEmail(c.Email)
	if err != nil {
		return Token{}, err
	}
	if err := ComparePassword(u.PasswordHash, c.Password); err != nil {
		return Token{}, err
	}
	return s.Tokens.Generate(u.ID)
}

// ValidateToken resolves a token to a user.
func (s *Service) ValidateToken(_ string) (entities.User, error) {
	return entities.User{}, nil
}
