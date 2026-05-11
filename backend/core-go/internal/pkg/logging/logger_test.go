package logging

import (
	"bytes"
	"log/slog"
	"testing"
)

func TestNewCreatesUsableLogger(t *testing.T) {
	t.Setenv("LOG_LEVEL", "debug")

	logger := New("production")

	if logger == nil {
		t.Fatal("expected logger")
	}
}

func TestSetDefault(t *testing.T) {
	var out bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&out, nil))

	SetDefault(logger)
	slog.Info("hello")

	if !bytes.Contains(out.Bytes(), []byte("hello")) {
		t.Fatalf("default logger did not write message: %q", out.String())
	}
}

func TestParseLevel(t *testing.T) {
	cases := map[string]slog.Level{
		"debug":   slog.LevelDebug,
		"warn":    slog.LevelWarn,
		"warning": slog.LevelWarn,
		"error":   slog.LevelError,
		"INFO":    slog.LevelInfo,
		"":        slog.LevelInfo,
	}
	for input, want := range cases {
		if got := parseLevel(input); got != want {
			t.Fatalf("%q: got %v, want %v", input, got, want)
		}
	}
}
