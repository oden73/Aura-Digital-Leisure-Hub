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
	Metadata     postgres.MetadataRepository
	Filter       *filter.Service
}

// NewGetRecommendations wires the dependencies.
func NewGetRecommendations(
	ranker hybrid.Orchestrator,
	userRepo postgres.UserRepository,
	metadata postgres.MetadataRepository,
	filterSvc *filter.Service,
) *GetRecommendations {
	return &GetRecommendations{
		HybridRanker: ranker,
		UserRepo:     userRepo,
		Metadata:     metadata,
		Filter:       filterSvc,
	}
}

// Execute satisfies GetRecommendationsUseCase.
func (u *GetRecommendations) Execute(
	userID string,
	filters entities.RecommendationFilters,
) (RecommendationResponse, error) {
	const defaultLimit = 20

	res, err := u.HybridRanker.GetHybridRecommendations(userID, defaultLimit, filters)
	if err != nil {
		return RecommendationResponse{}, err
	}

	scored := u.Filter.Apply(res.Items, filters)

	items := make([]RecommendationItem, 0, len(scored))
	for _, s := range scored {
		ri := RecommendationItem{
			ItemID: s.ItemID,
			Score:  s.Score,
		}
		if u.Metadata != nil {
			if meta, err := u.Metadata.GetItem(s.ItemID); err == nil {
				ri.Title = meta.Title
			}
		}
		if reason, ok := s.Metadata["match_reason"].(string); ok {
			ri.MatchReason = reason
		}
		items = append(items, ri)
	}
	return RecommendationResponse{
		Items:     items,
		Reasoning: res.Reasoning,
		Metadata:  map[string]any{"user_id": userID},
	}, nil
}
