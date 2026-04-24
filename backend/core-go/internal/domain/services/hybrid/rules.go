package hybrid

import "aura/backend/core-go/internal/domain/entities"

// DiversityRule penalises clusters of items from the same genre.
type DiversityRule struct {
	DiversityThreshold float64
}

// Apply is a stub that currently returns the input unchanged; a later
// implementation will re-weight items with saturated genres.
func (r DiversityRule) Apply(items []entities.ScoredItem, _ RankingContext) []entities.ScoredItem {
	return items
}

// RecencyBoostRule boosts items closer to the current date.
type RecencyBoostRule struct {
	DecayFactor float64
}

func (r RecencyBoostRule) Apply(items []entities.ScoredItem, _ RankingContext) []entities.ScoredItem {
	return items
}

// PopularityBalanceRule mixes niche items into an otherwise popular list to
// avoid filter bubbles.
type PopularityBalanceRule struct{}

func (r PopularityBalanceRule) Apply(items []entities.ScoredItem, _ RankingContext) []entities.ScoredItem {
	return items
}
