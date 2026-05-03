package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPHost        string
	HTTPPort        int
	Environment     string
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

	// ShutdownTimeout caps how long the server waits for in-flight
	// requests after receiving SIGINT/SIGTERM before forcing close.
	ShutdownTimeout time.Duration

	// CORSAllowedOrigins is the explicit allowlist of origins the API
	// will respond to. Empty means "no CORS" (only same-origin clients
	// can reach the API), the literal "*" entry means "any origin".
	CORSAllowedOrigins []string

	// RateLimitRPS / RateLimitBurst configure the per-identity token
	// bucket. RateLimitRPS == 0 disables rate limiting entirely.
	RateLimitRPS   float64
	RateLimitBurst float64

	// SteamAPIKey is the Steam Web API key used by SteamAdapter.
	SteamAPIKey string
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

	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}

	shutdownTimeout := 15 * time.Second
	if v := os.Getenv("SHUTDOWN_TIMEOUT_SECONDS"); v != "" {
		if s, err := strconv.Atoi(v); err == nil && s > 0 {
			shutdownTimeout = time.Duration(s) * time.Second
		}
	}

	var corsOrigins []string
	if v := os.Getenv("CORS_ALLOWED_ORIGINS"); v != "" {
		for _, o := range strings.Split(v, ",") {
			if trimmed := strings.TrimSpace(o); trimmed != "" {
				corsOrigins = append(corsOrigins, trimmed)
			}
		}
	}

	rps := envFloat("RATE_LIMIT_RPS", 20)
	burst := envFloat("RATE_LIMIT_BURST", 40)

	steamAPIKey := os.Getenv("STEAM_API_KEY")

	return Config{
		HTTPHost:                      "0.0.0.0",
		HTTPPort:                      port,
		Environment:                   env,
		DatabaseURL:                   dbURL,
		JWTSecret:                     jwtSecret,
		AIEngineURL:                   aiURL,
		AIEngineTimeout:               aiTimeout,
		SimilarityCacheTTL:            simTTL,
		UserSimilarityCacheMaxEntries: userSimMax,
		ItemSimilarityCacheMaxEntries: itemSimMax,
		ShutdownTimeout:               shutdownTimeout,
		CORSAllowedOrigins:            corsOrigins,
		RateLimitRPS:                  rps,
		RateLimitBurst:                burst,
		SteamAPIKey:                   steamAPIKey,
	}
}

func envFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f >= 0 {
			return f
		}
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}
