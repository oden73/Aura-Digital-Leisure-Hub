package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPHost        string
	HTTPPort        int
	DatabaseURL     string
	JWTSecret       string
	AIEngineURL     string
	AIEngineTimeout time.Duration
}

func Load() Config {
	port := 8080
	if v := os.Getenv("HTTP_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			port = p
		}
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://aura:aura@localhost:5432/aura?sslmode=disable"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dev-secret"
	}

	aiURL := os.Getenv("AI_ENGINE_URL")
	if aiURL == "" {
		aiURL = "http://localhost:8000"
	}

	aiTimeout := 2 * time.Second
	if v := os.Getenv("AI_ENGINE_TIMEOUT_MS"); v != "" {
		if ms, err := strconv.Atoi(v); err == nil && ms > 0 {
			aiTimeout = time.Duration(ms) * time.Millisecond
		}
	}

	return Config{
		HTTPHost:        "0.0.0.0",
		HTTPPort:        port,
		DatabaseURL:     dbURL,
		JWTSecret:       jwtSecret,
		AIEngineURL:     aiURL,
		AIEngineTimeout: aiTimeout,
	}
}
