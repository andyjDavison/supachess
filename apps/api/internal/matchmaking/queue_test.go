package matchmaking

import (
	"testing"
	"time"
)

func entry(id string, rating int, preset TimeControlPreset) Entry {
	return Entry{PlayerID: id, Rating: rating, Preset: preset, JoinedAt: time.Unix(0, 0).UTC()}
}

func TestQueueAddIsIdempotentPerPreset(t *testing.T) {
	q := NewQueue()

	if !q.Add(entry("alice", 1200, Preset3_0)) {
		t.Fatal("first Add returned false, want true")
	}
	if q.Add(entry("alice", 1200, Preset3_0)) {
		t.Error("second Add for the same preset returned true, want false - join is idempotent, not additive")
	}
	if got := len(q.Entries(Preset3_0)); got != 1 {
		t.Errorf("queue holds %d entries, want 1", got)
	}
}

func TestQueueTracksPresetsIndependently(t *testing.T) {
	q := NewQueue()
	q.Add(entry("alice", 1200, Preset3_0))

	if !q.Add(entry("alice", 1200, Preset5_0)) {
		t.Error("same player in a different preset was rejected, want allowed")
	}
	if got := len(q.Entries(Preset3_0)); got != 1 {
		t.Errorf("3+0 holds %d, want 1", got)
	}
	if got := len(q.Entries(Preset5_0)); got != 1 {
		t.Errorf("5+0 holds %d, want 1", got)
	}
}

func TestQueueRemove(t *testing.T) {
	q := NewQueue()
	q.Add(entry("alice", 1200, Preset3_0))

	if !q.Remove("alice", Preset3_0) {
		t.Error("removing a queued player returned false, want true")
	}
	if q.Remove("alice", Preset3_0) {
		t.Error("removing twice returned true the second time, want false")
	}
	if q.Remove("nobody", Preset3_0) {
		t.Error("removing an absent player returned true, want false")
	}
}

func TestQueueRemoveFromAll(t *testing.T) {
	q := NewQueue()
	q.Add(entry("alice", 1200, Preset3_0))
	q.Add(entry("alice", 1200, Preset10_0))
	q.Add(entry("bob", 1200, Preset3_0))

	q.RemoveFromAll("alice")

	if got := len(q.Entries(Preset3_0)); got != 1 {
		t.Errorf("3+0 holds %d after removing alice, want 1 (bob)", got)
	}
	if got := len(q.Entries(Preset10_0)); got != 0 {
		t.Errorf("10+0 holds %d after removing alice, want 0", got)
	}
}

// Entries hands out a snapshot; a caller mutating it must not corrupt the
// queue, which is what lets attemptMatch range over it without the lock.
func TestQueueEntriesReturnsACopy(t *testing.T) {
	q := NewQueue()
	q.Add(entry("alice", 1200, Preset3_0))

	snapshot := q.Entries(Preset3_0)
	snapshot[0].PlayerID = "mutated"

	if again := q.Entries(Preset3_0); again[0].PlayerID != "alice" {
		t.Errorf("queue entry is now %q - Entries leaked its backing array", again[0].PlayerID)
	}
}

func TestQueuePresetsOnlyListsNonEmpty(t *testing.T) {
	q := NewQueue()
	q.Add(entry("alice", 1200, Preset3_0))
	q.Add(entry("bob", 1200, Preset5_0))
	q.Remove("bob", Preset5_0)

	presets := q.Presets()
	if len(presets) != 1 || presets[0] != Preset3_0 {
		t.Errorf("Presets() = %v, want only [3+0] - an emptied queue should drop out", presets)
	}
}
