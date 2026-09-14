package game

import (
	"api/internal/domain"
	"time"
)

type TimerScheduler interface {
	// Schedule (re)arms a timer that fires HandleTimeExpired for
	// expiredColor if remaining elapses with no intervening move.
	Schedule(gameID string, remaining time.Duration, expiredColor domain.Color)
	Cancel(gameID string)
}