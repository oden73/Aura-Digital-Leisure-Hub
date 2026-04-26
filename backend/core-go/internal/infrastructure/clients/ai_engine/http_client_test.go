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
