package cf

import (
	"math"
	"sort"

	"aura/backend/core-go/internal/domain/entities"
)

// UserSimilarityCalculator computes Pearson correlation between two users
// using their rating maps from the InteractionMatrix.
type UserSimilarityCalculator struct {
	Matrix InteractionMatrix
}

// Calculate returns Pearson correlation in [-1, 1]; 0 means "not enough
// overlap to decide". Users with fewer than two co-rated items get 0.
func (c UserSimilarityCalculator) Calculate(userU string, userV string) (float64, error) {
	if c.Matrix == nil || userU == userV {
		return 0, nil
	}
	ru, err := c.Matrix.GetUserRatings(userU)
	if err != nil {
		return 0, err
	}
	rv, err := c.Matrix.GetUserRatings(userV)
	if err != nil {
		return 0, err
	}
	return pearson(ru, rv), nil
}

// UserNeighborhoodBuilder selects the top-k most similar users above the
// configured similarity threshold alpha.
type UserNeighborhoodBuilder struct {
	ThresholdAlpha float64
	Similarity     UserSimilarityCalculator
}

// Build returns at most k neighbours sorted by similarity desc.
// Negative similarities are dropped and the user itself is excluded.
func (b UserNeighborhoodBuilder) Build(userID string, k int) ([]Neighbor, error) {
	if b.Similarity.Matrix == nil {
		return nil, nil
	}
	users, err := b.Similarity.Matrix.AllUsers()
	if err != nil {
		return nil, err
	}

	cands := make([]Neighbor, 0, len(users))
	for _, v := range users {
		if v == userID {
			continue
		}
		sim, err := b.Similarity.Calculate(userID, v)
		if err != nil {
			return nil, err
		}
		if sim <= b.ThresholdAlpha {
			continue
		}
		cands = append(cands, Neighbor{ID: v, Similarity: sim})
	}

	sort.SliceStable(cands, func(i, j int) bool {
		if cands[i].Similarity == cands[j].Similarity {
			return cands[i].ID < cands[j].ID
		}
		return cands[i].Similarity > cands[j].Similarity
	})
	if k > 0 && len(cands) > k {
		cands = cands[:k]
	}
	return cands, nil
}

// UserBasedPredictor predicts a user's rating for an item from neighbours.
//
// Implements the variance-normalised weighted average from the project spec
// (recomendations.md, section 1.2.2):
//
//	r_hat(u, i) = mean(u) + sigma_u * sum( sim(u,v) * (r(v,i) - mean(v)) / sigma_v ) / sum(|sim(u,v)|)
//
// over neighbours v who have rated i. When no neighbour rated i the prediction
// falls back to mean(u). When sigma_v is zero the neighbour contributes
// nothing (we cannot rescale their deviation).
type UserBasedPredictor struct {
	Stats  UserStatisticsRepository
	Matrix InteractionMatrix
}

// PredictRating implements the σ-normalised mean-centered weighted average.
func (p UserBasedPredictor) PredictRating(userID string, itemID string, neighbors []Neighbor) (float64, error) {
	if p.Matrix == nil || p.Stats == nil {
		return 0, nil
	}
	meanU, err := p.Stats.GetMeanRating(userID)
	if err != nil {
		return 0, err
	}
	varU, err := p.Stats.GetVariance(userID)
	if err != nil {
		return 0, err
	}
	sigmaU := math.Sqrt(varU)

	itemRatings, err := p.Matrix.GetItemRatings(itemID)
	if err != nil {
		return 0, err
	}

	var num, den float64
	for _, n := range neighbors {
		ratingV, ok := itemRatings[n.ID]
		if !ok {
			continue
		}
		meanV, err := p.Stats.GetMeanRating(n.ID)
		if err != nil {
			return 0, err
		}
		varV, err := p.Stats.GetVariance(n.ID)
		if err != nil {
			return 0, err
		}
		sigmaV := math.Sqrt(varV)
		if sigmaV == 0 {
			// Neighbour rates everything the same: their deviation is zero
			// after normalisation, so they cannot contribute information.
			continue
		}
		num += n.Similarity * (ratingV - meanV) / sigmaV
		den += math.Abs(n.Similarity)
	}
	if den == 0 {
		return meanU, nil
	}
	return meanU + sigmaU*(num/den), nil
}

// User2UserRecommender is the assembled user-based pipeline.
type User2UserRecommender struct {
	Similarity   UserSimilarityCalculator
	Neighborhood UserNeighborhoodBuilder
	Predictor    UserBasedPredictor
	K            int
}

// ComputeScores predicts a rating for each candidate and returns scored items.
// A K of 0 falls back to a sensible default of 50 neighbours.
func (r User2UserRecommender) ComputeScores(userID string, candidates []string) ([]entities.ScoredItem, error) {
	if len(candidates) == 0 {
		return []entities.ScoredItem{}, nil
	}
	k := r.K
	if k <= 0 {
		k = 50
	}
	neighbors, err := r.Neighborhood.Build(userID, k)
	if err != nil {
		return nil, err
	}
	if len(neighbors) == 0 {
		return []entities.ScoredItem{}, nil
	}

	out := make([]entities.ScoredItem, 0, len(candidates))
	for _, itemID := range candidates {
		score, err := r.Predictor.PredictRating(userID, itemID, neighbors)
		if err != nil {
			return nil, err
		}
		out = append(out, entities.ScoredItem{
			ItemID: itemID,
			Score:  score,
			Source: entities.ScoreSourceCF,
		})
	}
	return out, nil
}

// pearson returns the Pearson correlation coefficient over the keys present
// in both maps. Returns 0 when the overlap is below 2 or when either side
// has zero variance.
func pearson(a, b map[string]float64) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}

	var (
		common []string
		sumA   float64
		sumB   float64
	)
	for k, va := range a {
		vb, ok := b[k]
		if !ok {
			continue
		}
		common = append(common, k)
		sumA += va
		sumB += vb
	}
	if len(common) < 2 {
		return 0
	}
	n := float64(len(common))
	meanA := sumA / n
	meanB := sumB / n

	var num, denA, denB float64
	for _, k := range common {
		da := a[k] - meanA
		db := b[k] - meanB
		num += da * db
		denA += da * da
		denB += db * db
	}
	if denA == 0 || denB == 0 {
		return 0
	}
	return num / (math.Sqrt(denA) * math.Sqrt(denB))
}
