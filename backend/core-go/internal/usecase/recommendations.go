package usecase

import (
	"aura/backend/core-go/internal/domain/entities"
	"aura/backend/core-go/internal/domain/services/hybrid"
	"aura/backend/core-go/internal/infrastructure/repository/postgres"
	"aura/backend/core-go/internal/pkg/filter"
)

// GetRecommendations implements GetRecommendationsUseCase by delegating to the
// hybrid ranking orchestrator and applying cross-cutting filters.
type GetRecommendations struct {
	HybridRanker hybrid.Orchestrator
	UserRepo     postgres.UserRepository
	Filter       *filter.Service
}

// NewGetRecommendations wires the dependencies.
func NewGetRecommendations(
	ranker hybrid.Orchestrator,
	userRepo postgres.UserRepository,
	filterSvc *filter.Service,
) *GetRecommendations {
	return &GetRecommendations{
		HybridRanker: ranker,
		UserRepo:     userRepo,
		Filter:       filterSvc,
	}
}

// Execute satisfies GetRecommendationsUseCase.
func (u *GetRecommendations) Execute(
	userID string,
	filters entities.RecommendationFilters,
) (RecommendationResponse, error) {
	const defaultLimit = 20

	scored, err := u.HybridRanker.GetHybridRecommendations(userID, defaultLimit, filters)
	if err != nil {
		return RecommendationResponse{}, err
	}
	scored = u.Filter.Apply(scored, filters)

	items := make([]RecommendationItem, 0, len(scored))
	for _, s := range scored {
		items = append(items, RecommendationItem{
			ItemID: s.ItemID,
			Score:  s.Score,
		})
	}
	return RecommendationResponse{
		Items:    items,
		Metadata: map[string]any{"user_id": userID},
	}, nil
}
