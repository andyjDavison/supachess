package ws

import (
	"context"
	"encoding/json"
	"fmt"

	"api/internal/domain"
)

// MatchmakingNotifier implements matchmaking.Notifier — the matchmaking
// equivalent of GameEventPublisher, same structural-typing relationship
// (it satisfies the interface without matchmaking importing this
// package, avoiding a cycle).
type MatchmakingNotifier struct {
	hub *Hub
}

func NewMatchmakingNotifier(hub *Hub) *MatchmakingNotifier {
	return &MatchmakingNotifier{hub: hub}
}

func (n *MatchmakingNotifier) NotifyMatched(_ context.Context, playerID string, g *domain.Game, opponent *domain.User) error {
	env := envelope{
		Type: "matchmaking.matched",
		Payload: struct {
			GameID   string `json:"gameId"`
			Opponent struct {
				Username string `json:"username"`
				Rating   int    `json:"rating"`
			} `json:"opponent"`
		}{
			GameID: string(g.ID),
			Opponent: struct {
				Username string `json:"username"`
				Rating   int    `json:"rating"`
			}{Username: opponent.Username, Rating: int(opponent.Rating)},
		},
	}

	data, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshaling matched envelope: %w", err)
	}
	return n.hub.SendToUser(string(playerID), data)
}