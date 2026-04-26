package cf

import (
	"testing"

	"aura/backend/core-go/internal/domain/entities"
)

type fakeRecommender struct {
	scores []entities.ScoredItem
	err    error
}

func (f fakeRecommender) ComputeScores(_ string, _ []string) ([]entities.ScoredItem, error) {
	return f.scores, f.err
}

type fakeCandidates struct{ items []string }

func (f fakeCandidates) CandidateItemsForUser(_ string, _ int) ([]string, error) {
	return f.items, nil
}

func TestSelectStrategy_Density(t *testing.T) {
	mat := newMatrix(map[string]map[string]float64{
		"sparse": {"a": 5},
		"medium": ratingMap(15),
		"dense":  ratingMap(50),
	})
	c := (&DefaultCoordinator{}).WithMatrix(mat)

	if got := c.SelectStrategy(entities.UserProfile{UserID: "sparse"}); got != StrategyItemBased {
		t.Errorf("sparse: want item-based, got %v", got)
	}
	if got := c.SelectStrategy(entities.UserProfile{UserID: "medium"}); got != StrategyUserBased {
		t.Errorf("medium: want user-based, got %v", got)
	}
	if got := c.SelectStrategy(entities.UserProfile{UserID: "dense"}); got != StrategyHybrid {
		t.Errorf("dense: want hybrid, got %v", got)
	}
}

func TestCoordinator_HybridMergesByMaxScore(t *testing.T) {
	c := NewCoordinator(
		fakeRecommender{scores: []entities.ScoredItem{
			{ItemID: "a", Score: 1, Source: entities.ScoreSourceCF},
			{ItemID: "b", Score: 5, Source: entities.ScoreSourceCF},
		}},
		fakeRecommender{scores: []entities.ScoredItem{
			{ItemID: "a", Score: 4, Source: entities.ScoreSourceCF},
			{ItemID: "c", Score: 3, Source: entities.ScoreSourceCF},
		}},
	).
		WithCandidates(fakeCandidates{items: []string{"a", "b", "c"}}).
		WithMatrix(newMatrix(map[string]map[string]float64{"u": ratingMap(50)}))

	got, err := c.GetRecommendations("u", 10)
	if err != nil {
		t.Fatal(err)
	}
	scoreOf := map[string]float64{}
	for _, it := range got {
		scoreOf[it.ItemID] = it.Score
	}
	if scoreOf["a"] != 4 {
		t.Fatalf("a should keep max score 4, got %v", scoreOf["a"])
	}
	if scoreOf["b"] != 5 || scoreOf["c"] != 3 {
		t.Fatalf("unexpected scores: %#v", scoreOf)
	}
	if got[0].ItemID != "b" {
		t.Fatalf("expected b on top, got %v", got)
	}
}

func TestCoordinator_NoCandidatesShortCircuits(t *testing.T) {
	c := NewCoordinator(fakeRecommender{}, fakeRecommender{}).
		WithCandidates(fakeCandidates{items: nil})
	got, err := c.GetRecommendations("u", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty result, got %#v", got)
	}
}

func ratingMap(n int) map[string]float64 {
	out := make(map[string]float64, n)
	for i := 0; i < n; i++ {
		out[itoa(i)] = float64(1 + (i % 10))
	}
	return out
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	digits := []byte{}
	neg := false
	if i < 0 {
		neg = true
		i = -i
	}
	for i > 0 {
		digits = append([]byte{byte('0' + i%10)}, digits...)
		i /= 10
	}
	if neg {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}
