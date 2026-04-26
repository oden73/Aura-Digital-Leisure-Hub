package hybrid

import (
	"math"
	"sort"
	"testing"

	"aura/backend/core-go/internal/domain/entities"
)

func TestScoreAggregator_MergesByItemID(t *testing.T) {
	a := NewScoreAggregator(1, 1)

	cf := []entities.ScoredItem{{ItemID: "a", Score: 0.8, Source: entities.ScoreSourceCF}}
	cb := []entities.ScoredItem{
		{ItemID: "a", Score: 0.4, Source: entities.ScoreSourceCB},
		{ItemID: "b", Score: 0.6, Source: entities.ScoreSourceCB},
	}

	got := a.AggregateScores(cf, cb)

	sort.SliceStable(got, func(i, j int) bool { return got[i].ItemID < got[j].ItemID })
	if len(got) != 2 {
		t.Fatalf("expected 2 items, got %d", len(got))
	}
	if got[0].ItemID != "a" || math.Abs(got[0].Score-0.6) > 1e-9 {
		t.Fatalf("a: want score=0.6, got %#v", got[0])
	}
	if got[1].ItemID != "b" || math.Abs(got[1].Score-0.3) > 1e-9 {
		t.Fatalf("b: want score=0.3, got %#v", got[1])
	}
	for _, it := range got {
		if it.Source != entities.ScoreSourceHybrid {
			t.Fatalf("expected hybrid source, got %q for %s", it.Source, it.ItemID)
		}
	}
}

func TestScoreAggregator_ZeroWeightsFallback(t *testing.T) {
	a := NewScoreAggregator(0, 0)

	got := a.AggregateScores(
		[]entities.ScoredItem{{ItemID: "a", Score: 1}},
		[]entities.ScoredItem{{ItemID: "a", Score: 1}},
	)

	if len(got) != 1 {
		t.Fatalf("expected 1 item, got %d", len(got))
	}
	if math.Abs(got[0].Score-1) > 1e-9 {
		t.Fatalf("expected 1.0, got %v", got[0].Score)
	}
}
