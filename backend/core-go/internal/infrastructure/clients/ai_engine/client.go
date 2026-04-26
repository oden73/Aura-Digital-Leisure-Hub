package ai_engine

import "aura/backend/core-go/internal/domain/entities"

// Request bundles all parameters the Python AI engine expects when computing
// content-based scores for a user.
type Request struct {
	UserID       string
	CandidateIDs []string
	Limit        int
	MediaTypes   []entities.MediaType
}

// Response mirrors the AI engine recommendation response.
type Response struct {
	Items     []entities.ScoredItem
	Reasoning string
}

// EmbeddingRequest describes a single item whose textual representation must
// be embedded and persisted in the vector store.
type EmbeddingRequest struct {
	ItemID string
	Text   string
}

// Client is the abstraction the hybrid orchestrator uses to talk to the AI
// engine. ComputeCB returns the CB scores plus an optional reasoning string
// produced by the LLM. GenerateEmbedding pushes an item's text into the
// vector store so subsequent CB queries can find it.
type Client interface {
	ComputeCB(req Request) (Response, error)
	GenerateReasoning(userID string, items []entities.ScoredItem) (string, error)
	GenerateEmbedding(req EmbeddingRequest) error
}

// StubClient is a no-op client used when the AI engine is unreachable; it
// returns empty results so the rest of the pipeline can still complete.
type StubClient struct{}

func (StubClient) ComputeCB(_ Request) (Response, error) { return Response{}, nil }

func (StubClient) GenerateReasoning(_ string, _ []entities.ScoredItem) (string, error) {
	return "", nil
}

func (StubClient) GenerateEmbedding(_ EmbeddingRequest) error { return nil }
