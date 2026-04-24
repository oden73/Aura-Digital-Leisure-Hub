package cf

import "aura/backend/core-go/internal/domain/entities"

// Coordinator is the public entry point used by higher layers. It picks the
// appropriate CF strategy and returns a unified list of scored items.
type Coordinator interface {
	GetRecommendations(userID string, k int) ([]entities.ScoredItem, error)
	SelectStrategy(profile entities.UserProfile) Strategy
}

// DefaultCoordinator wires the user- and item-based recommenders together.
type DefaultCoordinator struct {
	User2User Recommender
	Item2Item Recommender
}

// NewCoordinator constructs a coordinator with the provided recommenders.
func NewCoordinator(u2u Recommender, i2i Recommender) *DefaultCoordinator {
	return &DefaultCoordinator{User2User: u2u, Item2Item: i2i}
}

// SelectStrategy chooses a strategy based on the profile density.
// The concrete heuristics are TODO; the skeleton returns Hybrid.
func (c *DefaultCoordinator) SelectStrategy(profile entities.UserProfile) Strategy {
	_ = profile
	return StrategyHybrid
}

// GetRecommendations returns collaborative-filtering scores for the user.
func (c *DefaultCoordinator) GetRecommendations(userID string, k int) ([]entities.ScoredItem, error) {
	_ = userID
	_ = k
	return []entities.ScoredItem{}, nil
}
