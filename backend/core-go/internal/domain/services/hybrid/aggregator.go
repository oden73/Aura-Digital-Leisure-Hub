package hybrid

import "aura/backend/core-go/internal/domain/entities"

// ScoreAggregator merges CF and CB scores into a single ranked list using
// configurable weights.
type ScoreAggregator struct {
	WeightCF float64
	WeightCB float64
}

// NewScoreAggregator constructs an aggregator with weights that need not sum
// to 1 (they are normalised inside AggregateScores).
func NewScoreAggregator(weightCF, weightCB float64) *ScoreAggregator {
	return &ScoreAggregator{WeightCF: weightCF, WeightCB: weightCB}
}

// AggregateScores merges two score slices by item_id.
func (a *ScoreAggregator) AggregateScores(
	cf []entities.ScoredItem,
	cb []entities.ScoredItem,
) []entities.ScoredItem {
	combined := make(map[string]entities.ScoredItem, len(cf)+len(cb))

	totalWeight := a.WeightCF + a.WeightCB
	if totalWeight == 0 {
		totalWeight = 1
	}
	cfW := a.WeightCF / totalWeight
	cbW := a.WeightCB / totalWeight

	for _, s := range cf {
		existing := combined[s.ItemID]
		existing.ItemID = s.ItemID
		existing.Score += s.Score * cfW
		existing.Source = entities.ScoreSourceHybrid
		combined[s.ItemID] = existing
	}
	for _, s := range cb {
		existing := combined[s.ItemID]
		existing.ItemID = s.ItemID
		existing.Score += s.Score * cbW
		existing.Source = entities.ScoreSourceHybrid
		combined[s.ItemID] = existing
	}

	result := make([]entities.ScoredItem, 0, len(combined))
	for _, v := range combined {
		result = append(result, v)
	}
	return result
}
