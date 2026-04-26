package usecase

import "aura/backend/core-go/internal/domain/entities"

// RecommendationItem is a single entry of the API response (snake_case names
// match the OpenAPI contract in backend/contracts/openapi/core-api.yaml).
type RecommendationItem struct {
	ItemID      string  `json:"item_id"`
	Title       string  `json:"title"`
	Score       float64 `json:"score"`
	MatchReason string  `json:"match_reason,omitempty"`
}

// RecommendationResponse is the shape returned by GetRecommendationsUseCase.
// Keeps the reasoning string and arbitrary metadata as per the class diagram.
type RecommendationResponse struct {
	Items     []RecommendationItem `json:"items"`
	Reasoning string               `json:"reasoning,omitempty"`
	Metadata  map[string]any       `json:"metadata,omitempty"`
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
