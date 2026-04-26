package hybrid

import (
	"math"
	"testing"
	"time"

	"aura/backend/core-go/internal/domain/entities"
)

func TestFinalRanker_SortsByScoreDesc(t *testing.T) {
	r := NewFinalRanker()
	in := []entities.ScoredItem{
		{ItemID: "a", Score: 0.1},
		{ItemID: "b", Score: 0.7},
		{ItemID: "c", Score: 0.3},
	}
	got := r.Rank(in, RankingContext{})
	if got[0].ItemID != "b" || got[1].ItemID != "c" || got[2].ItemID != "a" {
		t.Fatalf("wrong order: %#v", got)
	}
}

func TestFinalRanker_RespectsTargetCount(t *testing.T) {
	r := NewFinalRanker()
	in := []entities.ScoredItem{
		{ItemID: "a", Score: 0.9},
		{ItemID: "b", Score: 0.5},
		{ItemID: "c", Score: 0.1},
	}
	got := r.Rank(in, RankingContext{TargetCount: 2})
	if len(got) != 2 || got[0].ItemID != "a" || got[1].ItemID != "b" {
		t.Fatalf("wrong truncation: %#v", got)
	}
}

func TestRecencyBoostRule_PenalisesOldItems(t *testing.T) {
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	old := time.Date(2014, 1, 1, 0, 0, 0, 0, time.UTC)
	fresh := time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC)

	rule := RecencyBoostRule{DecayFactor: 0.2}
	in := []entities.ScoredItem{
		{ItemID: "old", Score: 1},
		{ItemID: "new", Score: 1},
	}
	ctx := RankingContext{
		CurrentDate: now,
		ItemMeta: map[string]entities.Item{
			"old": {ReleaseDate: &old},
			"new": {ReleaseDate: &fresh},
		},
	}
	out := rule.Apply(in, ctx)
	if out[0].Score >= out[1].Score {
		t.Fatalf("old should score lower than new: %#v", out)
	}
	if out[0].Score >= 1 || out[1].Score > 1 {
		t.Fatalf("scores should be in (0, 1]: %#v", out)
	}
}

func TestDiversityRule_DemotesRepeatedGenre(t *testing.T) {
	rule := DiversityRule{DiversityThreshold: 0.5}
	in := []entities.ScoredItem{
		{ItemID: "a", Score: 1},
		{ItemID: "b", Score: 1},
		{ItemID: "c", Score: 1},
	}
	ctx := RankingContext{
		ItemMeta: map[string]entities.Item{
			"a": {Criteria: entities.BaseItemCriteria{Genre: "Drama"}},
			"b": {Criteria: entities.BaseItemCriteria{Genre: "Drama"}},
			"c": {Criteria: entities.BaseItemCriteria{Genre: "Sci-Fi"}},
		},
	}
	out := rule.Apply(in, ctx)
	if math.Abs(out[0].Score-1) > 1e-9 {
		t.Fatalf("first occurrence not preserved: %v", out[0].Score)
	}
	if !(out[1].Score < out[0].Score) {
		t.Fatalf("second drama should be demoted: %v", out[1].Score)
	}
	if math.Abs(out[2].Score-1) > 1e-9 {
		t.Fatalf("unique genre should be preserved: %v", out[2].Score)
	}
}

func TestDiversityRule_NoMetaPassThrough(t *testing.T) {
	rule := DiversityRule{DiversityThreshold: 0.9}
	in := []entities.ScoredItem{
		{ItemID: "a", Score: 1},
		{ItemID: "b", Score: 0.5},
	}
	out := rule.Apply(in, RankingContext{})
	if out[0].Score != 1 || out[1].Score != 0.5 {
		t.Fatalf("scores should not change without meta: %#v", out)
	}
}

func TestPopularityBalanceRule_InterleavesHeadAndTail(t *testing.T) {
	rule := PopularityBalanceRule{}
	in := []entities.ScoredItem{
		{ItemID: "p1", Score: 0.9},
		{ItemID: "p2", Score: 0.85},
		{ItemID: "n1", Score: 0.7},
		{ItemID: "n2", Score: 0.6},
	}
	ctx := RankingContext{
		ItemMeta: map[string]entities.Item{
			"p1": {AverageRating: 9},
			"p2": {AverageRating: 8.5},
			"n1": {AverageRating: 6},
			"n2": {AverageRating: 5.5},
		},
	}
	out := rule.Apply(in, ctx)
	if len(out) != 4 {
		t.Fatalf("length changed: %d", len(out))
	}
	// First and second items should come from different popularity halves.
	first := ctx.ItemMeta[out[0].ItemID].AverageRating
	second := ctx.ItemMeta[out[1].ItemID].AverageRating
	if !(first >= 8 && second <= 6) {
		t.Fatalf("expected popular then niche, got %v then %v", first, second)
	}
}

func TestFinalRanker_RulesPipelineKeepsDeterministicOrder(t *testing.T) {
	now := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	old := time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC)
	fresh := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	r := NewFinalRanker(
		RecencyBoostRule{DecayFactor: 0.1},
		DiversityRule{DiversityThreshold: 0.3},
	)
	in := []entities.ScoredItem{
		{ItemID: "old-drama", Score: 0.9},
		{ItemID: "new-drama", Score: 0.85},
		{ItemID: "new-scifi", Score: 0.82},
	}
	ctx := RankingContext{
		CurrentDate: now,
		ItemMeta: map[string]entities.Item{
			"old-drama": {ReleaseDate: &old, Criteria: entities.BaseItemCriteria{Genre: "Drama"}},
			"new-drama": {ReleaseDate: &fresh, Criteria: entities.BaseItemCriteria{Genre: "Drama"}},
			"new-scifi": {ReleaseDate: &fresh, Criteria: entities.BaseItemCriteria{Genre: "Sci-Fi"}},
		},
	}
	out := r.Rank(in, ctx)
	if len(out) != 3 {
		t.Fatalf("expected 3, got %d", len(out))
	}
	// Old drama drops via recency, new sci-fi escapes diversity penalty.
	if out[0].ItemID != "new-scifi" {
		t.Fatalf("expected new-scifi on top, got %s (full=%#v)", out[0].ItemID, out)
	}
}
