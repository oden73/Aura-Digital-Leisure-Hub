package usecase

import "aura/backend/core-go/internal/domain/entities"

// GetRecommendationsUseCase orchestrates hybrid ranking + filters for a user.
type GetRecommendationsUseCase interface {
	Execute(userID string, filters entities.RecommendationFilters) (RecommendationResponse, error)
}

// SearchContentUseCase performs metadata-driven search with optional ranking.
type SearchContentUseCase interface {
	Execute(query SearchQuery) ([]entities.Item, error)
}

// UpdateInteractionUseCase persists a user interaction and invalidates caches.
type UpdateInteractionUseCase interface {
	Execute(userID string, itemID string, data InteractionData) error
}

// SyncExternalContentUseCase ingests an external item into the catalog.
type SyncExternalContentUseCase interface {
	Execute(externalID string, source entities.ExternalService) (entities.Item, error)
}
