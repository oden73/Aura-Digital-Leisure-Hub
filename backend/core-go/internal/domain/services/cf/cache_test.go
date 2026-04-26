package cf

import (
	"sync/atomic"
	"testing"
	"time"

	"aura/backend/core-go/internal/pkg/simcache"
)

// countingMatrix wraps memoryMatrix and counts GetUserRatings /
// GetItemRatings calls so we can prove the cache short-circuits the
// expensive lookups.
type countingMatrix struct {
	*memoryMatrix
	userCalls int64
	itemCalls int64
}

func (c *countingMatrix) GetUserRatings(uid string) (map[string]float64, error) {
	atomic.AddInt64(&c.userCalls, 1)
	return c.memoryMatrix.GetUserRatings(uid)
}

func (c *countingMatrix) GetItemRatings(itemID string) (map[string]float64, error) {
	atomic.AddInt64(&c.itemCalls, 1)
	return c.memoryMatrix.GetItemRatings(itemID)
}

func TestUserSimilarity_CacheServesRepeatedCalls(t *testing.T) {
	m := &countingMatrix{memoryMatrix: newMatrix(map[string]map[string]float64{
		"alice": {"i1": 5, "i2": 3, "i3": 4},
		"bob":   {"i1": 4, "i2": 3, "i3": 5},
	})}
	c := simcache.New(time.Minute, 0)
	calc := UserSimilarityCalculator{Matrix: m, Cache: c}

	first, err := calc.Calculate("alice", "bob")
	if err != nil {
		t.Fatalf("first calculate: %v", err)
	}
	beforeSecond := atomic.LoadInt64(&m.userCalls)

	for i := 0; i < 5; i++ {
		// Symmetric calls and repeats must all hit the cache.
		got, err := calc.Calculate("bob", "alice")
		if err != nil {
			t.Fatalf("cached calculate: %v", err)
		}
		if got != first {
			t.Fatalf("cached value drifted: got %v, want %v", got, first)
		}
	}
	after := atomic.LoadInt64(&m.userCalls)
	if after != beforeSecond {
		t.Fatalf("expected zero matrix calls after warm cache, got %d more", after-beforeSecond)
	}
}

func TestUserSimilarity_InvalidationForcesRecompute(t *testing.T) {
	m := &countingMatrix{memoryMatrix: newMatrix(map[string]map[string]float64{
		"alice": {"i1": 5, "i2": 3, "i3": 4},
		"bob":   {"i1": 4, "i2": 3, "i3": 5},
	})}
	c := simcache.New(time.Minute, 0)
	calc := UserSimilarityCalculator{Matrix: m, Cache: c}

	if _, err := calc.Calculate("alice", "bob"); err != nil {
		t.Fatalf("warm: %v", err)
	}
	c.Invalidate("alice")

	before := atomic.LoadInt64(&m.userCalls)
	if _, err := calc.Calculate("alice", "bob"); err != nil {
		t.Fatalf("recompute: %v", err)
	}
	after := atomic.LoadInt64(&m.userCalls)
	if after-before != 2 {
		t.Fatalf("expected 2 matrix calls (one per side) after invalidation, got %d", after-before)
	}
}

func TestItemSimilarity_CacheServesRepeatedCalls(t *testing.T) {
	m := &countingMatrix{memoryMatrix: newMatrix(map[string]map[string]float64{
		"u1": {"a": 5, "b": 4},
		"u2": {"a": 4, "b": 5},
		"u3": {"a": 3, "b": 3},
	})}
	c := simcache.New(time.Minute, 0)
	calc := ItemSimilarityCalculator{Matrix: m, Stats: m.memoryMatrix, Cache: c}

	if _, err := calc.Calculate("a", "b"); err != nil {
		t.Fatalf("warm: %v", err)
	}
	beforeSecond := atomic.LoadInt64(&m.itemCalls)
	for i := 0; i < 3; i++ {
		if _, err := calc.Calculate("b", "a"); err != nil {
			t.Fatalf("cached: %v", err)
		}
	}
	after := atomic.LoadInt64(&m.itemCalls)
	if after != beforeSecond {
		t.Fatalf("expected zero item lookups after warm cache, got %d", after-beforeSecond)
	}
}
