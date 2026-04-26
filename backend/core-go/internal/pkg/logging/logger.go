// Package logging centralises the slog setup so every binary in the
// backend uses the same handler, level and field naming. Using slog
// directly (instead of a custom logger interface) keeps third-party
// libraries that already log via slog.Default() integrated for free.
package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// New returns a slog.Logger configured for the given environment.
//
// "production" → JSON handler on stdout (machine-readable, ready for any
// log aggregator). Anything else → human-readable text on stderr, useful
// in development. The level is read from LOG_LEVEL (debug|info|warn|error)
// and defaults to info.
func New(env string) *slog.Logger {
	level := parseLevel(os.Getenv("LOG_LEVEL"))
	opts := &slog.HandlerOptions{Level: level}

	var (
		out     io.Writer = os.Stderr
		handler slog.Handler
	)
	if strings.EqualFold(env, "production") || strings.EqualFold(env, "prod") {
		out = os.Stdout
		handler = slog.NewJSONHandler(out, opts)
	} else {
		handler = slog.NewTextHandler(out, opts)
	}
	return slog.New(handler)
}

// SetDefault swaps the package-level slog default so libraries that call
// slog.Info / slog.Error without taking a logger inherit our formatting.
func SetDefault(l *slog.Logger) {
	slog.SetDefault(l)
}

func parseLevel(v string) slog.Level {
	switch strings.ToLower(v) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
