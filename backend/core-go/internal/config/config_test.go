package config

import (
	"reflect"
	"testing"
	"time"
)

func TestLoadUsesDefaultsForInvalidOrMissingValues(t *testing.T) {
	t.Setenv("APP_ENV", "")
	t.Setenv("HTTP_PORT", "bad")
	t.Setenv("AI_ENGINE_TIMEOUT_MS", "-1")
	t.Setenv("SIMILARITY_CACHE_TTL_SECONDS", "bad")
	t.Setenv("USER_SIMILARITY_CACHE_MAX", "0")
	t.Setenv("ITEM_SIMILARITY_CACHE_MAX", "-2")
	t.Setenv("SHUTDOWN_TIMEOUT_SECONDS", "0")
	t.Setenv("RATE_LIMIT_RPS", "-1")
	t.Setenv("RATE_LIMIT_BURST", "bad")

	cfg := Load()

	if cfg.HTTPPort != 8080 {
		t.Fatalf("HTTPPort = %d, want 8080", cfg.HTTPPort)
	}
	if cfg.DatabaseURL != "postgres://aura:aura@localhost:5432/aura?sslmode=disable" {
		t.Fatalf("DatabaseURL = %q", cfg.DatabaseURL)
	}
	if cfg.JWTSecret != "dev-secret" {
		t.Fatalf("JWTSecret = %q", cfg.JWTSecret)
	}
	if cfg.AIEngineURL != "http://localhost:8000" {
		t.Fatalf("AIEngineURL = %q", cfg.AIEngineURL)
	}
	if cfg.AIEngineTimeout != 2*time.Second {
		t.Fatalf("AIEngineTimeout = %v", cfg.AIEngineTimeout)
	}
	if cfg.SimilarityCacheTTL != 30*time.Minute {
		t.Fatalf("SimilarityCacheTTL = %v", cfg.SimilarityCacheTTL)
	}
	if cfg.UserSimilarityCacheMaxEntries != 100_000 {
		t.Fatalf("UserSimilarityCacheMaxEntries = %d", cfg.UserSimilarityCacheMaxEntries)
	}
	if cfg.ItemSimilarityCacheMaxEntries != 50_000 {
		t.Fatalf("ItemSimilarityCacheMaxEntries = %d", cfg.ItemSimilarityCacheMaxEntries)
	}
	if cfg.ShutdownTimeout != 15*time.Second {
		t.Fatalf("ShutdownTimeout = %v", cfg.ShutdownTimeout)
	}
	if cfg.RateLimitRPS != 20 || cfg.RateLimitBurst != 40 {
		t.Fatalf("rate limit = %v/%v", cfg.RateLimitRPS, cfg.RateLimitBurst)
	}
	wantOrigins := []string{
		"http://localhost:3000",
		"http://127.0.0.1:3000",
		"http://localhost:5173",
		"http://127.0.0.1:5173",
	}
	if !reflect.DeepEqual(cfg.CORSAllowedOrigins, wantOrigins) {
		t.Fatalf("CORSAllowedOrigins = %#v", cfg.CORSAllowedOrigins)
	}
}

func TestLoadAppliesEnvironmentOverrides(t *testing.T) {
	t.Setenv("HTTP_PORT", "9090")
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("JWT_SECRET", "secret")
	t.Setenv("AI_ENGINE_URL", "http://ai:8000")
	t.Setenv("AI_ENGINE_TIMEOUT_MS", "1500")
	t.Setenv("SIMILARITY_CACHE_TTL_SECONDS", "45")
	t.Setenv("USER_SIMILARITY_CACHE_MAX", "11")
	t.Setenv("ITEM_SIMILARITY_CACHE_MAX", "12")
	t.Setenv("APP_ENV", "production")
	t.Setenv("SHUTDOWN_TIMEOUT_SECONDS", "7")
	t.Setenv("CORS_ALLOWED_ORIGINS", " https://a.example,https://b.example ,, ")
	t.Setenv("RATE_LIMIT_RPS", "3.5")
	t.Setenv("RATE_LIMIT_BURST", "8")
	t.Setenv("STEAM_API_KEY", "steam-key")

	cfg := Load()

	if cfg.HTTPPort != 9090 || cfg.DatabaseURL != "postgres://example" || cfg.JWTSecret != "secret" {
		t.Fatalf("basic overrides not applied: %+v", cfg)
	}
	if cfg.AIEngineURL != "http://ai:8000" || cfg.AIEngineTimeout != 1500*time.Millisecond {
		t.Fatalf("AI overrides not applied: %+v", cfg)
	}
	if cfg.SimilarityCacheTTL != 45*time.Second ||
		cfg.UserSimilarityCacheMaxEntries != 11 ||
		cfg.ItemSimilarityCacheMaxEntries != 12 {
		t.Fatalf("similarity overrides not applied: %+v", cfg)
	}
	if cfg.Environment != "production" || cfg.ShutdownTimeout != 7*time.Second {
		t.Fatalf("environment overrides not applied: %+v", cfg)
	}
	wantOrigins := []string{"https://a.example", "https://b.example"}
	if !reflect.DeepEqual(cfg.CORSAllowedOrigins, wantOrigins) {
		t.Fatalf("CORSAllowedOrigins = %#v", cfg.CORSAllowedOrigins)
	}
	if cfg.RateLimitRPS != 3.5 || cfg.RateLimitBurst != 8 {
		t.Fatalf("rate limit overrides = %v/%v", cfg.RateLimitRPS, cfg.RateLimitBurst)
	}
	if cfg.SteamAPIKey != "steam-key" {
		t.Fatalf("SteamAPIKey = %q", cfg.SteamAPIKey)
	}
}

func TestLoadLeavesProductionCORSClosedByDefault(t *testing.T) {
	t.Setenv("APP_ENV", "production")

	cfg := Load()

	if len(cfg.CORSAllowedOrigins) != 0 {
		t.Fatalf("production CORS origins = %#v, want none", cfg.CORSAllowedOrigins)
	}
}
