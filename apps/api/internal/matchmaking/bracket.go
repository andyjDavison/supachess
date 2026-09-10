package matchmaking

import "time"

// Placeholder constants, per the earlier ballpark figures — worth
// tuning against real usage once there's real usage to tune against.
const (
	baseBracket   = 100
	widenStep     = 50
	widenInterval = 15 * time.Second
	maxBracket    = 400
)

// allowedRange returns the rating range this entry is currently willing
// to be matched against, given how long they've waited. Deliberately a
// pure function — now is a parameter, not read via time.Now() internally
// — so widening is unit-testable without a real sleep: production calls
// this with time.Now(), tests pass a fixed instant.
func allowedRange(rating int, joinedAt, now time.Time) (min, max int) {
	waited := now.Sub(joinedAt)
	bracket := baseBracket + int(waited/widenInterval)*widenStep
	if bracket > maxBracket {
		bracket = maxBracket
	}
	return rating - bracket, rating + bracket
}

// compatible reports whether a and b's currently-allowed ranges permit
// pairing them — each must accept the other, not just one direction.
func compatible(a, b Entry, now time.Time) bool {
	aMin, aMax := allowedRange(a.Rating, a.JoinedAt, now)
	bMin, bMax := allowedRange(b.Rating, b.JoinedAt, now)
	return b.Rating >= aMin && b.Rating <= aMax && a.Rating >= bMin && a.Rating <= bMax
}