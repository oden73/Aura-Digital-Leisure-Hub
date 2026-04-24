package cf

import "aura/backend/core-go/internal/domain/entities"

type Coordinator interface {
	GetRecommendations(userID string, limit int) ([]entities.ScoredItem, error)
}

type StubCoordinator struct{}

func (s StubCoordinator) GetRecommendations(_ string, _ int) ([]entities.ScoredItem, error) {
	return []entities.ScoredItem{}, nil
}
