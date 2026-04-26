package cf

import (
	"math"
	"testing"
)

// memoryMatrix is a tiny in-memory InteractionMatrix used by CF tests.
type memoryMatrix struct {
	users map[string]map[string]float64 // user -> item -> rating
}

func newMatrix(rows map[string]map[string]float64) *memoryMatrix {
	return &memoryMatrix{users: rows}
}

func (m *memoryMatrix) GetUserRatings(uid string) (map[string]float64, error) {
	out := make(map[string]float64, len(m.users[uid]))
	for k, v := range m.users[uid] {
		out[k] = v
	}
	return out, nil
}

func (m *memoryMatrix) GetItemRatings(itemID string) (map[string]float64, error) {
	out := map[string]float64{}
	for uid, items := range m.users {
		if r, ok := items[itemID]; ok {
			out[uid] = r
		}
	}
	return out, nil
}

func (m *memoryMatrix) GetMeanRating(uid string) (float64, error) {
	rs := m.users[uid]
	if len(rs) == 0 {
		return 0, nil
	}
	var s float64
	for _, v := range rs {
		s += v
	}
	return s / float64(len(rs)), nil
}

func (m *memoryMatrix) GetVariance(uid string) (float64, error) {
	rs := m.users[uid]
	if len(rs) < 2 {
		return 0, nil
	}
	mean, _ := m.GetMeanRating(uid)
	var s float64
	for _, v := range rs {
		s += (v - mean) * (v - mean)
	}
	return s / float64(len(rs)-1), nil
}

func (m *memoryMatrix) GetCommonUsers(itemI, itemJ string) ([]string, error) {
	var out []string
	for uid, rs := range m.users {
		if _, a := rs[itemI]; !a {
			continue
		}
		if _, b := rs[itemJ]; !b {
			continue
		}
		out = append(out, uid)
	}
	return out, nil
}

func (m *memoryMatrix) AllUsers() ([]string, error) {
	out := make([]string, 0, len(m.users))
	for k := range m.users {
		out = append(out, k)
	}
	return out, nil
}

func TestPearson_PerfectPositive(t *testing.T) {
	a := map[string]float64{"x": 1, "y": 2, "z": 3}
	b := map[string]float64{"x": 2, "y": 4, "z": 6}
	if got := pearson(a, b); math.Abs(got-1) > 1e-9 {
		t.Fatalf("expected 1.0, got %v", got)
	}
}

func TestPearson_PerfectNegative(t *testing.T) {
	a := map[string]float64{"x": 1, "y": 2, "z": 3}
	b := map[string]float64{"x": 6, "y": 4, "z": 2}
	if got := pearson(a, b); math.Abs(got-(-1)) > 1e-9 {
		t.Fatalf("expected -1.0, got %v", got)
	}
}

func TestPearson_NotEnoughOverlap(t *testing.T) {
	a := map[string]float64{"x": 1, "y": 2}
	b := map[string]float64{"y": 4}
	if got := pearson(a, b); got != 0 {
		t.Fatalf("expected 0, got %v", got)
	}
}

func TestUserNeighborhoodBuilder_TopK(t *testing.T) {
	mat := newMatrix(map[string]map[string]float64{
		"u":  {"i1": 5, "i2": 3, "i3": 4},
		"v1": {"i1": 5, "i2": 3, "i3": 4}, // identical -> sim=1
		"v2": {"i1": 1, "i2": 5, "i3": 1}, // anti-correlated
		"v3": {"i1": 4, "i2": 4, "i3": 4}, // zero variance -> sim=0
	})
	nb := UserNeighborhoodBuilder{
		ThresholdAlpha: 0,
		Similarity:     UserSimilarityCalculator{Matrix: mat},
	}
	got, err := nb.Build("u", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "v1" {
		t.Fatalf("want only v1 above threshold, got %#v", got)
	}
}

func TestUserBasedPredictor_FallsBackToMean(t *testing.T) {
	mat := newMatrix(map[string]map[string]float64{
		"u":  {"i1": 5, "i2": 3},
		"v1": {"i1": 5, "i2": 3},
	})
	p := UserBasedPredictor{Stats: mat, Matrix: mat}
	score, err := p.PredictRating("u", "unknown", []Neighbor{{ID: "v1", Similarity: 1}})
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(score-4) > 1e-9 {
		t.Fatalf("expected mean fallback 4, got %v", score)
	}
}

func TestUser2UserRecommender_ComputeScores(t *testing.T) {
	mat := newMatrix(map[string]map[string]float64{
		"u":  {"a": 5, "b": 3},
		"v1": {"a": 5, "b": 3, "target": 5},
		"v2": {"a": 5, "b": 3, "target": 4},
	})
	rec := User2UserRecommender{
		Similarity:   UserSimilarityCalculator{Matrix: mat},
		Neighborhood: UserNeighborhoodBuilder{Similarity: UserSimilarityCalculator{Matrix: mat}},
		Predictor:    UserBasedPredictor{Stats: mat, Matrix: mat},
		K:            10,
	}
	got, err := rec.ComputeScores("u", []string{"target"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ItemID != "target" {
		t.Fatalf("unexpected output: %#v", got)
	}
	if got[0].Score <= 4 || got[0].Score > 6 {
		t.Fatalf("expected predicted score >4 and <=6 (mean(u)=4 + positive deviation), got %v", got[0].Score)
	}
}
