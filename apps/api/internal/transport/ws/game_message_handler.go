package ws

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"api/internal/domain"
	"api/internal/game"
	"api/internal/matchmaking"
)

// incomingEnvelope mirrors envelope's {type, payload} shape but keeps
// Payload as raw JSON until Type tells us which struct to decode it into.
type incomingEnvelope struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// GameMessageHandler dispatches incoming WS messages to game.Service
// methods — the WS equivalent of AuthHandler, translating between the
// wire protocol and the service's Go types, same "no business logic
// here" discipline.
type GameMessageHandler struct {
	games *game.GameService
	matchmaking *matchmaking.MatchmakingService
	hub   *Hub
}

func NewGameMessageHandler(games *game.GameService, matchmaking *matchmaking.MatchmakingService, hub *Hub) *GameMessageHandler {
	return &GameMessageHandler{games: games, matchmaking: matchmaking, hub: hub}
}

// Handle is wired up as the onText callback in conn.go — one call per
// incoming WS text message. clientID is already the authenticated user's
// ID, resolved during the upgrade handshake, never trusted from the
// message payload itself.
func (h *GameMessageHandler) Handle(clientID string, message []byte) {
	ctx := context.Background() // no natural per-message context for a raw WS frame

	var env incomingEnvelope
	if err := json.Unmarshal(message, &env); err != nil {
		h.sendError(clientID, "invalid_message", "malformed message envelope")
		return
	}

	playerID := clientID

	switch env.Type {
	case "game.submit_move":
		h.handleSubmitMove(ctx, playerID, env.Payload)
	case "game.resign":
		h.handleResign(ctx, playerID, env.Payload)
	case "matchmaking.join":
		h.handleJoinQueue(ctx, playerID, env.Payload)
	case "matchmaking.leave":
		h.handleLeaveQueue(playerID, env.Payload)
	default:
		// Covers game.subscribe_game and every matchmaking.* type for
		// now — neither owned by this handler yet. Logged, not treated
		// as a hard error: a client sending a type this server doesn't
		// handle yet shouldn't get punished for being forward-compatible.
		slog.Debug("unhandled message type", "type", env.Type, "client", clientID)
	}
}

type submitMovePayload struct {
	GameID    string `json:"gameId"`
	From      string `json:"from"`
	To        string `json:"to"`
	Promotion string `json:"promotion,omitempty"`
}

type joinQueuePayload struct {
	Preset string `json:"preset"`
}

func (h *GameMessageHandler) handleSubmitMove(ctx context.Context, playerID string, raw json.RawMessage) {
	var p submitMovePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		h.sendError(playerID, "invalid_message", "malformed submit_move payload")
		return
	}

	_, _, err := h.games.SubmitMove(ctx, game.SubmitMoveInput{
		ID:    p.GameID,
		PlayerID:  playerID,
		From:      domain.Square(p.From),
		To:        domain.Square(p.To),
		Promotion: domain.PieceType(p.Promotion),
	})
	if err != nil {
		h.sendServiceError(string(playerID), err)
		return
	}
	// No success response constructed here — GameEventPublisher already
	// broadcast game.move_made to both players as a side effect inside
	// SubmitMove itself.
}

type resignPayload struct {
	GameID string `json:"gameId"`
}

func (h *GameMessageHandler) handleResign(ctx context.Context, playerID string, raw json.RawMessage) {
	var p resignPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		h.sendError(string(playerID), "invalid_message", "malformed resign payload")
		return
	}

	if _, err := h.games.Resign(ctx, p.GameID, playerID); err != nil {
		h.sendServiceError(string(playerID), err)
		return
	}
}

func (h *GameMessageHandler) handleJoinQueue(ctx context.Context, playerID string, raw json.RawMessage) {
	var p joinQueuePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		h.sendError(string(playerID), "invalid_message", "malformed join payload")
		return
	}

	err := h.matchmaking.JoinQueue(ctx, playerID, matchmaking.TimeControlPreset(p.Preset))
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrUnknownPreset):
			h.sendError(playerID, "unknown_preset", err.Error())
		case errors.Is(err, domain.ErrAlreadyInGame):
			h.sendError(playerID, "already_in_game", err.Error())
		default:
			slog.Error("joining matchmaking queue", "error", err)
			h.sendError(playerID, "internal_error", "something went wrong")
		}
	}
}

func (h *GameMessageHandler) handleLeaveQueue(playerID string, raw json.RawMessage) {
	var p joinQueuePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		h.sendError(playerID, "invalid_message", "malformed leave payload")
		return
	}
	h.matchmaking.LeaveQueue(playerID, matchmaking.TimeControlPreset(p.Preset))
}

// sendServiceError maps a domain.Err* sentinel to the error codes in
// packages/shared-types/websocket-protocol.md — the WS-layer equivalent
// of writeServiceError in your HTTP handlers.
func (h *GameMessageHandler) sendServiceError(clientID string, err error) {
	switch {
	case errors.Is(err, domain.ErrIllegalMove):
		h.sendError(clientID, "illegal_move", err.Error())
	case errors.Is(err, domain.ErrNotYourTurn):
		h.sendError(clientID, "not_your_turn", err.Error())
	case errors.Is(err, domain.ErrGameNotFound):
		h.sendError(clientID, "game_not_found", err.Error())
	case errors.Is(err, domain.ErrGameAlreadyFinished):
		h.sendError(clientID, "game_already_finished", err.Error())
	default:
		slog.Error("unhandled game service error", "error", err)
		h.sendError(clientID, "internal_error", "something went wrong")
	}
}

func (h *GameMessageHandler) sendError(clientID, code, message string) {
	env := envelope{
		Type: "error",
		Payload: struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}{Code: code, Message: message},
	}
	data, err := json.Marshal(env)
	if err != nil {
		slog.Error("marshaling error envelope", "error", err)
		return
	}
	_ = h.hub.SendToUser(clientID, data)
}