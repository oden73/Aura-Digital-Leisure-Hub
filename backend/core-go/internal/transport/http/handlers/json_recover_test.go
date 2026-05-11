package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSONSetsStatusHeaderAndBody(t *testing.T) {
	w := httptest.NewRecorder()

	writeJSON(w, http.StatusCreated, map[string]string{"id": "i-1"})

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", w.Code)
	}
	if got := w.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q", got)
	}
	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["id"] != "i-1" {
		t.Fatalf("body = %#v", body)
	}
}

func TestWriteErrorUsesAPIErrorShape(t *testing.T) {
	w := httptest.NewRecorder()

	writeError(w, http.StatusBadRequest, "bad_request", "Bad request")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	var body apiError
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Code != "bad_request" || body.Message != "Bad request" {
		t.Fatalf("body = %+v", body)
	}
}

func TestRecoverConvertsPanicToInternalError(t *testing.T) {
	h := Recover(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/panic", nil))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	var body apiError
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Code != "panic" || body.Message != "Internal error" {
		t.Fatalf("body = %+v", body)
	}
}

func TestRecoverPassesThroughHealthyHandler(t *testing.T) {
	h := Recover(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusAccepted, map[string]string{"status": "ok"})
	}))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ok", nil))

	if w.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", w.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("body = %#v", body)
	}
}
