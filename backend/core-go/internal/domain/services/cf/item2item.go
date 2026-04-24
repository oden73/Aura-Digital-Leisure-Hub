package cf

import "aura/backend/core-go/internal/domain/entities"

// ItemSimilarityCalculator computes similarity between two items using the
// co-occurrence data available in the interaction matrix.
type ItemSimilarityCalculator struct {
	Matrix InteractionMatrix
}

func (c ItemSimilarityCalculator) Calculate(itemI string, itemJ string) (float64, error) {
	_ = itemI
	_ = itemJ
	_ = c.Matrix
	return 0, nil
}

// ItemNeighborhoodBuilder selects the top-k items most similar to a seed item,
// filtering by the similarity threshold beta.
type ItemNeighborhoodBuilder struct {
	ThresholdBeta float64
	Similarity    ItemSimilarityCalculator
}

func (b ItemNeighborhoodBuilder) Build(itemID string, k int) ([]Neighbor, error) {
	_ = itemID
	_ = k
	return nil, nil
}

// ItemBasedPredictor predicts a rating from neighbouring items.
type ItemBasedPredictor struct{}

func (p ItemBasedPredictor) PredictRating(userID string, itemID string, neighbors []Neighbor) (float64, error) {
	_ = userID
	_ = itemID
	_ = neighbors
	return 0, nil
}

// Item2ItemRecommender is the assembled item-based pipeline.
type Item2ItemRecommender struct {
	Similarity   ItemSimilarityCalculator
	Neighborhood ItemNeighborhoodBuilder
	Predictor    ItemBasedPredictor
}

// ComputeScores satisfies the Recommender interface.
func (r Item2ItemRecommender) ComputeScores(userID string, candidates []string) ([]entities.ScoredItem, error) {
	_ = userID
	_ = candidates
	return []entities.ScoredItem{}, nil
}
