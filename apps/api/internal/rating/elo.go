// Package rating holds the Elo arithmetic: which category a time control
// rates, the K-factor, and the change a result makes. Pure functions, no
// I/O - the game service applies them and the repository persists them.
package rating

import (
	"math"
	"time"

	"api/internal/domain"
)

const (
	// ProvisionalGames is how many games in a category a player is
	// provisional for, rated with the larger K so they reach their real
	// strength quickly.
	ProvisionalGames = 30
	ProvisionalK     = 40
	EstablishedK     = 20

	// Floor is the lowest a rating can go.
	Floor = 100
)

// CategoryFor rates a time control by its estimated length, initial + 40
// moves' worth of increment (chess.com's rule): under 3 minutes is Bullet,
// under 10 is Blitz, anything longer Rapid.
func CategoryFor(tc domain.TimeControl) domain.RatingCategory {
	estimate := tc.Initial + 40*tc.Increment
	switch {
	case estimate < 3*time.Minute:
		return domain.CategoryBullet
	case estimate < 10*time.Minute:
		return domain.CategoryBlitz
	default:
		return domain.CategoryRapid
	}
}

// KFactor is the K a player is rated with, given how many games they've
// finished in the category.
func KFactor(gamesPlayed int) int32 {
	if gamesPlayed < ProvisionalGames {
		return ProvisionalK
	}
	return EstablishedK
}

// Delta is the change to rating from scoring score (1 win, 0.5 draw, 0
// loss) against opp: round(K * (score - expected)). It never takes the
// rating below Floor, so the delta reported is the delta applied.
func Delta(rating, opp, k int32, score float64) int32 {
	expected := 1 / (1 + math.Pow(10, float64(opp-rating)/400))
	d := int32(math.Round(float64(k) * (score - expected)))
	if rating+d < Floor {
		d = min(Floor-rating, 0)
	}
	return d
}

// Stakes is what each result would do to a rating, shown before the game.
type Stakes struct {
	Win, Draw, Loss int32
}

func StakesFor(rating, opp, k int32) Stakes {
	return Stakes{
		Win:  Delta(rating, opp, k, 1),
		Draw: Delta(rating, opp, k, 0.5),
		Loss: Delta(rating, opp, k, 0),
	}
}
