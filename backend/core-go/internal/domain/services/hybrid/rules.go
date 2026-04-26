package hybrid

import (
	"math"
	"sort"
	"strings"
	"time"

	"aura/backend/core-go/internal/domain/entities"
)

// DiversityRule penalises clusters of items from the same genre by demoting
// every subsequent item that shares its genre with one already accepted.
//
// DiversityThreshold (0..1) controls how aggressively the demotion is applied:
// the n-th occurrence of a genre is multiplied by (1 - threshold)^n.
type DiversityRule struct {
	DiversityThreshold float64
}

// Apply preserves order but multiplies the score of repeating genres.
// Items without metadata in ctx.ItemMeta are passed through unchanged.
func (r DiversityRule) Apply(items []entities.ScoredItem, ctx RankingContext) []entities.ScoredItem {
	if r.DiversityThreshold <= 0 || len(ctx.ItemMeta) == 0 {
		return items
	}
	penalty := 1 - clamp(r.DiversityThreshold, 0, 1)
	seen := make(map[string]int, len(items))
	out := make([]entities.ScoredItem, len(items))
	for i, it := range items {
		out[i] = it
		meta, ok := ctx.ItemMeta[it.ItemID]
		if !ok {
			continue
		}
		genre := strings.ToLower(strings.TrimSpace(meta.Criteria.Genre))
		if genre == "" {
			continue
		}
		if n := seen[genre]; n > 0 {
			out[i].Score *= math.Pow(penalty, float64(n))
		}
		seen[genre]++
	}
	return out
}

// RecencyBoostRule boosts items closer to the current date using exponential
// decay over the age of the item: score *= exp(-DecayFactor * years).
type RecencyBoostRule struct {
	DecayFactor float64
}

// Apply multiplies each score by an age-decay factor when release date is
// known. Items without a release date are returned unchanged.
func (r RecencyBoostRule) Apply(items []entities.ScoredItem, ctx RankingContext) []entities.ScoredItem {
	if r.DecayFactor <= 0 || len(ctx.ItemMeta) == 0 {
		return items
	}
	now := ctx.CurrentDate
	if now.IsZero() {
		now = time.Now()
	}
	out := make([]entities.ScoredItem, len(items))
	for i, it := range items {
		out[i] = it
		meta, ok := ctx.ItemMeta[it.ItemID]
		if !ok || meta.ReleaseDate == nil {
			continue
		}
		years := now.Sub(*meta.ReleaseDate).Hours() / 24 / 365.25
		if years < 0 {
			years = 0
		}
		out[i].Score *= math.Exp(-r.DecayFactor * years)
	}
	return out
}

// PopularityBalanceRule mixes niche items into an otherwise popular list.
//
// We split the input list in half by current score order and interleave the
// two halves so that long-tail picks are surfaced earlier. Items lacking
// metadata still participate using their raw score.
type PopularityBalanceRule struct{}

// Apply returns an interleaved list (head = top, tail = niche). It does not
// mutate scores; it only reorders so that the subsequent stable sort by score
// in FinalRanker keeps the balance whenever niche scores are competitive.
func (r PopularityBalanceRule) Apply(items []entities.ScoredItem, ctx RankingContext) []entities.ScoredItem {
	if len(items) < 4 {
		return items
	}

	popular := make([]entities.ScoredItem, len(items))
	copy(popular, items)
	sort.SliceStable(popular, func(i, j int) bool {
		ai := popularity(ctx, popular[i])
		aj := popularity(ctx, popular[j])
		if ai == aj {
			return popular[i].Score > popular[j].Score
		}
		return ai > aj
	})

	mid := len(popular) / 2
	head := popular[:mid]
	tail := popular[mid:]

	out := make([]entities.ScoredItem, 0, len(items))
	for i := 0; i < mid; i++ {
		out = append(out, head[i])
		if i < len(tail) {
			out = append(out, tail[i])
		}
	}
	for i := mid; i < len(tail); i++ {
		out = append(out, tail[i])
	}
	return out
}

func popularity(ctx RankingContext, it entities.ScoredItem) float64 {
	if meta, ok := ctx.ItemMeta[it.ItemID]; ok {
		return meta.AverageRating
	}
	return it.Score
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
