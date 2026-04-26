package simcache

import (
	"testing"
	"time"
)

func TestCache_GetSetSymmetric(t *testing.T) {
	c := New(time.Minute, 0)
	c.Set("alice", "bob", 0.42)

	if v, ok := c.Get("alice", "bob"); !ok || v != 0.42 {
		t.Fatalf("Get(alice, bob) = (%v, %v), want (0.42, true)", v, ok)
	}
	if v, ok := c.Get("bob", "alice"); !ok || v != 0.42 {
		t.Fatalf("Get(bob, alice) must hit the same slot, got (%v, %v)", v, ok)
	}
}

func TestCache_NilReceiverIsNoop(t *testing.T) {
	var c *Cache
	if v, ok := c.Get("a", "b"); ok || v != 0 {
		t.Fatal("nil cache must miss everything")
	}
	c.Set("a", "b", 1.0) // must not panic
	c.Invalidate("a")    // must not panic
	if c.Len() != 0 {
		t.Fatal("nil cache must report Len == 0")
	}
}

func TestCache_ExpiresAfterTTL(t *testing.T) {
	c := New(20*time.Millisecond, 0)
	c.Set("a", "b", 1.0)

	if _, ok := c.Get("a", "b"); !ok {
		t.Fatal("expected a fresh entry to be a hit")
	}
	time.Sleep(40 * time.Millisecond)
	if _, ok := c.Get("a", "b"); ok {
		t.Fatal("expected an expired entry to miss")
	}
	if c.Len() != 0 {
		t.Fatalf("Get of an expired entry should remove it, Len = %d", c.Len())
	}
}

func TestCache_InvalidateDropsAllPairsForID(t *testing.T) {
	c := New(time.Minute, 0)
	c.Set("u1", "u2", 0.1)
	c.Set("u1", "u3", 0.2)
	c.Set("u2", "u3", 0.3) // does not mention u1, must survive
	c.Set("u1", "u4", 0.4)

	c.Invalidate("u1")

	if _, ok := c.Get("u1", "u2"); ok {
		t.Fatal("(u1, u2) should be invalidated")
	}
	if _, ok := c.Get("u1", "u3"); ok {
		t.Fatal("(u1, u3) should be invalidated")
	}
	if _, ok := c.Get("u1", "u4"); ok {
		t.Fatal("(u1, u4) should be invalidated")
	}
	if v, ok := c.Get("u2", "u3"); !ok || v != 0.3 {
		t.Fatalf("(u2, u3) must survive, got (%v, %v)", v, ok)
	}
}

func TestCache_EvictionRespectsMaxSize(t *testing.T) {
	c := New(time.Minute, 4)
	c.Set("a", "b", 1)
	c.Set("a", "c", 2)
	c.Set("a", "d", 3)
	c.Set("a", "e", 4)
	// Inserting a 5th entry must keep us at or below maxSize.
	c.Set("a", "f", 5)

	if got := c.Len(); got > 4 {
		t.Fatalf("expected len <= maxSize (4), got %d", got)
	}
}
