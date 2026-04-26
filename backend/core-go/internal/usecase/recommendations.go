package usecase

import (
	"log"

	"aura/backend/core-go/internal/domain/entities"
	"aura/backend/core-go/internal/domain/services/hybrid"
	"aura/backend/core-go/internal/infrastructure/repository/postgres"
	"aura/backend/core-go/internal/pkg/filter"
)

const (
	defaultRecommendationLimit = 20
	matchReasonPopular         = "popular pick"
)

// PopularityRepository returns the highest-rated catalog items, optionally
// restricted by media type. Used by the cold-start fallback to backfill the
// response when CF and CB produced too few candidates.
type PopularityRepository interface {
	TopRated(limit int, mediaTypes []entities.MediaType) ([]entities.Item, error)
}

// GetRecommendations implements GetRecommendationsUseCase by delegating to the
// hybrid ranking orchestrator and applying cross-cutting filters. When the
// personalised pipeline yields too few items (new users, sparse catalog,
// strict filters) it backfills with top-rated items so the API never returns
// an empty list.
type GetRecommendations struct {
	HybridRanker hybrid.Orchestrator
	UserRepo     postgres.UserRepository
	Metadata     postgres.MetadataRepository
	Filter       *filter.Service
	Popularity   PopularityRepository
}

// NewGetRecommendations wires the dependencies. The optional cold-start
// fallback can be attached via WithPopularity.
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

// WithPopularity enables the cold-start fallback. When set, the use case
// pads recommendation responses with top-rated items if the personalised
// pipeline returned fewer than the desired limit.
func (u *GetRecommendations) WithPopularity(p PopularityRepository) *GetRecommendations {
	u.Popularity = p
	return u
}

// Execute satisfies GetRecommendationsUseCase.
func (u *GetRecommendations) Execute(
	userID string,
	filters entities.RecommendationFilters,
) (RecommendationResponse, error) {
	limit := defaultRecommendationLimit

	res, err := u.HybridRanker.GetHybridRecommendations(userID, limit, filters)
	if err != nil {
		return RecommendationResponse{}, err
	}

	scored := u.Filter.Apply(res.Items, filters)
	personalisedCount := len(scored)
	coldStart := len(res.Items) == 0

	popularAdded := 0
	if u.Popularity != nil && len(scored) < limit {
		scored, popularAdded = u.applyColdStartFallback(scored, filters, limit)
	}

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

	meta := map[string]any{
		"user_id":            userID,
		"cold_start":         coldStart,
		"personalised_count": personalisedCount,
	}
	if popularAdded > 0 {
		meta["popular_fallback_count"] = popularAdded
	}

	return RecommendationResponse{
		Items:     items,
		Reasoning: res.Reasoning,
		Metadata:  meta,
	}, nil
}

// applyColdStartFallback backfills `scored` with top-rated items so the
// response reaches `limit` entries. It honours the same media-type filter
// as the personalised request and runs the full Filter.Apply pipeline so
// genre/release/rating constraints are respected.
func (u *GetRecommendations) applyColdStartFallback(
	scored []entities.ScoredItem,
	filters entities.RecommendationFilters,
	limit int,
) ([]entities.ScoredItem, int) {
	missing := limit - len(scored)
	if missing <= 0 {
		return scored, 0
	}

	// Over-fetch to compensate for items that may already be in `scored`
	// or be dropped by the filter pipeline.
	fetchLimit := (missing + len(scored)) * 2
	if fetchLimit < missing*2 {
		fetchLimit = missing * 2
	}

	popular, err := u.Popularity.TopRated(fetchLimit, filters.MediaTypes)
	if err != nil {
		log.Printf("recommendations: cold-start fallback failed: %v", err)
		return scored, 0
	}

	seen := make(map[string]struct{}, len(scored))
	for _, s := range scored {
		seen[s.ItemID] = struct{}{}
	}

	candidates := make([]entities.ScoredItem, 0, len(popular))
	for _, it := range popular {
		if _, dup := seen[it.ID]; dup {
			continue
		}
		candidates = append(candidates, entities.ScoredItem{
			ItemID:   it.ID,
			Score:    it.AverageRating / 10.0,
			Source:   entities.ScoreSourcePopular,
			Metadata: map[string]any{"match_reason": matchReasonPopular},
		})
	}

	candidates = u.Filter.Apply(candidates, filters)

	added := 0
	for _, c := range candidates {
		if len(scored) >= limit {
			break
		}
		if _, dup := seen[c.ItemID]; dup {
			continue
		}
		scored = append(scored, c)
		seen[c.ItemID] = struct{}{}
		added++
	}
	return scored, added
}
