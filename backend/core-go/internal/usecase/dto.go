package usecase

import "aura/backend/core-go/internal/domain/entities"

// RecommendationItem is a single entry of the API response.
type RecommendationItem struct {
	ItemID      string
	Title       string
	Score       float64
	MatchReason string
}

// RecommendationResponse is the shape returned by GetRecommendationsUseCase.
// Keeps the reasoning string and arbitrary metadata as per the class diagram.
type RecommendationResponse struct {
	Items     []RecommendationItem
	Reasoning string
	Metadata  map[string]any
}

// SearchQuery bundles all search inputs.
type SearchQuery struct {
	Text       string
	MediaTypes []entities.MediaType
	Limit      int
}

// InteractionData is the payload for UpdateInteractionUseCase.
type InteractionData struct {
	Status     entities.InteractionStatus
	Rating     int
	IsFavorite bool
	ReviewText string
}
