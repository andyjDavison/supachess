package matchmaking

import (
	"testing"
	"time"
)

func TestAllowedRangeWidensWithWaiting(t *testing.T) {
	joined := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name    string
		waited  time.Duration
		bracket int
	}{
		{"on arrival", 0, baseBracket},
		{"just under the first step", widenInterval - time.Millisecond, baseBracket},
		{"one step", widenInterval, baseBracket + widenStep},
		{"two steps", 2 * widenInterval, baseBracket + 2*widenStep},
		{"exactly at the cap", 6 * widenInterval, maxBracket},
		{"past the cap stays capped", 100 * widenInterval, maxBracket},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			min, max := allowedRange(1200, joined, joined.Add(c.waited))
			if min != 1200-c.bracket || max != 1200+c.bracket {
				t.Fatalf("allowedRange after %v = (%d, %d), want (%d, %d)",
					c.waited, min, max, 1200-c.bracket, 1200+c.bracket)
			}
		})
	}
}

func TestCompatibleRequiresBothToAccept(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	// Fresh entries accept +/-100.
	a := Entry{PlayerID: "a", Rating: 1200, JoinedAt: now}

	withinBoth := Entry{PlayerID: "b", Rating: 1250, JoinedAt: now}
	if !compatible(a, withinBoth, now) {
		t.Error("1200 and 1250 are inside each other's range, want compatible")
	}

	outsideBoth := Entry{PlayerID: "c", Rating: 1500, JoinedAt: now}
	if compatible(a, outsideBoth, now) {
		t.Error("1200 and 1500 are 300 apart on fresh brackets, want incompatible")
	}
}

// The asymmetric case is the one a single-sided check would get wrong: the
// long-waiting player's range has widened enough to accept the newcomer,
// but the newcomer's has not widened to accept them back.
func TestCompatibleRejectsOneSidedAcceptance(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	patient := Entry{PlayerID: "patient", Rating: 1200, JoinedAt: now.Add(-4 * widenInterval)} // +/-300
	fresh := Entry{PlayerID: "fresh", Rating: 1450, JoinedAt: now}                             // +/-100

	if pMin, pMax := allowedRange(patient.Rating, patient.JoinedAt, now); fresh.Rating < pMin || fresh.Rating > pMax {
		t.Fatalf("precondition: patient should accept 1450, range was (%d, %d)", pMin, pMax)
	}
	if compatible(patient, fresh, now) {
		t.Error("only the patient player accepts the pairing, want incompatible")
	}
}
