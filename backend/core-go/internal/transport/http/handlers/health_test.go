package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHealthHandler_NoCheckersIsLiveness(t *testing.T) {
	w := httptest.NewRecorder()
	HealthHandler(time.Second)(w, httptest.NewRequest(http.MethodGet, "/health", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("status field = %v", body["status"])
	}
}

func TestHealthHandler_ReportsAllChecks(t *testing.T) {
	checker := CheckerFunc{NameValue: "database", Fn: func(context.Context) error { return nil }}
	w := httptest.NewRecorder()
	HealthHandler(time.Second, checker)(w, httptest.NewRequest(http.MethodGet, "/health", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var body struct {
		Status string                    `json:"status"`
		Checks map[string]map[string]any `json:"checks"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Status != "ok" {
		t.Fatalf("status = %q", body.Status)
	}
	if body.Checks["database"]["status"] != "up" {
		t.Fatalf("db check = %v", body.Checks["database"])
	}
}

func TestHealthHandler_FailingCheckTurnsResponse503(t *testing.T) {
	good := CheckerFunc{NameValue: "database", Fn: func(context.Context) error { return nil }}
	bad := CheckerFunc{NameValue: "ai_engine", Fn: func(context.Context) error { return errors.New("boom") }}

	w := httptest.NewRecorder()
	HealthHandler(time.Second, good, bad)(w, httptest.NewRequest(http.MethodGet, "/health", nil))

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
	var body struct {
		Status string                    `json:"status"`
		Checks map[string]map[string]any `json:"checks"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Status != "degraded" {
		t.Fatalf("status = %q, want \"degraded\"", body.Status)
	}
	if body.Checks["ai_engine"]["status"] != "down" {
		t.Fatalf("ai_engine check = %v", body.Checks["ai_engine"])
	}
	if body.Checks["ai_engine"]["error"] != "boom" {
		t.Fatalf("ai_engine error = %v", body.Checks["ai_engine"]["error"])
	}
	if body.Checks["database"]["status"] != "up" {
		t.Fatalf("database check should remain up: %v", body.Checks["database"])
	}
}

func TestHealthHandler_AppliesTimeout(t *testing.T) {
	slow := CheckerFunc{NameValue: "slow", Fn: func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
			return nil
		}
	}}

	w := httptest.NewRecorder()
	HealthHandler(20*time.Millisecond, slow)(w, httptest.NewRequest(http.MethodGet, "/health", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected timeout to surface as 503, got %d", w.Code)
	}
}
