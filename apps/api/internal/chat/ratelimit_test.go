package chat

import (
	"fmt"
	"testing"
	"time"
)

var t0 = time.Unix(1_700_000_000, 0).UTC()

func TestRateLimiterBurstThenRefill(t *testing.T) {
	l := NewRateLimiter(3, 1)

	for i := 0; i < 3; i++ {
		if !l.Allow("u", t0) {
			t.Fatalf("send %d within the burst was refused", i+1)
		}
	}
	if l.Allow("u", t0) {
		t.Fatal("4th send at the same instant was allowed")
	}
	if l.Allow("u", t0.Add(500*time.Millisecond)) {
		t.Error("allowed after half a token's refill")
	}
	if !l.Allow("u", t0.Add(1500*time.Millisecond)) {
		t.Error("refused after a full token refilled")
	}
}

func TestRateLimiterCapsAtCapacity(t *testing.T) {
	l := NewRateLimiter(2, 1)
	l.Allow("u", t0)

	// An hour idle refills to 2, not 3600.
	later := t0.Add(time.Hour)
	allowed := 0
	for i := 0; i < 5; i++ {
		if l.Allow("u", later) {
			allowed++
		}
	}
	if allowed != 2 {
		t.Errorf("allowed %d after a long idle, want capacity (2)", allowed)
	}
}

func TestRateLimiterKeysAreIndependent(t *testing.T) {
	l := NewRateLimiter(1, 1)

	if !l.Allow("a", t0) || !l.Allow("b", t0) {
		t.Error("one key's spend affected another")
	}
}

// A clock stepping backwards must not mint tokens.
func TestRateLimiterIgnoresBackwardsTime(t *testing.T) {
	l := NewRateLimiter(1, 1)
	l.Allow("u", t0)

	if l.Allow("u", t0.Add(-time.Hour)) {
		t.Error("going back in time granted a token")
	}
}

func TestRateLimiterPrunesFullBuckets(t *testing.T) {
	l := NewRateLimiter(1, 1)
	for i := 0; i < pruneThreshold; i++ {
		l.Allow(fmt.Sprintf("user-%d", i), t0)
	}

	l.Allow("newcomer", t0.Add(time.Minute))

	if n := len(l.buckets); n != 1 {
		t.Errorf("%d buckets after pruning, want just the newcomer", n)
	}
}
