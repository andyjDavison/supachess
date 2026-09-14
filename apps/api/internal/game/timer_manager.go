// internal/game/timer_manager.go
package game

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"api/internal/domain"
)

// TimerManager implements TimerScheduler using one time.AfterFunc per
// active game. Known limitation: entirely in-memory - a server restart
// loses every scheduled timer for games already in progress, with no
// rehydration on startup. Flagged, not solved here.
type TimerManager struct {
	mu      sync.Mutex
	timers  map[string]*time.Timer
	gameService *GameService
}

func NewTimerManager(gameService *GameService) *TimerManager {
	return &TimerManager{timers: make(map[string]*time.Timer), gameService: gameService}
}

func (tm *TimerManager) Schedule(gameID string, remaining time.Duration, expiredColor domain.Color) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if existing, ok := tm.timers[gameID]; ok {
		existing.Stop()
	}

	tm.timers[gameID] = time.AfterFunc(remaining, func() {
		ctx := context.Background()
		if _, err := tm.gameService.HandleTimeExpired(ctx, gameID, expiredColor); err != nil {
			slog.Error("handling time expiry", "game", gameID, "error", err)
		}
		tm.mu.Lock()
		delete(tm.timers, gameID)
		tm.mu.Unlock()
	})
}

func (tm *TimerManager) Cancel(gameID string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if existing, ok := tm.timers[gameID]; ok {
		existing.Stop()
		delete(tm.timers, gameID)
	}
}