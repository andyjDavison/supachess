package matchmaking

import (
	"sync"
	"time"
)

type Entry struct {
	PlayerID string
	Rating   int
	Preset   TimeControlPreset
	JoinedAt time.Time
}

// Queue holds waiting players, partitioned by preset so players are
// never matched across different game modes. Safe for concurrent use —
// same reasoning as Hub: touched by many request-handling goroutines at
// once.
type Queue struct {
	mu       sync.Mutex
	byPreset map[TimeControlPreset][]Entry
}

func NewQueue() *Queue {
	return &Queue{byPreset: make(map[TimeControlPreset][]Entry)}
}

// Add enqueues e. Returns false if the player already has an entry
// queued for this preset — join is idempotent, not additive.
func (q *Queue) Add(e Entry) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	for _, existing := range q.byPreset[e.Preset] {
		if existing.PlayerID == e.PlayerID {
			return false
		}
	}
	q.byPreset[e.Preset] = append(q.byPreset[e.Preset], e)
	return true
}

func (q *Queue) Remove(playerID string, preset TimeControlPreset) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.removeLocked(playerID, preset)
}

func (q *Queue) removeLocked(playerID string, preset TimeControlPreset) bool {
	entries := q.byPreset[preset]
	for i, e := range entries {
		if e.PlayerID == playerID {
			q.byPreset[preset] = append(entries[:i], entries[i+1:]...)
			return true
		}
	}
	return false
}

// RemoveFromAll removes playerID from every preset's queue — used on
// disconnect, since the caller there doesn't know which preset (if any)
// they were queued for.
func (q *Queue) RemoveFromAll(playerID string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for preset := range q.byPreset {
		q.removeLocked(playerID, preset)
	}
}

// Entries returns a snapshot copy for preset — safe to range over
// without holding the lock, and safe from a caller mutating it.
func (q *Queue) Entries(preset TimeControlPreset) []Entry {
	q.mu.Lock()
	defer q.mu.Unlock()
	out := make([]Entry, len(q.byPreset[preset]))
	copy(out, q.byPreset[preset])
	return out
}

// Presets returns every preset with at least one waiting entry, so the
// sweep loop only scans non-empty queues.
func (q *Queue) Presets() []TimeControlPreset {
	q.mu.Lock()
	defer q.mu.Unlock()
	presets := make([]TimeControlPreset, 0, len(q.byPreset))
	for p, entries := range q.byPreset {
		if len(entries) > 0 {
			presets = append(presets, p)
		}
	}
	return presets
}