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

// SearchQuery bundles all search inputs. JSON tags exist so the same DTO
// can be decoded from a request body if a future endpoint moves search
// from query string to JSON.
type SearchQuery struct {
	Text       string               `json:"q,omitempty"`
	MediaTypes []entities.MediaType `json:"media_types,omitempty"`
	Limit      int                  `json:"limit,omitempty"`
}

// InteractionData is the payload for UpdateInteractionUseCase. The tags
// must match the OpenAPI contract because this DTO is decoded directly
// from the body of PUT /v1/interactions.
type InteractionData struct {
	Status     entities.InteractionStatus `json:"status,omitempty"`
	Rating     int                        `json:"rating,omitempty"`
	IsFavorite bool                       `json:"is_favorite,omitempty"`
	ReviewText string                     `json:"review_text,omitempty"`
}
