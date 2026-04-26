package cf

import (
	"math"
	"testing"
)

func TestItemSimilarityCalculator_AdjustedCosine(t *testing.T) {
	// Three users rate items i, j and k. i and j move together (after each
	// user is mean-centered), while k disagrees with j.
	mat := newMatrix(map[string]map[string]float64{
		"u1": {"i": 5, "j": 5, "k": 1},
		"u2": {"i": 4, "j": 4, "k": 2},
		"u3": {"i": 3, "j": 3, "k": 3},
	})

	calc := ItemSimilarityCalculator{Matrix: mat, Stats: mat}

	simIJ, err := calc.Calculate("i", "j")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(simIJ-1) > 1e-9 {
		t.Fatalf("i/j should be perfectly similar, got %v", simIJ)
	}

	simJK, err := calc.Calculate("j", "k")
	if err != nil {
		t.Fatal(err)
	}
	if simJK >= 0 {
		t.Fatalf("j/k should disagree, got %v", simJK)
	}
}

func TestItem2ItemRecommender_PredictsForUnratedCandidate(t *testing.T) {
	// Each neighbour rates a "noise" item so their personal mean is not
	// equal to the rating they gave to a/b/target — required for adjusted
	// cosine to produce a non-zero similarity.
	mat := newMatrix(map[string]map[string]float64{
		"u":  {"a": 5, "b": 5},
		"u1": {"a": 5, "b": 5, "target": 5, "noise": 1},
		"u2": {"a": 4, "b": 4, "target": 4, "noise": 2},
		"u3": {"a": 3, "b": 3, "target": 3, "noise": 1},
	})

	rec := Item2ItemRecommender{
		Similarity:   ItemSimilarityCalculator{Matrix: mat, Stats: mat},
		Neighborhood: ItemNeighborhoodBuilder{Similarity: ItemSimilarityCalculator{Matrix: mat, Stats: mat}},
		Predictor:    ItemBasedPredictor{Matrix: mat},
		K:            5,
	}

	got, err := rec.ComputeScores("u", []string{"target"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 scored item, got %d", len(got))
	}
	if got[0].ItemID != "target" {
		t.Fatalf("unexpected item: %#v", got[0])
	}
	// User rated a=b=5; both neighbours have similarity > 0 to target so
	// the weighted average must be exactly 5.
	if math.Abs(got[0].Score-5) > 1e-6 {
		t.Fatalf("expected score=5, got %v", got[0].Score)
	}
}

func TestItem2ItemRecommender_SkipsAlreadyRated(t *testing.T) {
	mat := newMatrix(map[string]map[string]float64{
		"u":  {"a": 5},
		"u1": {"a": 5, "b": 4},
	})
	rec := Item2ItemRecommender{
		Similarity:   ItemSimilarityCalculator{Matrix: mat, Stats: mat},
		Neighborhood: ItemNeighborhoodBuilder{Similarity: ItemSimilarityCalculator{Matrix: mat, Stats: mat}},
		Predictor:    ItemBasedPredictor{Matrix: mat},
		K:            5,
	}
	got, err := rec.ComputeScores("u", []string{"a"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("expected the already-rated candidate to be filtered, got %#v", got)
	}
}
