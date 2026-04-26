package cf

import (
	"math"
	"sort"

	"aura/backend/core-go/internal/domain/entities"
	"aura/backend/core-go/internal/pkg/simcache"
)

// ItemSimilarityCalculator computes similarity between two items using the
// adjusted cosine formula on co-rating users.
//
// Spec reference: docs/predone/recomendations.md, section 1.2.3, formula
//
//	s(i, j) = sum_{u in U_i ∩ U_j} (r_ui - mean_u) * (r_uj - mean_u)
//	          / ( sqrt(sum (r_ui - mean_u)^2) * sqrt(sum (r_uj - mean_u)^2) )
//
// NB: the spec PDF prints the numerator with a single deviation term; that
// is treated as a typesetting error — the canonical adjusted cosine has the
// product of the two deviations in the numerator (otherwise the measure is
// not symmetric in i, j and degenerates). We implement the canonical form.
type ItemSimilarityCalculator struct {
	Matrix InteractionMatrix
	Stats  UserStatisticsRepository
	Cache  *simcache.Cache // optional; nil disables caching
}

// Calculate returns adjusted cosine similarity in [-1, 1].
// Returns 0 when the items have fewer than two common users or when either
// side has zero variance among the common users.
func (c ItemSimilarityCalculator) Calculate(itemI string, itemJ string) (float64, error) {
	if c.Matrix == nil || itemI == itemJ {
		return 0, nil
	}
	if v, ok := c.Cache.Get(itemI, itemJ); ok {
		return v, nil
	}

	ri, err := c.Matrix.GetItemRatings(itemI)
	if err != nil {
		return 0, err
	}
	rj, err := c.Matrix.GetItemRatings(itemJ)
	if err != nil {
		return 0, err
	}

	var num, denI, denJ float64
	common := 0
	for uid, rui := range ri {
		ruj, ok := rj[uid]
		if !ok {
			continue
		}
		mean, err := c.userMean(uid)
		if err != nil {
			return 0, err
		}
		di := rui - mean
		dj := ruj - mean
		num += di * dj
		denI += di * di
		denJ += dj * dj
		common++
	}
	if common < 2 || denI == 0 || denJ == 0 {
		c.Cache.Set(itemI, itemJ, 0)
		return 0, nil
	}
	val := num / (math.Sqrt(denI) * math.Sqrt(denJ))
	c.Cache.Set(itemI, itemJ, val)
	return val, nil
}

func (c ItemSimilarityCalculator) userMean(userID string) (float64, error) {
	if c.Stats != nil {
		return c.Stats.GetMeanRating(userID)
	}
	return c.Matrix.GetMeanRating(userID)
}

// ItemNeighborhoodBuilder selects the top-k items most similar to a seed item,
// constrained to a candidate pool (items the user has actually rated, in the
// item-based use-case). Filters by similarity threshold beta.
type ItemNeighborhoodBuilder struct {
	ThresholdBeta float64
	Similarity    ItemSimilarityCalculator
}

// Build returns at most k neighbours for the seed item from the provided pool.
// Negative similarities are dropped and the seed is excluded from the result.
func (b ItemNeighborhoodBuilder) Build(itemID string, pool []string, k int) ([]Neighbor, error) {
	cands := make([]Neighbor, 0, len(pool))
	for _, candidate := range pool {
		if candidate == itemID {
			continue
		}
		sim, err := b.Similarity.Calculate(itemID, candidate)
		if err != nil {
			return nil, err
		}
		if sim <= b.ThresholdBeta {
			continue
		}
		cands = append(cands, Neighbor{ID: candidate, Similarity: sim})
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

// ItemBasedPredictor predicts a user's rating for a target item from
// neighbouring items the user has already rated.
type ItemBasedPredictor struct {
	Matrix InteractionMatrix
}

// PredictRating implements the item-item formula from the spec
// (recomendations.md, section 1.2.3):
//
//	r_hat(u, i) = sum_{j in N(i)} sim(i, j) * r_uj
//	            / sum_{j in N(i)} |sim(i, j)|
//
// over neighbour items j the user has rated. Returns 0 when no neighbour is
// usable (caller decides on a fallback).
func (p ItemBasedPredictor) PredictRating(userID string, itemID string, neighbors []Neighbor) (float64, error) {
	if p.Matrix == nil {
		return 0, nil
	}
	userRatings, err := p.Matrix.GetUserRatings(userID)
	if err != nil {
		return 0, err
	}
	var num, den float64
	for _, n := range neighbors {
		ruj, ok := userRatings[n.ID]
		if !ok {
			continue
		}
		num += n.Similarity * ruj
		den += math.Abs(n.Similarity)
	}
	if den == 0 {
		return 0, nil
	}
	return num / den, nil
}

// Item2ItemRecommender is the assembled item-based pipeline.
type Item2ItemRecommender struct {
	Similarity   ItemSimilarityCalculator
	Neighborhood ItemNeighborhoodBuilder
	Predictor    ItemBasedPredictor
	K            int
}

// ComputeScores predicts a rating for each candidate item using the user's
// own rated items as the neighbourhood pool. Skips candidates the user
// already rated and items for which the predictor cannot produce a number.
func (r Item2ItemRecommender) ComputeScores(userID string, candidates []string) ([]entities.ScoredItem, error) {
	if len(candidates) == 0 {
		return []entities.ScoredItem{}, nil
	}
	if r.Predictor.Matrix == nil {
		return []entities.ScoredItem{}, nil
	}

	userRatings, err := r.Predictor.Matrix.GetUserRatings(userID)
	if err != nil {
		return nil, err
	}
	if len(userRatings) == 0 {
		return []entities.ScoredItem{}, nil
	}
	pool := make([]string, 0, len(userRatings))
	for itemID := range userRatings {
		pool = append(pool, itemID)
	}

	k := r.K
	if k <= 0 {
		k = 50
	}

	out := make([]entities.ScoredItem, 0, len(candidates))
	for _, itemID := range candidates {
		if _, alreadyRated := userRatings[itemID]; alreadyRated {
			continue
		}
		neighbors, err := r.Neighborhood.Build(itemID, pool, k)
		if err != nil {
			return nil, err
		}
		if len(neighbors) == 0 {
			continue
		}
		score, err := r.Predictor.PredictRating(userID, itemID, neighbors)
		if err != nil {
			return nil, err
		}
		if score == 0 {
			continue
		}
		out = append(out, entities.ScoredItem{
			ItemID: itemID,
			Score:  score,
			Source: entities.ScoreSourceCF,
		})
	}
	return out, nil
}
