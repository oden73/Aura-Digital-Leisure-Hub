package cf

import "aura/backend/core-go/internal/domain/entities"

// Strategy selects which collaborative-filtering flavour is used.
type Strategy string

const (
	StrategyUserBased Strategy = "user_based"
	StrategyItemBased Strategy = "item_based"
	StrategyHybrid    Strategy = "hybrid"
)

// Recommender is the common interface implemented by user- and item-based
// collaborative-filtering recommenders.
type Recommender interface {
	ComputeScores(userID string, candidates []string) ([]entities.ScoredItem, error)
}

// Neighbor is a similarity-weighted neighbour (user or item).
type Neighbor struct {
	ID         string
	Similarity float64
}

// InteractionMatrix exposes the Rui matrix (user/item ratings) to calculators.
type InteractionMatrix interface {
	GetUserRatings(userID string) (map[string]float64, error)
	GetItemRatings(itemID string) (map[string]float64, error)
	GetMeanRating(userID string) (float64, error)
	GetVariance(userID string) (float64, error)
	GetCommonUsers(itemI string, itemJ string) ([]string, error)
}

// UserStatisticsRepository provides aggregates used by the predictors.
type UserStatisticsRepository interface {
	GetMeanRating(userID string) (float64, error)
	GetVariance(userID string) (float64, error)
}
