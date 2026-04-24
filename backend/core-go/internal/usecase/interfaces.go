package usecase

import "aura/backend/core-go/internal/domain/entities"

type GetRecommendationsUseCase interface {
	Execute(userID string, filters entities.RecommendationFilters) ([]entities.ScoredItem, error)
}

type SearchContentUseCase interface {
	Execute(query string) ([]entities.Item, error)
}

type UpdateInteractionUseCase interface {
	Execute(userID string, itemID string, rating int) error
}

type SyncExternalContentUseCase interface {
	Execute(externalID string, source string) (entities.Item, error)
}
