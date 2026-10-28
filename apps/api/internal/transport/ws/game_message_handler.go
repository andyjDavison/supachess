package ws

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"api/internal/domain"
	"api/internal/game"
	"api/internal/invite"
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
// moveSubmitter and queueJoiner are the slices of GameService and
// MatchmakingService this handler calls. Narrow interfaces keep the
// domain-error-to-wire-code mapping in sendServiceError testable without
// a repository or a live queue behind it.
type moveSubmitter interface {
	SubmitMove(ctx context.Context, input game.SubmitMoveInput) (*domain.Move, *domain.Game, error)
	Resign(ctx context.Context, gameID string, resignationID string) (*domain.Game, error)
}

type queueJoiner interface {
	JoinQueue(ctx context.Context, playerID string, preset matchmaking.TimeControlPreset) error
	LeaveQueue(playerID string, preset matchmaking.TimeControlPreset)
}

// inviter is the slice of invite.InviteService this handler calls.
type inviter interface {
	Offer(ctx context.Context, meID, toUserID string, preset matchmaking.TimeControlPreset) (*invite.Invite, error)
	Accept(ctx context.Context, meID, gameID string) (*domain.Game, error)
	Decline(ctx context.Context, meID, gameID string) error
	Cancel(ctx context.Context, meID, gameID string) error
}

type GameMessageHandler struct {
	games       moveSubmitter
	matchmaking queueJoiner
	invites     inviter
	hub         *Hub
}

func NewGameMessageHandler(games moveSubmitter, matchmaking queueJoiner, invites inviter, hub *Hub) *GameMessageHandler {
	return &GameMessageHandler{games: games, matchmaking: matchmaking, invites: invites, hub: hub}
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
	case "invite.offer":
		h.handleInviteOffer(ctx, playerID, env.Payload)
	case "invite.accept":
		h.handleInviteAction(playerID, env.Payload, "accept", func(gameID string) error {
			_, err := h.invites.Accept(ctx, playerID, gameID)
			return err
		})
	case "invite.decline":
		h.handleInviteAction(playerID, env.Payload, "decline", func(gameID string) error {
			return h.invites.Decline(ctx, playerID, gameID)
		})
	case "invite.cancel":
		h.handleInviteAction(playerID, env.Payload, "cancel", func(gameID string) error {
			return h.invites.Cancel(ctx, playerID, gameID)
		})
	default:
		// Covers game.subscribe_game for now — not owned by this
		// handler yet. Logged, not treated
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
		Promotion: parsePromotion(p.Promotion),
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

	if err := h.matchmaking.JoinQueue(ctx, playerID, matchmaking.TimeControlPreset(p.Preset)); err != nil {
		h.sendServiceError(playerID, err)
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

type inviteOfferPayload struct {
	ToUserID string `json:"toUserId"`
	Preset   string `json:"preset"`
}

type inviteGamePayload struct {
	GameID string `json:"gameId"`
}

func (h *GameMessageHandler) handleInviteOffer(ctx context.Context, playerID string, raw json.RawMessage) {
	var p inviteOfferPayload
	if err := json.Unmarshal(raw, &p); err != nil || p.ToUserID == "" {
		h.sendError(playerID, "invalid_message", "malformed invite.offer payload")
		return
	}
	// Success is reported by the invite.sent the service pushes.
	if _, err := h.invites.Offer(ctx, playerID, p.ToUserID, matchmaking.TimeControlPreset(p.Preset)); err != nil {
		h.sendServiceError(playerID, err)
	}
}

// handleInviteAction decodes the {gameId} payload shared by accept,
// decline and cancel. Their success replies (invite.accepted /
// invite.closed) come from the service via InviteNotifier.
func (h *GameMessageHandler) handleInviteAction(playerID string, raw json.RawMessage, name string, action func(gameID string) error) {
	var p inviteGamePayload
	if err := json.Unmarshal(raw, &p); err != nil || p.GameID == "" {
		h.sendError(playerID, "invalid_message", "malformed invite."+name+" payload")
		return
	}
	if err := action(p.GameID); err != nil {
		h.sendServiceError(playerID, err)
	}
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
	case errors.Is(err, domain.ErrCannotFriendSelf):
		h.sendError(clientID, "cannot_friend_self", err.Error())
	case errors.Is(err, domain.ErrAlreadyFriends):
		h.sendError(clientID, "already_friends", err.Error())
	case errors.Is(err, domain.ErrFriendRequestExists):
		h.sendError(clientID, "friend_request_exists", err.Error())
	case errors.Is(err, domain.ErrFriendRequestNotFound):
		h.sendError(clientID, "friend_request_not_found", err.Error())
	case errors.Is(err, domain.ErrNotFriends):
		h.sendError(clientID, "not_friends", err.Error())
	case errors.Is(err, domain.ErrInviteNotFound):
		h.sendError(clientID, "invite_not_found", err.Error())
	case errors.Is(err, domain.ErrInviteAlreadyPending):
		h.sendError(clientID, "invite_already_pending", err.Error())
	case errors.Is(err, domain.ErrCannotInviteSelf):
		h.sendError(clientID, "cannot_invite_self", err.Error())
	case errors.Is(err, domain.ErrUserNotFound):
		h.sendError(clientID, "user_not_found", err.Error())
	case errors.Is(err, domain.ErrUserNotConnected):
		h.sendError(clientID, "user_not_connected", err.Error())
	case errors.Is(err, domain.ErrUnknownPreset):
		h.sendError(clientID, "unknown_preset", err.Error())
	case errors.Is(err, domain.ErrAlreadyInGame):
		h.sendError(clientID, "already_in_game", err.Error())
	default:
		slog.Error("unhandled game service error", "error", err)
		h.sendError(clientID, "internal_error", "something went wrong")
	}
}

func parsePromotion(s string) domain.PieceType {
	switch s {
	case "q":
		return domain.Queen
	case "r":
		return domain.Rook
	case "b":
		return domain.Bishop
	case "n":
		return domain.Knight
	default:
		return ""
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