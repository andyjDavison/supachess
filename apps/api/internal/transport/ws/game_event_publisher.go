package ws

import (
	"context"
	"encoding/json"
	"fmt"

	"api/internal/domain"
	"api/internal/transport/dto"
)

// envelope matches the {type, payload} shape defined in
// packages/shared-types/websocket-protocol.md — every message in either
// direction uses this outer structure.
type envelope struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

// GameEventPublisher implements game.GameEventPublisher by serializing
// envelopes matching the protocol doc and delivering them via Hub. It
// satisfies that interface structurally — it doesn't import the game
// package, since all it needs are domain types.
type GameEventPublisher struct {
	hub *Hub
}

func NewGameEventPublisher(hub *Hub) *GameEventPublisher {
	return &GameEventPublisher{hub: hub}
}

func (p *GameEventPublisher) PublishMoveMade(_ context.Context, g *domain.Game, m *domain.Move) error {
	env := envelope{
		Type: "game.move_made",
		Payload: struct {
			Game dto.GameDTO `json:"game"`
			Move dto.MoveDTO `json:"move"`
		}{
			Game: dto.NewGameDTO(g),
			Move: dto.NewMoveDTO(m),
		},
	}
	return p.sendToBothPlayers(g, env)
}

func (p *GameEventPublisher) PublishGameOver(_ context.Context, g *domain.Game) error {
	env := envelope{
		Type: "game.game_over",
		Payload: struct {
			Game dto.GameDTO `json:"game"`
		}{
			Game: dto.NewGameDTO(g),
		},
	}
	return p.sendToBothPlayers(g, env)
}

// sendToBothPlayers delivers env to both participants. A player who isn't
// currently connected is not an error worth surfacing loudly — they'll
// get current state via game.subscribe_game (or an equivalent REST fetch)
// on reconnect, per the full-state-not-delta design in the protocol doc.
// Genuine marshal failures are returned, since those indicate a real bug.
func (p *GameEventPublisher) sendToBothPlayers(g *domain.Game, env envelope) error {
	data, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshaling %s envelope: %w", env.Type, err)
	}

	for _, userID := range []string{string(g.WhiteID), string(g.BlackID)} {
		if err := p.hub.SendToUser(userID, data); err != nil {
			// Not connected, or their send buffer was full and they got
			// disconnected (per wsConn.Send's existing behavior) — either
			// way, expected and non-fatal. Not returned as an error.
			continue
		}
	}
	return nil
}