package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"aura/backend/core-go/internal/pkg/ratelimit"
)

func TestCheckAIEngine_EmptyURL(t *testing.T) {
	err := checkAIEngine(context.Background(), "", time.Second)
	if err == nil {
		t.Fatal("expected error for empty base URL")
	}
	if !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckAIEngine_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := checkAIEngine(context.Background(), srv.URL, time.Second)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestCheckAIEngine_Non2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	if err := checkAIEngine(context.Background(), srv.URL, time.Second); err == nil {
		t.Fatal("expected error on non-2xx")
	}
}

func TestCheckAIEngine_DefaultTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	if err := checkAIEngine(context.Background(), srv.URL, 0); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestRunRateLimitSweeper_TickerCallsSweep(t *testing.T) {
	l := ratelimit.New(1, 2, time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	done := make(chan struct{})
	go func() {
		runRateLimitSweeper(ctx, l, time.Millisecond)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("sweeper did not exit")
	}
}

func TestRunRateLimitSweeper_StopsOnCancel(t *testing.T) {
	l := ratelimit.New(1, 2, time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		runRateLimitSweeper(ctx, l, time.Millisecond)
		close(done)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("sweeper did not exit after cancel")
	}
}
