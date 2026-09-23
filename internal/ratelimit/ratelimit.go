// Package ratelimit provides keyed token-bucket limiters used for per-caller
// limits at the gateway and per-product, per-tenant limits toward Trimble.
package ratelimit

import (
	"math"
	"sync"
	"time"
)

// Keyed is a set of token buckets, one per key.
type Keyed struct {
	mu      sync.Mutex
	rate    float64 // tokens per second
	burst   float64
	buckets map[string]*bucket
	now     func() time.Time
	maxKeys int
}

type bucket struct {
	tokens float64
	last   time.Time
}

// NewKeyed returns a limiter allowing rate events per second with the given
// burst for each key.
func NewKeyed(rate float64, burst int) *Keyed {
	return &Keyed{rate: rate, burst: float64(burst), buckets: map[string]*bucket{}, now: time.Now, maxKeys: 10000}
}

// Allow consumes one token for key. When denied it returns the wait until a
// token is available.
func (k *Keyed) Allow(key string) (bool, time.Duration) {
	k.mu.Lock()
	defer k.mu.Unlock()
	now := k.now()
	b, ok := k.buckets[key]
	if !ok {
		if len(k.buckets) >= k.maxKeys {
			k.evict(now)
		}
		b = &bucket{tokens: k.burst, last: now}
		k.buckets[key] = b
	}
	b.tokens = math.Min(k.burst, b.tokens+now.Sub(b.last).Seconds()*k.rate)
	b.last = now
	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}
	wait := time.Duration((1 - b.tokens) / k.rate * float64(time.Second))
	return false, wait
}

// evict removes buckets that have fully refilled; they carry no state.
func (k *Keyed) evict(now time.Time) {
	for key, b := range k.buckets {
		if b.tokens+now.Sub(b.last).Seconds()*k.rate >= k.burst {
			delete(k.buckets, key)
		}
	}
}

// SetClock replaces the time source; tests only.
func (k *Keyed) SetClock(now func() time.Time) { k.now = now }
