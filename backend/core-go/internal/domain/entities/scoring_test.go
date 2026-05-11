package entities

import (
	"math"
	"testing"
)

func TestScoredItemCombineWith(t *testing.T) {
	base := ScoredItem{ItemID: "i-1", Score: 0.2, Source: ScoreSourceCF, Metadata: map[string]any{"k": "v"}}
	other := ScoredItem{ItemID: "i-1", Score: 0.8, Source: ScoreSourceCB}

	got := base.CombineWith(other, 0.25)

	if got.ItemID != "i-1" {
		t.Fatalf("item id = %q", got.ItemID)
	}
	if math.Abs(got.Score-0.35) > 1e-9 {
		t.Fatalf("score = %v, want 0.35", got.Score)
	}
	if got.Source != ScoreSourceHybrid {
		t.Fatalf("source = %q", got.Source)
	}
	if got.Metadata["k"] != "v" {
		t.Fatalf("metadata not preserved: %+v", got.Metadata)
	}
}
