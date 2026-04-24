package hybrid

import (
	"time"

	"aura/backend/core-go/internal/domain/entities"
)

// RankingContext captures the context in which ranking rules are applied.
type RankingContext struct {
	UserProfile entities.UserProfile
	CurrentDate time.Time
	TargetCount int
}

// RankingRule is a single pluggable rule (diversity, recency, etc.).
type RankingRule interface {
	Apply(items []entities.ScoredItem, ctx RankingContext) []entities.ScoredItem
}

// FinalRanker sorts the aggregated scores and applies a chain of rules.
type FinalRanker struct {
	Rules []RankingRule
}

// NewFinalRanker constructs a ranker with the provided rules (order matters).
func NewFinalRanker(rules ...RankingRule) *FinalRanker {
	return &FinalRanker{Rules: rules}
}

// Rank applies every configured rule in order.
func (r *FinalRanker) Rank(items []entities.ScoredItem, ctx RankingContext) []entities.ScoredItem {
	out := items
	for _, rule := range r.Rules {
		out = rule.Apply(out, ctx)
	}
	return out
}
