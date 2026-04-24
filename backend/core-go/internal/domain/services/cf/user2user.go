package cf

import "aura/backend/core-go/internal/domain/entities"

// UserSimilarityCalculator computes similarity between two users using the
// interaction matrix. Concrete implementations may use Pearson, cosine, etc.
type UserSimilarityCalculator struct {
	Matrix InteractionMatrix
}

// Calculate returns a similarity value in [-1, 1].
func (c UserSimilarityCalculator) Calculate(userU string, userV string) (float64, error) {
	_ = userU
	_ = userV
	_ = c.Matrix
	return 0, nil
}

// UserNeighborhoodBuilder selects the top-k most similar users above a
// similarity threshold alpha.
type UserNeighborhoodBuilder struct {
	ThresholdAlpha float64
	Similarity     UserSimilarityCalculator
}

func (b UserNeighborhoodBuilder) Build(userID string, k int) ([]Neighbor, error) {
	_ = userID
	_ = k
	return nil, nil
}

// UserBasedPredictor predicts a user's rating for an item from neighbours.
type UserBasedPredictor struct {
	Stats UserStatisticsRepository
}

func (p UserBasedPredictor) PredictRating(userID string, itemID string, neighbors []Neighbor) (float64, error) {
	_ = userID
	_ = itemID
	_ = neighbors
	_ = p.Stats
	return 0, nil
}

// User2UserRecommender is the assembled user-based pipeline.
type User2UserRecommender struct {
	Similarity   UserSimilarityCalculator
	Neighborhood UserNeighborhoodBuilder
	Predictor    UserBasedPredictor
}

// ComputeScores satisfies the Recommender interface.
func (r User2UserRecommender) ComputeScores(userID string, candidates []string) ([]entities.ScoredItem, error) {
	_ = userID
	_ = candidates
	return []entities.ScoredItem{}, nil
}
