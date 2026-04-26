// Package simcache caches pairwise similarity values used by the
// collaborative-filtering pipeline.
//
// CF similarity is the most expensive step on the hot path: for every
// recommendation request the user-based pipeline computes Pearson
// correlation against every other user, and the item-based pipeline
// computes adjusted cosine for every pair of items in the candidate pool.
// Both metrics are stable as long as the underlying ratings don't change,
// so caching them with a TTL + ratings-keyed invalidation cuts orders of
// magnitude of redundant work.
//
// The cache is symmetric: Get(a, b) and Get(b, a) read the same slot,
// because Pearson correlation and adjusted cosine are both symmetric in
// their arguments. Invalidate(id) drops every entry that mentions id on
// either side, so a single rating update by user u, or a re-import of
// item i, evicts only what depends on that id.
package simcache

import (
	"strings"
	"sync"
	"time"
)

// Cache is a TTL'd map of pairwise similarity values keyed by an ordered
// pair of ids. The zero value is not usable; construct with New.
//
// All methods are safe for concurrent use. A nil receiver is a no-op for
// every method, so callers can pass a nil *Cache to disable caching
// without branching at the call site.
type Cache struct {
	mu      sync.RWMutex
	entries map[string]entry
	ttl     time.Duration
	maxSize int
}

type entry struct {
	value     float64
	expiresAt time.Time // zero means "never expires"
}

// New constructs a Cache. A non-positive ttl disables expiry; a
// non-positive maxSize disables size capping (and lets the map grow
// unbounded, only do that for tests).
func New(ttl time.Duration, maxSize int) *Cache {
	return &Cache{
		entries: make(map[string]entry),
		ttl:     ttl,
		maxSize: maxSize,
	}
}

// Get returns the cached similarity for the unordered pair (a, b).
// The second return value indicates a hit — for a miss callers should
// recompute and call Set.
func (c *Cache) Get(a, b string) (float64, bool) {
	if c == nil {
		return 0, false
	}
	key := canonical(a, b)

	c.mu.RLock()
	e, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok {
		return 0, false
	}
	if !e.expiresAt.IsZero() && time.Now().After(e.expiresAt) {
		c.mu.Lock()
		// Re-check under the write lock in case another goroutine
		// already refreshed the slot.
		if cur, still := c.entries[key]; still && cur == e {
			delete(c.entries, key)
		}
		c.mu.Unlock()
		return 0, false
	}
	return e.value, true
}

// Set stores the similarity for the unordered pair (a, b). If the cache
// is at capacity expired entries are dropped first; if it is still at
// capacity a fraction of arbitrary entries is evicted to make room. The
// eviction policy is intentionally not LRU — the map is treated as a
// best-effort cache, not an authoritative store.
func (c *Cache) Set(a, b string, value float64) {
	if c == nil {
		return
	}
	key := canonical(a, b)

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.maxSize > 0 && len(c.entries) >= c.maxSize {
		c.evictLocked()
	}

	var exp time.Time
	if c.ttl > 0 {
		exp = time.Now().Add(c.ttl)
	}
	c.entries[key] = entry{value: value, expiresAt: exp}
}

// Invalidate drops every entry that mentions id on either side. Called by
// the use case layer whenever an interaction (rating) is upserted: the
// user's similarity to every other user, and the item's similarity to
// every other item, may have changed.
func (c *Cache) Invalidate(id string) {
	if c == nil || id == "" {
		return
	}
	prefix := id + "|"
	suffix := "|" + id

	c.mu.Lock()
	defer c.mu.Unlock()
	for k := range c.entries {
		if strings.HasPrefix(k, prefix) || strings.HasSuffix(k, suffix) {
			delete(c.entries, k)
		}
	}
}

// Len returns the number of cached entries (for tests / metrics).
func (c *Cache) Len() int {
	if c == nil {
		return 0
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

func (c *Cache) evictLocked() {
	now := time.Now()
	for k, e := range c.entries {
		if !e.expiresAt.IsZero() && now.After(e.expiresAt) {
			delete(c.entries, k)
		}
	}
	if c.maxSize <= 0 || len(c.entries) < c.maxSize {
		return
	}
	// Cull ~20% of the remaining entries. Map iteration order in Go is
	// randomised, so this is an approximation of "drop something" rather
	// than true LRU; that's acceptable for a similarity cache because
	// every value is recomputable from the database.
	drop := c.maxSize / 5
	if drop < 1 {
		drop = 1
	}
	for k := range c.entries {
		if drop == 0 {
			break
		}
		delete(c.entries, k)
		drop--
	}
}

// canonical returns a stable key for the unordered pair (a, b). Using the
// lexicographic minimum first means Get(a, b) and Get(b, a) hit the same
// slot, which is required for correctness given that Pearson correlation
// and adjusted cosine are both symmetric.
func canonical(a, b string) string {
	if a <= b {
		return a + "|" + b
	}
	return b + "|" + a
}
