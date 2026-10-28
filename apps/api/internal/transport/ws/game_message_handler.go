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

// chatSender is the slice of chat.ChatService this handler calls.
type chatSender interface {
	SendDirect(ctx context.Context, meID, toUserID, body string) (*domain.DirectMessage, error)
	SendGame(ctx context.Context, meID, gameID, body string) (*domain.GameMessage, error)
}

type GameMessageHandler struct {
	games       moveSubmitter
	matchmaking queueJoiner
	invites     inviter
	chat        chatSender
	hub         *Hub
}

func NewGameMessageHandler(games moveSubmitter, matchmaking queueJoiner, invites inviter, chat chatSender, hub *Hub) *GameMessageHandler {
	return &GameMessageHandler{games: games, matchmaking: matchmaking, invites: invites, chat: chat, hub: hub}
}

// Handle is wired up as the onText callback in conn.go — one call per
// incoming WS text message. clientID is already the authenticated user's
// ID, resolved during the upgrade handshake, never trusted from the
// message payload itself.
func (h *GameMessageHandler) Handle(clientID string, message []byte) {
	ctx := context.Background() // no natural per-message context for a raw WS frame

	var env incomingEnvelope
	if err := json.Unmarshal(message, &env); err != nil {
		h.sendError(reply{clientID: clientID}, "invalid_message", "malformed message envelope")
		return
	}

	r := reply{clientID: clientID, inReplyTo: env.Type}
	playerID := clientID

	switch env.Type {
	case "game.submit_move":
		h.handleSubmitMove(ctx, r, env.Payload)
	case "game.resign":
		h.handleResign(ctx, r, env.Payload)
	case "matchmaking.join":
		h.handleJoinQueue(ctx, r, env.Payload)
	case "matchmaking.leave":
		h.handleLeaveQueue(r, env.Payload)
	case "invite.offer":
		h.handleInviteOffer(ctx, r, env.Payload)
	case "invite.accept":
		h.handleInviteAction(r, env.Payload, "accept", func(gameID string) error {
			_, err := h.invites.Accept(ctx, playerID, gameID)
			return err
		})
	case "invite.decline":
		h.handleInviteAction(r, env.Payload, "decline", func(gameID string) error {
			return h.invites.Decline(ctx, playerID, gameID)
		})
	case "invite.cancel":
		h.handleInviteAction(r, env.Payload, "cancel", func(gameID string) error {
			return h.invites.Cancel(ctx, playerID, gameID)
		})
	case "chat.send_direct":
		h.handleSendDirect(ctx, r, env.Payload)
	case "chat.send_game":
		h.handleSendGame(ctx, r, env.Payload)
	default:
		// Covers game.subscribe_game for now — not owned by this
		// handler yet. Logged, not treated as a hard error: a client
		// sending a type this server doesn't handle yet shouldn't get
		// punished for being forward-compatible.
		slog.Debug("unhandled message type", "type", env.Type, "client", clientID)
	}
}

// reply addresses an error back to the connection that sent a message,
// tagged with that message's type. inReplyTo is what lets the client tell
// which of its requests failed - error codes alone are ambiguous once two
// features share one (user_not_found means different things to an invite
// and a DM).
type reply struct {
	clientID  string
	inReplyTo string
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

func (h *GameMessageHandler) handleSubmitMove(ctx context.Context, r reply, raw json.RawMessage) {
	var p submitMovePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		h.sendError(r, "invalid_message", "malformed submit_move payload")
		return
	}

	_, _, err := h.games.SubmitMove(ctx, game.SubmitMoveInput{
		ID:    p.GameID,
		PlayerID:  r.clientID,
		From:      domain.Square(p.From),
		To:        domain.Square(p.To),
		Promotion: parsePromotion(p.Promotion),
	})
	if err != nil {
		h.sendServiceError(r, err)
		return
	}
	// No success response constructed here — GameEventPublisher already
	// broadcast game.move_made to both players as a side effect inside
	// SubmitMove itself.
}

type resignPayload struct {
	GameID string `json:"gameId"`
}

func (h *GameMessageHandler) handleResign(ctx context.Context, r reply, raw json.RawMessage) {
	var p resignPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		h.sendError(r, "invalid_message", "malformed resign payload")
		return
	}

	if _, err := h.games.Resign(ctx, p.GameID, r.clientID); err != nil {
		h.sendServiceError(r, err)
		return
	}
}

func (h *GameMessageHandler) handleJoinQueue(ctx context.Context, r reply, raw json.RawMessage) {
	var p joinQueuePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		h.sendError(r, "invalid_message", "malformed join payload")
		return
	}

	if err := h.matchmaking.JoinQueue(ctx, r.clientID, matchmaking.TimeControlPreset(p.Preset)); err != nil {
		h.sendServiceError(r, err)
	}
}

func (h *GameMessageHandler) handleLeaveQueue(r reply, raw json.RawMessage) {
	var p joinQueuePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		h.sendError(r, "invalid_message", "malformed leave payload")
		return
	}
	h.matchmaking.LeaveQueue(r.clientID, matchmaking.TimeControlPreset(p.Preset))
}

type inviteOfferPayload struct {
	ToUserID string `json:"toUserId"`
	Preset   string `json:"preset"`
}

type inviteGamePayload struct {
	GameID string `json:"gameId"`
}

func (h *GameMessageHandler) handleInviteOffer(ctx context.Context, r reply, raw json.RawMessage) {
	var p inviteOfferPayload
	if err := json.Unmarshal(raw, &p); err != nil || p.ToUserID == "" {
		h.sendError(r, "invalid_message", "malformed invite.offer payload")
		return
	}
	// Success is reported by the invite.sent the service pushes.
	if _, err := h.invites.Offer(ctx, r.clientID, p.ToUserID, matchmaking.TimeControlPreset(p.Preset)); err != nil {
		h.sendServiceError(r, err)
	}
}

// handleInviteAction decodes the {gameId} payload shared by accept,
// decline and cancel. Their success replies (invite.accepted /
// invite.closed) come from the service via InviteNotifier.
func (h *GameMessageHandler) handleInviteAction(r reply, raw json.RawMessage, name string, action func(gameID string) error) {
	var p inviteGamePayload
	if err := json.Unmarshal(raw, &p); err != nil || p.GameID == "" {
		h.sendError(r, "invalid_message", "malformed invite."+name+" payload")
		return
	}
	if err := action(p.GameID); err != nil {
		h.sendServiceError(r, err)
	}
}

type sendDirectPayload struct {
	ToUserID string `json:"toUserId"`
	Body     string `json:"body"`
}

type sendGamePayload struct {
	GameID string `json:"gameId"`
	Body   string `json:"body"`
}

// handleSendDirect and handleSendGame reply only on failure: the message
// itself reaches the sender, like the recipient, through ChatNotifier.
func (h *GameMessageHandler) handleSendDirect(ctx context.Context, r reply, raw json.RawMessage) {
	var p sendDirectPayload
	if err := json.Unmarshal(raw, &p); err != nil || p.ToUserID == "" {
		h.sendError(r, "invalid_message", "malformed chat.send_direct payload")
		return
	}
	if _, err := h.chat.SendDirect(ctx, r.clientID, p.ToUserID, p.Body); err != nil {
		h.sendServiceError(r, err)
	}
}

func (h *GameMessageHandler) handleSendGame(ctx context.Context, r reply, raw json.RawMessage) {
	var p sendGamePayload
	if err := json.Unmarshal(raw, &p); err != nil || p.GameID == "" {
		h.sendError(r, "invalid_message", "malformed chat.send_game payload")
		return
	}
	if _, err := h.chat.SendGame(ctx, r.clientID, p.GameID, p.Body); err != nil {
		h.sendServiceError(r, err)
	}
}

// sendServiceError maps a domain.Err* sentinel to the error codes in
// packages/shared-types/websocket-protocol.md — the WS-layer equivalent
// of writeServiceError in your HTTP handlers.
func (h *GameMessageHandler) sendServiceError(r reply, err error) {
	switch {
	case errors.Is(err, domain.ErrIllegalMove):
		h.sendError(r, "illegal_move", err.Error())
	case errors.Is(err, domain.ErrNotYourTurn):
		h.sendError(r, "not_your_turn", err.Error())
	case errors.Is(err, domain.ErrGameNotFound):
		h.sendError(r, "game_not_found", err.Error())
	case errors.Is(err, domain.ErrGameAlreadyFinished):
		h.sendError(r, "game_already_finished", err.Error())
	case errors.Is(err, domain.ErrCannotFriendSelf):
		h.sendError(r, "cannot_friend_self", err.Error())
	case errors.Is(err, domain.ErrAlreadyFriends):
		h.sendError(r, "already_friends", err.Error())
	case errors.Is(err, domain.ErrFriendRequestExists):
		h.sendError(r, "friend_request_exists", err.Error())
	case errors.Is(err, domain.ErrFriendRequestNotFound):
		h.sendError(r, "friend_request_not_found", err.Error())
	case errors.Is(err, domain.ErrNotFriends):
		h.sendError(r, "not_friends", err.Error())
	case errors.Is(err, domain.ErrInviteNotFound):
		h.sendError(r, "invite_not_found", err.Error())
	case errors.Is(err, domain.ErrInviteAlreadyPending):
		h.sendError(r, "invite_already_pending", err.Error())
	case errors.Is(err, domain.ErrCannotInviteSelf):
		h.sendError(r, "cannot_invite_self", err.Error())
	case errors.Is(err, domain.ErrUserNotFound):
		h.sendError(r, "user_not_found", err.Error())
	case errors.Is(err, domain.ErrUserNotConnected):
		h.sendError(r, "user_not_connected", err.Error())
	case errors.Is(err, domain.ErrUnknownPreset):
		h.sendError(r, "unknown_preset", err.Error())
	case errors.Is(err, domain.ErrAlreadyInGame):
		h.sendError(r, "already_in_game", err.Error())
	case errors.Is(err, domain.ErrConversationNotFound):
		h.sendError(r, "conversation_not_found", err.Error())
	case errors.Is(err, domain.ErrInvalidMessageBody):
		h.sendError(r, "invalid_message_body", err.Error())
	case errors.Is(err, domain.ErrCannotMessageSelf):
		h.sendError(r, "cannot_message_self", err.Error())
	case errors.Is(err, domain.ErrRateLimited):
		h.sendError(r, "rate_limited", err.Error())
	default:
		slog.Error("unhandled service error", "type", r.inReplyTo, "error", err)
		h.sendError(r, "internal_error", "something went wrong")
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

type errorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	// InReplyTo is omitted when the message couldn't even be parsed far
	// enough to know its type.
	InReplyTo string `json:"inReplyTo,omitempty"`
}

func (h *GameMessageHandler) sendError(r reply, code, message string) {
	env := envelope{
		Type:    "error",
		Payload: errorPayload{Code: code, Message: message, InReplyTo: r.inReplyTo},
	}
	data, err := json.Marshal(env)
	if err != nil {
		slog.Error("marshaling error envelope", "error", err)
		return
	}
	_ = h.hub.SendToUser(r.clientID, data)
}