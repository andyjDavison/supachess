package game

import (
	"api/internal/domain"
	"context"
)

// GameEventPublisher is the seam between Service and the WebSocket layer.
// Service has no idea WebSockets exist — it just announces "this
// happened," and something living in transport/ws is responsible for
// turning that into an actual broadcast to the two connected clients.
type GameEventPublisher interface {
	PublishMoveMade(ctx context.Context, g *domain.Game, m *domain.Move) error
	PublishGameOver(ctx context.Context, g *domain.Game) error
}