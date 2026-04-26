// Package ratelimit implements a small token-bucket limiter keyed by an
// arbitrary string (user id, IP, …). It is designed for protecting the
// API edge — not for downstream services — so it lives in-process and
// does not need cross-instance synchronisation. A reverse proxy in front
// of the cluster handles network-level limiting; this layer protects the
// CPU/DB hot path from a single hostile client.
package ratelimit

import (
	"sync"
	"time"
)

// Limiter is a token-bucket rate limiter. The zero value is not usable;
// construct with New. Concurrency-safe.
type Limiter struct {
	rate      float64       // tokens per second
	burst     float64       // bucket capacity
	idleAfter time.Duration // evict bucket if not used for this long

	mu      sync.Mutex
	buckets map[string]*bucket
	now     func() time.Time // injected for tests
}

type bucket struct {
	tokens   float64
	updated  time.Time
	lastSeen time.Time
}

// New constructs a Limiter. rate is requests per second, burst is the
// bucket capacity (and the maximum number of requests that can be
// served back-to-back). idleAfter controls how long an unused bucket
// is kept around before being evicted; pass 0 to disable eviction
// entirely.
func New(rate float64, burst float64, idleAfter time.Duration) *Limiter {
	if burst <= 0 {
		burst = rate
	}
	return &Limiter{
		rate:      rate,
		burst:     burst,
		idleAfter: idleAfter,
		buckets:   make(map[string]*bucket),
		now:       time.Now,
	}
}

// Allow reports whether a request keyed by `key` should be served right
// now. It deducts one token on success. Disabled limiters (rate <= 0)
// always allow.
func (l *Limiter) Allow(key string) bool {
	if l == nil || l.rate <= 0 {
		return true
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	b, ok := l.buckets[key]
	if !ok {
		// New bucket starts full so a single first request is never
		// rejected — the limit is about sustained traffic.
		b = &bucket{tokens: l.burst, updated: now, lastSeen: now}
		l.buckets[key] = b
	} else {
		elapsed := now.Sub(b.updated).Seconds()
		if elapsed > 0 {
			b.tokens += elapsed * l.rate
			if b.tokens > l.burst {
				b.tokens = l.burst
			}
			b.updated = now
		}
		b.lastSeen = now
	}

	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

// Sweep evicts buckets that have not been used for at least IdleAfter.
// Callers can run it on a timer; it's a no-op when eviction is disabled.
func (l *Limiter) Sweep() {
	if l == nil || l.idleAfter <= 0 {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	cutoff := l.now().Add(-l.idleAfter)
	for k, b := range l.buckets {
		if b.lastSeen.Before(cutoff) {
			delete(l.buckets, k)
		}
	}
}

// Size returns the number of tracked buckets (for tests / metrics).
func (l *Limiter) Size() int {
	if l == nil {
		return 0
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.buckets)
}
