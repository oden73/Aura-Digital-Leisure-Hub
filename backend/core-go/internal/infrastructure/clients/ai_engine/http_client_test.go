package ai_engine

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"aura/backend/core-go/internal/domain/entities"
)

func TestHTTPClient_DefaultTimeoutWhenNonPositive(t *testing.T) {
	c := NewHTTPClient("http://example.test", 0)
	if c.HTTP.Timeout <= 0 {
		t.Fatalf("expected positive timeout, got %v", c.HTTP.Timeout)
	}
}

func TestHTTPClient_ComputeCB(t *testing.T) {
	var captured cbRequestPayload

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/recommendations/cb" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"items": [
				{"item_id": "a", "score": 0.9, "source": "cb", "match_reason": "similar to X"},
				{"item_id": "b", "score": 0.5, "source": "cb"}
			],
			"reasoning": "because reasons"
		}`))
	}))
	defer srv.Close()

	c := NewHTTPClient(srv.URL, 500*time.Millisecond)
	resp, err := c.ComputeCB(Request{
		UserID:     "u-1",
		Limit:      10,
		MediaTypes: []entities.MediaType{entities.MediaTypeBook, entities.MediaTypeGame},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if captured.UserID != "u-1" || captured.Limit != 10 || len(captured.MediaTypes) != 2 {
		t.Fatalf("payload mismatch: %#v", captured)
	}
	if resp.Reasoning != "because reasons" {
		t.Fatalf("unexpected reasoning %q", resp.Reasoning)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(resp.Items))
	}
	if resp.Items[0].Source != entities.ScoreSourceCB {
		t.Fatalf("source not propagated: %q", resp.Items[0].Source)
	}
	if r, _ := resp.Items[0].Metadata["match_reason"].(string); r != "similar to X" {
		t.Fatalf("match_reason not captured: %#v", resp.Items[0].Metadata)
	}
}

func TestHTTPClient_ComputeCB_Non2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}))
	defer srv.Close()

	c := NewHTTPClient(srv.URL, 500*time.Millisecond)
	if _, err := c.ComputeCB(Request{UserID: "u-1", Limit: 5}); err == nil {
		t.Fatal("expected error on non-2xx response")
	}
}

func TestHTTPClient_GenerateEmbedding(t *testing.T) {
	var captured embeddingRequestPayload

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/embeddings/generate" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"item_id":"i-1","vector":[0.1,0.2]}`))
	}))
	defer srv.Close()

	c := NewHTTPClient(srv.URL, 500*time.Millisecond)
	if err := c.GenerateEmbedding(EmbeddingRequest{ItemID: "i-1", Text: "Описание книги"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if captured.ItemID != "i-1" || captured.Text != "Описание книги" {
		t.Fatalf("payload mismatch: %#v", captured)
	}
}

func TestHTTPClient_GenerateEmbedding_RejectsEmpty(t *testing.T) {
	c := NewHTTPClient("http://unused", 100*time.Millisecond)
	if err := c.GenerateEmbedding(EmbeddingRequest{ItemID: "", Text: "x"}); err == nil {
		t.Fatal("expected error when item_id is empty")
	}
	if err := c.GenerateEmbedding(EmbeddingRequest{ItemID: "i", Text: ""}); err == nil {
		t.Fatal("expected error when text is empty")
	}
}

func TestHTTPClient_GenerateEmbedding_Non2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("upstream"))
	}))
	defer srv.Close()

	c := NewHTTPClient(srv.URL, 500*time.Millisecond)
	if err := c.GenerateEmbedding(EmbeddingRequest{ItemID: "i", Text: "t"}); err == nil {
		t.Fatal("expected error on non-2xx response")
	}
}

func TestHTTPClient_GenerateReasoningPlaceholder(t *testing.T) {
	c := NewHTTPClient("http://unused", time.Second)

	got, err := c.GenerateReasoning("u-1", []entities.ScoredItem{{ItemID: "i-1"}})
	if err != nil {
		t.Fatalf("generate reasoning: %v", err)
	}
	if got != "" {
		t.Fatalf("reasoning = %q, want empty", got)
	}
}

func TestHTTPClient_Chat(t *testing.T) {
	var captured chatRequestPayload
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/assistant/chat" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)
		_, _ = w.Write([]byte(`{"text":"hello","recommendation_ids":["i-1"]}`))
	}))
	defer srv.Close()

	c := NewHTTPClient(srv.URL, 500*time.Millisecond)
	got, err := c.Chat(ChatRequest{
		Message: "hi",
		History: []ChatMessage{{Role: "user", Content: "previous"}},
	})
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	if captured.Message != "hi" || len(captured.History) != 1 {
		t.Fatalf("payload mismatch: %+v", captured)
	}
	if got.Text != "hello" || len(got.RecommendationIDs) != 1 || got.RecommendationIDs[0] != "i-1" {
		t.Fatalf("response mismatch: %+v", got)
	}
}

func TestHTTPClient_ChatNormalizesNilHistoryAndNilIDs(t *testing.T) {
	var captured chatRequestPayload
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)
		_, _ = w.Write([]byte(`{"text":"hello"}`))
	}))
	defer srv.Close()

	c := NewHTTPClient(srv.URL, 500*time.Millisecond)
	got, err := c.Chat(ChatRequest{Message: "hi"})
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	if captured.History == nil || len(captured.History) != 0 {
		t.Fatalf("history should be encoded as empty slice: %+v", captured.History)
	}
	if got.RecommendationIDs == nil || len(got.RecommendationIDs) != 0 {
		t.Fatalf("recommendation ids should be empty slice: %+v", got.RecommendationIDs)
	}
}

func TestHTTPClient_ChatNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("upstream"))
	}))
	defer srv.Close()

	c := NewHTTPClient(srv.URL, 500*time.Millisecond)
	if _, err := c.Chat(ChatRequest{Message: "hi"}); err == nil {
		t.Fatal("expected chat error on non-2xx response")
	}
}

type spyCallMetrics struct {
	ops []string
	ok  []bool
}

func (s *spyCallMetrics) IncAIEngineCall(operation string, success bool) {
	s.ops = append(s.ops, operation)
	s.ok = append(s.ok, success)
}

func TestHTTPClient_WithMetricsRecordsSuccessAndFailure(t *testing.T) {
	spy := &spyCallMetrics{}

	okSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"items":[],"reasoning":""}`))
	}))
	defer okSrv.Close()

	c := NewHTTPClient(okSrv.URL, 500*time.Millisecond).WithMetrics(spy)
	if _, err := c.ComputeCB(Request{UserID: "u", Limit: 1}); err != nil {
		t.Fatalf("compute: %v", err)
	}

	failSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer failSrv.Close()
	c2 := NewHTTPClient(failSrv.URL, 500*time.Millisecond).WithMetrics(spy)
	if _, err := c2.ComputeCB(Request{UserID: "u", Limit: 1}); err == nil {
		t.Fatal("expected error")
	}

	if len(spy.ops) != 2 || spy.ops[0] != "compute_cb" || spy.ops[1] != "compute_cb" {
		t.Fatalf("ops = %#v", spy.ops)
	}
	if len(spy.ok) != 2 || !spy.ok[0] || spy.ok[1] {
		t.Fatalf("success flags = %#v", spy.ok)
	}
}

func TestStubClientReturnsEmptyResults(t *testing.T) {
	stub := StubClient{}

	if resp, err := stub.ComputeCB(Request{}); err != nil || len(resp.Items) != 0 {
		t.Fatalf("ComputeCB = %+v %v", resp, err)
	}
	if got, err := stub.GenerateReasoning("u", nil); err != nil || got != "" {
		t.Fatalf("GenerateReasoning = %q %v", got, err)
	}
	if err := stub.GenerateEmbedding(EmbeddingRequest{}); err != nil {
		t.Fatalf("GenerateEmbedding = %v", err)
	}
	if resp, err := stub.Chat(ChatRequest{}); err != nil || resp.Text != "" {
		t.Fatalf("Chat = %+v %v", resp, err)
	}
}
