package hybrid

import "aura/backend/core-go/internal/domain/entities"

type Orchestrator interface {
	GetHybridRecommendations(userID string, limit int, filters entities.RecommendationFilters) ([]entities.ScoredItem, error)
}

type StubOrchestrator struct{}

func (s StubOrchestrator) GetHybridRecommendations(_ string, _ int, _ entities.RecommendationFilters) ([]entities.ScoredItem, error) {
	return []entities.ScoredItem{}, nil
}
