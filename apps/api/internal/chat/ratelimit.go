package chat

import (
	"sync"
	"time"
)

// RateLimiter is an in-memory token bucket per key (user ID). Each bucket
// holds up to capacity tokens and refills at refillPerSec; a send spends
// one. Like the matchmaking queue it's per-process, so limits reset on
// restart and aren't shared across instances.
//
// Allow takes now as a parameter, the same reason matchmaking's
// allowedRange does: tests can step time instead of sleeping.
type RateLimiter struct {
	mu           sync.Mutex
	capacity     float64
	refillPerSec float64
	buckets      map[string]*bucket
}

type bucket struct {
	tokens float64
	at     time.Time
}

// pruneThreshold bounds memory: past this many buckets, Allow drops the
// ones that have refilled completely, which are equivalent to no bucket.
const pruneThreshold = 1024

func NewRateLimiter(capacity int, refillPerSec float64) *RateLimiter {
	return &RateLimiter{
		capacity:     float64(capacity),
		refillPerSec: refillPerSec,
		buckets:      map[string]*bucket{},
	}
}

func (l *RateLimiter) Allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	b, ok := l.buckets[key]
	if !ok {
		if len(l.buckets) >= pruneThreshold {
			l.pruneLocked(now)
		}
		b = &bucket{tokens: l.capacity, at: now}
		l.buckets[key] = b
	}

	b.tokens = l.refilled(b, now)
	b.at = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func (l *RateLimiter) refilled(b *bucket, now time.Time) float64 {
	elapsed := now.Sub(b.at).Seconds()
	if elapsed < 0 {
		elapsed = 0
	}
	return min(l.capacity, b.tokens+elapsed*l.refillPerSec)
}

func (l *RateLimiter) pruneLocked(now time.Time) {
	for key, b := range l.buckets {
		if l.refilled(b, now) >= l.capacity {
			delete(l.buckets, key)
		}
	}
}
