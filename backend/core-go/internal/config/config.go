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

	// Similarity-cache tuning. TTL applies to both caches; the max-entries
	// limits are separate because the user pool is typically larger than
	// the popular-items pool, so each side wants its own ceiling.
	SimilarityCacheTTL            time.Duration
	UserSimilarityCacheMaxEntries int
	ItemSimilarityCacheMaxEntries int
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

	simTTL := 30 * time.Minute
	if v := os.Getenv("SIMILARITY_CACHE_TTL_SECONDS"); v != "" {
		if s, err := strconv.Atoi(v); err == nil && s > 0 {
			simTTL = time.Duration(s) * time.Second
		}
	}
	userSimMax := envInt("USER_SIMILARITY_CACHE_MAX", 100_000)
	itemSimMax := envInt("ITEM_SIMILARITY_CACHE_MAX", 50_000)

	return Config{
		HTTPHost:                      "0.0.0.0",
		HTTPPort:                      port,
		DatabaseURL:                   dbURL,
		JWTSecret:                     jwtSecret,
		AIEngineURL:                   aiURL,
		AIEngineTimeout:               aiTimeout,
		SimilarityCacheTTL:            simTTL,
		UserSimilarityCacheMaxEntries: userSimMax,
		ItemSimilarityCacheMaxEntries: itemSimMax,
	}
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}
