package cf

import (
	"errors"
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

func TestSelectStrategy_NoMatrixUsesHybrid(t *testing.T) {
	c := NewCoordinator(fakeRecommender{}, fakeRecommender{})
	if got := c.SelectStrategy(entities.UserProfile{UserID: "u"}); got != StrategyHybrid {
		t.Fatalf("want hybrid, got %v", got)
	}
}

func TestSelectStrategy_UserRatingsErrorUsesHybrid(t *testing.T) {
	c := NewCoordinator(fakeRecommender{}, fakeRecommender{}).
		WithMatrix(brokenMatrix{})
	if got := c.SelectStrategy(entities.UserProfile{UserID: "u"}); got != StrategyHybrid {
		t.Fatalf("want hybrid on matrix error, got %v", got)
	}
}

type brokenMatrix struct{}

func (brokenMatrix) GetUserRatings(string) (map[string]float64, error) {
	return nil, errors.New("fail")
}
func (brokenMatrix) GetItemRatings(string) (map[string]float64, error) { return nil, nil }
func (brokenMatrix) GetMeanRating(string) (float64, error)             { return 0, nil }
func (brokenMatrix) GetVariance(string) (float64, error)               { return 0, nil }
func (brokenMatrix) GetCommonUsers(string, string) ([]string, error)   { return nil, nil }
func (brokenMatrix) AllUsers() ([]string, error)                        { return nil, nil }

func TestCoordinator_TopKRespectsLimitAndTieBreak(t *testing.T) {
	c := NewCoordinator(
		fakeRecommender{scores: []entities.ScoredItem{
			{ItemID: "b", Score: 1},
			{ItemID: "a", Score: 1},
			{ItemID: "c", Score: 3},
		}},
		fakeRecommender{},
	).
		WithCandidates(fakeCandidates{items: []string{"a", "b", "c"}}).
		WithMatrix(newMatrix(map[string]map[string]float64{"u": ratingMap(50)}))

	got, err := c.GetRecommendations("u", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 items, got %#v", got)
	}
	if got[0].ItemID != "c" || got[1].ItemID != "a" {
		t.Fatalf("want score order then id tie-break, got %#v", got)
	}
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

func TestCoordinator_WithStatsChains(t *testing.T) {
	c := NewCoordinator(fakeRecommender{}, fakeRecommender{}).
		WithStats(newMatrix(map[string]map[string]float64{"u": {"a": 1}}))
	if c.Stats == nil {
		t.Fatal("stats not attached")
	}
}

func TestCoordinator_ItemBasedUsesItemRecommender(t *testing.T) {
	uScores := []entities.ScoredItem{{ItemID: "bad", Score: 99}}
	iScores := []entities.ScoredItem{{ItemID: "good", Score: 2}}
	c := NewCoordinator(
		fakeRecommender{scores: uScores},
		fakeRecommender{scores: iScores},
	).
		WithCandidates(fakeCandidates{items: []string{"good"}}).
		WithMatrix(newMatrix(map[string]map[string]float64{"u": {"a": 1}}))

	got, err := c.GetRecommendations("u", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ItemID != "good" {
		t.Fatalf("expected item-based branch, got %#v", got)
	}
}

func TestCoordinator_UserBasedUsesUserRecommender(t *testing.T) {
	uScores := []entities.ScoredItem{{ItemID: "picked", Score: 7}}
	iScores := []entities.ScoredItem{{ItemID: "ignored", Score: 1}}
	c := NewCoordinator(
		fakeRecommender{scores: uScores},
		fakeRecommender{scores: iScores},
	).
		WithCandidates(fakeCandidates{items: []string{"picked", "ignored"}}).
		WithMatrix(newMatrix(map[string]map[string]float64{"u": ratingMap(10)}))

	got, err := c.GetRecommendations("u", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ItemID != "picked" {
		t.Fatalf("expected user-based branch, got %#v", got)
	}
}

func TestCoordinator_RunOneNilRecommenderReturnsEmpty(t *testing.T) {
	mat := newMatrix(map[string]map[string]float64{"u": {"a": 1}})
	c := &DefaultCoordinator{
		Item2Item:  nil,
		Candidates: fakeCandidates{items: []string{"x"}},
		Matrix:     mat,
	}
	got, err := c.GetRecommendations("u", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty when recommender nil, got %#v", got)
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
