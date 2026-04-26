package cf

import (
	"sort"

	"aura/backend/core-go/internal/domain/entities"
)

// CandidateProvider produces a pool of items to score for a given user.
// Typical implementation: return items the user has not rated yet, optionally
// pre-filtered by media type / popularity.
type CandidateProvider interface {
	CandidateItemsForUser(userID string, limit int) ([]string, error)
}

// Coordinator is the public entry point used by higher layers. It picks the
// appropriate CF strategy and returns a unified list of scored items.
type Coordinator interface {
	GetRecommendations(userID string, k int) ([]entities.ScoredItem, error)
	SelectStrategy(profile entities.UserProfile) Strategy
}

// DefaultCoordinator wires the user- and item-based recommenders together.
type DefaultCoordinator struct {
	User2User        Recommender
	Item2Item        Recommender
	Candidates       CandidateProvider
	Stats            UserStatisticsRepository
	Matrix           InteractionMatrix
	CandidatePoolCap int
}

// NewCoordinator constructs a coordinator with the provided recommenders.
func NewCoordinator(u2u Recommender, i2i Recommender) *DefaultCoordinator {
	return &DefaultCoordinator{User2User: u2u, Item2Item: i2i}
}

// WithCandidates attaches the candidate generation strategy.
func (c *DefaultCoordinator) WithCandidates(p CandidateProvider) *DefaultCoordinator {
	c.Candidates = p
	return c
}

// WithMatrix lets the coordinator consult the interaction matrix when
// choosing a strategy (e.g. count of ratings the user already has).
func (c *DefaultCoordinator) WithMatrix(m InteractionMatrix) *DefaultCoordinator {
	c.Matrix = m
	return c
}

// WithStats lets the coordinator consult per-user aggregates when choosing a
// strategy.
func (c *DefaultCoordinator) WithStats(s UserStatisticsRepository) *DefaultCoordinator {
	c.Stats = s
	return c
}

// SelectStrategy picks user-based, item-based or hybrid CF based on profile
// density. Sparse profiles use item-based, mid-density profiles use the
// user-based pipeline, dense profiles run both in parallel.
func (c *DefaultCoordinator) SelectStrategy(profile entities.UserProfile) Strategy {
	if c.Matrix == nil {
		return StrategyHybrid
	}
	ratings, err := c.Matrix.GetUserRatings(profile.UserID)
	if err != nil {
		return StrategyHybrid
	}
	switch n := len(ratings); {
	case n < 5:
		return StrategyItemBased
	case n < 30:
		return StrategyUserBased
	default:
		return StrategyHybrid
	}
}

// GetRecommendations runs candidate generation + the selected CF strategy.
func (c *DefaultCoordinator) GetRecommendations(userID string, k int) ([]entities.ScoredItem, error) {
	if c.Candidates == nil {
		return []entities.ScoredItem{}, nil
	}

	poolSize := c.CandidatePoolCap
	if poolSize <= 0 {
		poolSize = 200
	}
	if k > 0 && poolSize < k*5 {
		poolSize = k * 5
	}

	candidates, err := c.Candidates.CandidateItemsForUser(userID, poolSize)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return []entities.ScoredItem{}, nil
	}

	strategy := c.SelectStrategy(entities.UserProfile{UserID: userID})

	switch strategy {
	case StrategyUserBased:
		return c.runOne(c.User2User, userID, candidates, k)
	case StrategyItemBased:
		return c.runOne(c.Item2Item, userID, candidates, k)
	default:
		return c.runHybrid(userID, candidates, k)
	}
}

func (c *DefaultCoordinator) runOne(r Recommender, userID string, candidates []string, k int) ([]entities.ScoredItem, error) {
	if r == nil {
		return []entities.ScoredItem{}, nil
	}
	scores, err := r.ComputeScores(userID, candidates)
	if err != nil {
		return nil, err
	}
	return topK(scores, k), nil
}

// runHybrid runs both recommenders and merges results by item_id, keeping
// the maximum score so that strong CF signals from either side survive.
func (c *DefaultCoordinator) runHybrid(userID string, candidates []string, k int) ([]entities.ScoredItem, error) {
	merged := make(map[string]entities.ScoredItem)

	if c.User2User != nil {
		s, err := c.User2User.ComputeScores(userID, candidates)
		if err != nil {
			return nil, err
		}
		for _, it := range s {
			merged[it.ItemID] = it
		}
	}
	if c.Item2Item != nil {
		s, err := c.Item2Item.ComputeScores(userID, candidates)
		if err != nil {
			return nil, err
		}
		for _, it := range s {
			if existing, ok := merged[it.ItemID]; !ok || it.Score > existing.Score {
				merged[it.ItemID] = it
			}
		}
	}

	out := make([]entities.ScoredItem, 0, len(merged))
	for _, v := range merged {
		out = append(out, v)
	}
	return topK(out, k), nil
}

func topK(items []entities.ScoredItem, k int) []entities.ScoredItem {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Score == items[j].Score {
			return items[i].ItemID < items[j].ItemID
		}
		return items[i].Score > items[j].Score
	})
	if k > 0 && len(items) > k {
		items = items[:k]
	}
	return items
}
