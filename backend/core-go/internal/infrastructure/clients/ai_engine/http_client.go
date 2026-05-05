package ai_engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"aura/backend/core-go/internal/domain/entities"
)

// CallMetrics is the slice of the metrics surface this client touches.
// Defining it locally lets the package stay metrics-agnostic — production
// wires metrics.Recorder, tests pass nil.
type CallMetrics interface {
	IncAIEngineCall(operation string, success bool)
}

// HTTPClient is a thin REST client for the Python AI engine described in
// docs/predone/diagrams/sequence_diagram.puml (CB scores, LLM reasoning).
type HTTPClient struct {
	BaseURL string
	HTTP    *http.Client
	Metrics CallMetrics // optional
}

// NewHTTPClient constructs an HTTP client with sensible defaults. baseURL
// should not end with a slash; the configured timeout applies to every call.
func NewHTTPClient(baseURL string, timeout time.Duration) *HTTPClient {
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	return &HTTPClient{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTP:    &http.Client{Timeout: timeout},
	}
}

// WithMetrics enables Prometheus instrumentation for AI engine calls.
func (c *HTTPClient) WithMetrics(m CallMetrics) *HTTPClient {
	c.Metrics = m
	return c
}

func (c *HTTPClient) recordCall(op string, err error) {
	if c.Metrics != nil {
		c.Metrics.IncAIEngineCall(op, err == nil)
	}
}

type cbRequestPayload struct {
	UserID       string   `json:"user_id"`
	CandidateIDs []string `json:"candidate_ids"`
	Limit        int      `json:"limit"`
	MediaTypes   []string `json:"media_types"`
}

type cbScoredItem struct {
	ItemID      string  `json:"item_id"`
	Score       float64 `json:"score"`
	Source      string  `json:"source"`
	MatchReason string  `json:"match_reason"`
}

type cbResponsePayload struct {
	Items     []cbScoredItem `json:"items"`
	Reasoning string         `json:"reasoning"`
}

// ComputeCB performs POST /v1/recommendations/cb against the AI engine.
func (c *HTTPClient) ComputeCB(req Request) (Response, error) {
	out, err := c.computeCB(req)
	c.recordCall("compute_cb", err)
	return out, err
}

func (c *HTTPClient) computeCB(req Request) (Response, error) {
	payload := cbRequestPayload{
		UserID:       req.UserID,
		CandidateIDs: req.CandidateIDs,
		Limit:        req.Limit,
		MediaTypes:   make([]string, 0, len(req.MediaTypes)),
	}
	for _, m := range req.MediaTypes {
		payload.MediaTypes = append(payload.MediaTypes, string(m))
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return Response{}, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.HTTP.Timeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.BaseURL+"/v1/recommendations/cb",
		bytes.NewReader(body),
	)
	if err != nil {
		return Response{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return Response{}, fmt.Errorf("ai engine: status=%d body=%s", resp.StatusCode, string(raw))
	}

	var decoded cbResponsePayload
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return Response{}, err
	}

	out := Response{Reasoning: decoded.Reasoning}
	out.Items = make([]entities.ScoredItem, 0, len(decoded.Items))
	for _, it := range decoded.Items {
		meta := map[string]any{}
		if it.MatchReason != "" {
			meta["match_reason"] = it.MatchReason
		}
		out.Items = append(out.Items, entities.ScoredItem{
			ItemID:   it.ItemID,
			Score:    it.Score,
			Source:   entities.ScoreSourceCB,
			Metadata: meta,
		})
	}
	return out, nil
}

// GenerateReasoning is a placeholder until the AI engine exposes a dedicated
// reasoning endpoint. We currently piggy-back on whatever ComputeCB returned.
func (c *HTTPClient) GenerateReasoning(_ string, _ []entities.ScoredItem) (string, error) {
	return "", nil
}

type chatRequestPayload struct {
	Message string        `json:"message"`
	History []ChatMessage `json:"history"`
}

type chatResponsePayload struct {
	Text              string   `json:"text"`
	RecommendationIDs []string `json:"recommendation_ids"`
}

// Chat sends POST /v1/assistant/chat to the AI engine and returns the LLM reply.
func (c *HTTPClient) Chat(req ChatRequest) (ChatResponse, error) {
	out, err := c.chat(req)
	c.recordCall("chat", err)
	return out, err
}

func (c *HTTPClient) chat(req ChatRequest) (ChatResponse, error) {
	history := req.History
	if history == nil {
		history = []ChatMessage{}
	}
	body, err := json.Marshal(chatRequestPayload{Message: req.Message, History: history})
	if err != nil {
		return ChatResponse{}, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.BaseURL+"/v1/assistant/chat",
		bytes.NewReader(body),
	)
	if err != nil {
		return ChatResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return ChatResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return ChatResponse{}, fmt.Errorf("ai engine: status=%d body=%s", resp.StatusCode, string(raw))
	}

	var decoded chatResponsePayload
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return ChatResponse{}, err
	}
	ids := decoded.RecommendationIDs
	if ids == nil {
		ids = []string{}
	}
	return ChatResponse{Text: decoded.Text, RecommendationIDs: ids}, nil
}

type embeddingRequestPayload struct {
	ItemID string `json:"item_id"`
	Text   string `json:"text"`
}

// GenerateEmbedding sends item text to the AI engine for embedding and
// upserting into the vector store. The response body is intentionally
// discarded — Go core only cares that the embedding was persisted.
func (c *HTTPClient) GenerateEmbedding(req EmbeddingRequest) error {
	err := c.generateEmbedding(req)
	c.recordCall("generate_embedding", err)
	return err
}

func (c *HTTPClient) generateEmbedding(req EmbeddingRequest) error {
	if req.ItemID == "" || req.Text == "" {
		return fmt.Errorf("ai engine: item_id and text are required")
	}

	body, err := json.Marshal(embeddingRequestPayload{
		ItemID: req.ItemID,
		Text:   req.Text,
	})
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.HTTP.Timeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.BaseURL+"/v1/embeddings/generate",
		bytes.NewReader(body),
	)
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("ai engine: status=%d body=%s", resp.StatusCode, string(raw))
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}
