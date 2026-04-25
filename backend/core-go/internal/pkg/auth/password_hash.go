package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

func HashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	sum := sha256.Sum256(append(salt, []byte(password)...))
	return hex.EncodeToString(salt) + "$" + hex.EncodeToString(sum[:]), nil
}

func ComparePassword(stored, password string) error {
	parts := strings.Split(stored, "$")
	if len(parts) != 2 {
		return ErrInvalidCredentials
	}
	salt, err := hex.DecodeString(parts[0])
	if err != nil {
		return ErrInvalidCredentials
	}
	want := parts[1]
	sum := sha256.Sum256(append(salt, []byte(password)...))
	if hex.EncodeToString(sum[:]) != want {
		return ErrInvalidCredentials
	}
	return nil
}

