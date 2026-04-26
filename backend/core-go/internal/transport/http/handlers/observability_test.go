package handlers

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestID_GeneratesAndExposesHeader(t *testing.T) {
	got := ""
	h := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = RequestIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if got == "" {
		t.Fatal("expected a generated request id in context")
	}
	if w.Header().Get("X-Request-ID") != got {
		t.Fatalf("expected response header to mirror context id: header=%q ctx=%q", w.Header().Get("X-Request-ID"), got)
	}
}

func TestRequestID_TrustsValidIncomingHeader(t *testing.T) {
	in := "abcdef0123456789abcdef0123456789"
	var got string
	h := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = RequestIDFromContext(r.Context())
	}))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Request-ID", in)
	h.ServeHTTP(httptest.NewRecorder(), r)

	if got != in {
		t.Fatalf("expected pass-through of valid X-Request-ID, got %q", got)
	}
}

func TestRequestID_RejectsInjection(t *testing.T) {
	bad := "this is not a valid id\nwith newline" // contains spaces & newline
	var got string
	h := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = RequestIDFromContext(r.Context())
	}))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Request-ID", bad)
	h.ServeHTTP(httptest.NewRecorder(), r)

	if got == bad {
		t.Fatalf("expected suspicious header to be replaced, got %q", got)
	}
	if got == "" {
		t.Fatal("expected a generated id even when input was rejected")
	}
}

func TestAccessLog_EmitsStructuredEntry(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	chain := RequestID(AccessLog(logger)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("ok"))
	})))

	r := httptest.NewRequest(http.MethodPost, "/v1/auth/register", strings.NewReader(""))
	w := httptest.NewRecorder()
	chain.ServeHTTP(w, r)

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("expected JSON access log line, got %s", buf.String())
	}
	if entry["msg"] != "http_request" {
		t.Fatalf("unexpected msg: %v", entry["msg"])
	}
	if entry["method"] != "POST" {
		t.Fatalf("expected method POST, got %v", entry["method"])
	}
	if entry["status"] != float64(http.StatusCreated) {
		t.Fatalf("expected status 201, got %v", entry["status"])
	}
	if entry["request_id"] == "" || entry["request_id"] == nil {
		t.Fatalf("expected request_id in log entry, got %v", entry["request_id"])
	}
}

func TestAccessLog_LogsErrorLevelOn5xx(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	chain := AccessLog(logger)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	chain.ServeHTTP(httptest.NewRecorder(), r)

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if entry["level"] != "ERROR" {
		t.Fatalf("expected ERROR level for a 500 response, got %v", entry["level"])
	}
}
