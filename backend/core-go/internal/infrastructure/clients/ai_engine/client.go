package ai_engine

import "aura/backend/core-go/internal/domain/entities"

type Client interface {
	ComputeCBScores(userID string, candidateIDs []string, limit int) ([]entities.ScoredItem, error)
	GenerateReasoning(userID string, items []entities.ScoredItem) (string, error)
}

type StubClient struct{}

func (s StubClient) ComputeCBScores(_ string, _ []string, _ int) ([]entities.ScoredItem, error) {
	return []entities.ScoredItem{}, nil
}

func (s StubClient) GenerateReasoning(_ string, _ []entities.ScoredItem) (string, error) {
	return "", nil
}
