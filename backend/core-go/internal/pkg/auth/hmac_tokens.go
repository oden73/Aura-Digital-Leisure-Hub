package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var ErrInvalidToken = errors.New("invalid token")

type HMACTokenManager struct {
	Secret     []byte
	AccessTTL  time.Duration
	RefreshTTL time.Duration
	Issuer     string
}

type tokenClaims struct {
	Iss string `json:"iss"`
	Sub string `json:"sub"`
	UID string `json:"uid"`
	Exp int64  `json:"exp"`
	Iat int64  `json:"iat"`
	Jti string `json:"jti"`
}

func (m HMACTokenManager) Generate(userID string) (Token, error) {
	now := time.Now()
	access, err := m.sign(tokenClaims{
		Iss: m.Issuer,
		Sub: userID,
		UID: userID,
		Iat: now.Unix(),
		Exp: now.Add(m.AccessTTL).Unix(),
		Jti: randID(),
	})
	if err != nil {
		return Token{}, err
	}
	refresh, err := m.sign(tokenClaims{
		Iss: m.Issuer,
		Sub: userID,
		UID: userID,
		Iat: now.Unix(),
		Exp: now.Add(m.RefreshTTL).Unix(),
		Jti: randID(),
	})
	if err != nil {
		return Token{}, err
	}
	return Token{Access: access, Refresh: refresh}, nil
}

func (m HMACTokenManager) Validate(access string) (string, error) {
	c, err := m.verify(access)
	if err != nil {
		return "", err
	}
	return c.UID, nil
}

func (m HMACTokenManager) Refresh(refresh string) (Token, error) {
	c, err := m.verify(refresh)
	if err != nil {
		return Token{}, err
	}
	return m.Generate(c.UID)
}

func (m HMACTokenManager) sign(c tokenClaims) (string, error) {
	payload, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	p := base64.RawURLEncoding.EncodeToString(payload)
	sig := hmacSHA256(m.Secret, []byte(p))
	return p + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

func (m HMACTokenManager) verify(tok string) (tokenClaims, error) {
	parts := strings.Split(tok, ".")
	if len(parts) != 2 {
		return tokenClaims{}, ErrInvalidToken
	}
	payloadB, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return tokenClaims{}, ErrInvalidToken
	}
	sigB, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return tokenClaims{}, ErrInvalidToken
	}
	want := hmacSHA256(m.Secret, []byte(parts[0]))
	if !hmac.Equal(sigB, want) {
		return tokenClaims{}, ErrInvalidToken
	}
	var c tokenClaims
	if err := json.Unmarshal(payloadB, &c); err != nil {
		return tokenClaims{}, ErrInvalidToken
	}
	if c.Iss != m.Issuer || c.UID == "" {
		return tokenClaims{}, ErrInvalidToken
	}
	if time.Now().Unix() >= c.Exp {
		return tokenClaims{}, ErrInvalidToken
	}
	return c, nil
}

func hmacSHA256(secret, msg []byte) []byte {
	h := hmac.New(sha256.New, secret)
	_, _ = h.Write(msg)
	return h.Sum(nil)
}

func randID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return base64.RawURLEncoding.EncodeToString(b[:])
}

