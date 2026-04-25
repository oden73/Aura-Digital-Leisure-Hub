package config

import (
	"os"
	"strconv"
)

type Config struct {
	HTTPHost string
	HTTPPort int
	DatabaseURL string
	JWTSecret   string
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

	return Config{
		HTTPHost: "0.0.0.0",
		HTTPPort: port,
		DatabaseURL: dbURL,
		JWTSecret:   jwtSecret,
	}
}
