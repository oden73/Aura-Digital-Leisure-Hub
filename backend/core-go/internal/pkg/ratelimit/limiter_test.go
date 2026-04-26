package ratelimit

import (
	"testing"
	"time"
)

func TestLimiter_BurstThenRefill(t *testing.T) {
	now := time.Unix(0, 0)
	l := New(10, 3, 0)
	l.now = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		if !l.Allow("u") {
			t.Fatalf("burst slot %d should be allowed", i)
		}
	}
	if l.Allow("u") {
		t.Fatal("4th request must be rejected with empty bucket")
	}

	now = now.Add(200 * time.Millisecond) // 2 tokens at 10rps
	if !l.Allow("u") {
		t.Fatal("after refill the next request should be allowed")
	}
}

func TestLimiter_KeysAreIndependent(t *testing.T) {
	now := time.Unix(0, 0)
	l := New(1, 1, 0)
	l.now = func() time.Time { return now }

	if !l.Allow("alice") {
		t.Fatal("alice should pass")
	}
	if !l.Allow("bob") {
		t.Fatal("bob should pass — independent bucket")
	}
	if l.Allow("alice") {
		t.Fatal("alice should be limited on her own bucket")
	}
}

func TestLimiter_DisabledAlwaysAllows(t *testing.T) {
	l := New(0, 0, 0)
	for i := 0; i < 100; i++ {
		if !l.Allow("anyone") {
			t.Fatal("rate=0 should disable the limiter entirely")
		}
	}
}

func TestLimiter_SweepEvictsIdle(t *testing.T) {
	now := time.Unix(0, 0)
	l := New(1, 1, 5*time.Second)
	l.now = func() time.Time { return now }

	l.Allow("idle")
	l.Allow("active")

	now = now.Add(10 * time.Second)
	l.Allow("active") // refresh active's lastSeen

	l.Sweep()

	if l.Size() != 1 {
		t.Fatalf("expected only the active bucket to survive, size = %d", l.Size())
	}
}

func TestLimiter_SweepDisabledKeepsBuckets(t *testing.T) {
	now := time.Unix(0, 0)
	l := New(1, 1, 0)
	l.now = func() time.Time { return now }

	l.Allow("a")
	now = now.Add(time.Hour)
	l.Sweep()
	if l.Size() != 1 {
		t.Fatal("idleAfter=0 must disable eviction")
	}
}
