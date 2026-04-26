package hybrid

import (
	"sort"
	"time"

	"aura/backend/core-go/internal/domain/entities"
)

// RankingContext captures the context in which ranking rules are applied.
//
// ItemMeta is an optional lookup of catalog metadata for the items being
// ranked. Higher layers (use-cases / orchestrator) populate it before calling
// Rank so that pluggable rules (recency, diversity, popularity, ...) can
// inspect genre / release date / average rating without touching the DB.
type RankingContext struct {
	UserProfile entities.UserProfile
	CurrentDate time.Time
	TargetCount int
	ItemMeta    map[string]entities.Item
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

// Rank applies every configured rule in order and returns items sorted by
// descending score (ties broken by item_id for determinism). The returned
// slice never exceeds ctx.TargetCount when it is positive.
func (r *FinalRanker) Rank(items []entities.ScoredItem, ctx RankingContext) []entities.ScoredItem {
	out := make([]entities.ScoredItem, len(items))
	copy(out, items)

	for _, rule := range r.Rules {
		out = rule.Apply(out, ctx)
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Score == out[j].Score {
			return out[i].ItemID < out[j].ItemID
		}
		return out[i].Score > out[j].Score
	})

	if ctx.TargetCount > 0 && len(out) > ctx.TargetCount {
		out = out[:ctx.TargetCount]
	}
	return out
}
