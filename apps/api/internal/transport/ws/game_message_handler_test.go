package ws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"api/internal/domain"
	"api/internal/game"
	"api/internal/invite"
	"api/internal/matchmaking"
)

// --- doubles for the handler's two service seams ---

type fakeMoves struct {
	submitted []game.SubmitMoveInput
	submitErr error
	resigned  []string
	resignErr error
}

func (f *fakeMoves) SubmitMove(_ context.Context, in game.SubmitMoveInput) (*domain.Move, *domain.Game, error) {
	f.submitted = append(f.submitted, in)
	if f.submitErr != nil {
		return nil, nil, f.submitErr
	}
	return &domain.Move{GameID: in.ID}, &domain.Game{ID: in.ID}, nil
}

func (f *fakeMoves) Resign(_ context.Context, gameID string, resignationID string) (*domain.Game, error) {
	f.resigned = append(f.resigned, gameID+":"+resignationID)
	if f.resignErr != nil {
		return nil, f.resignErr
	}
	return &domain.Game{ID: gameID}, nil
}

type fakeQueue struct {
	joined  []string
	joinErr error
	left    []string
}

func (f *fakeQueue) JoinQueue(_ context.Context, playerID string, preset matchmaking.TimeControlPreset) error {
	f.joined = append(f.joined, playerID+":"+string(preset))
	return f.joinErr
}

func (f *fakeQueue) LeaveQueue(playerID string, preset matchmaking.TimeControlPreset) {
	f.left = append(f.left, playerID+":"+string(preset))
}

// fakeInvites records each call as "method:me:arg" and returns err.
type fakeInvites struct {
	calls []string
	err   error
}

func (f *fakeInvites) Offer(_ context.Context, meID, toUserID string, preset matchmaking.TimeControlPreset) (*invite.Invite, error) {
	f.calls = append(f.calls, "offer:"+meID+":"+toUserID+":"+string(preset))
	return nil, f.err
}

func (f *fakeInvites) Accept(_ context.Context, meID, gameID string) (*domain.Game, error) {
	f.calls = append(f.calls, "accept:"+meID+":"+gameID)
	return nil, f.err
}

func (f *fakeInvites) Decline(_ context.Context, meID, gameID string) error {
	f.calls = append(f.calls, "decline:"+meID+":"+gameID)
	return f.err
}

func (f *fakeInvites) Cancel(_ context.Context, meID, gameID string) error {
	f.calls = append(f.calls, "cancel:"+meID+":"+gameID)
	return f.err
}

type fakeChat struct {
	calls []string // "direct:me:to:body" / "game:me:gameID:body"
	err   error
}

func (f *fakeChat) SendDirect(_ context.Context, meID, toUserID, body string) (*domain.DirectMessage, error) {
	f.calls = append(f.calls, "direct:"+meID+":"+toUserID+":"+body)
	return nil, f.err
}

func (f *fakeChat) SendGame(_ context.Context, meID, gameID, body string) (*domain.GameMessage, error) {
	f.calls = append(f.calls, "game:"+meID+":"+gameID+":"+body)
	return nil, f.err
}

type handlerHarness struct {
	handler *GameMessageHandler
	moves   *fakeMoves
	queue   *fakeQueue
	invites *fakeInvites
	chat    *fakeChat
	client  *fakeClient
}

func newHandlerHarness() *handlerHarness {
	hub := NewHub()
	client := &fakeClient{id: "player-1"}
	hub.Register(client)
	moves := &fakeMoves{}
	queue := &fakeQueue{}
	invites := &fakeInvites{}
	chat := &fakeChat{}
	return &handlerHarness{
		handler: NewGameMessageHandler(moves, queue, invites, chat, hub),
		moves:   moves,
		queue:   queue,
		invites: invites,
		chat:    chat,
		client:  client,
	}
}

// errorCode returns the code from the single error envelope sent back, or
// "" if the handler replied with nothing.
func (h *handlerHarness) errorCode(t *testing.T) string {
	t.Helper()
	code, _ := h.errorReply(t)
	return code
}

// errorReply is errorCode plus the envelope's inReplyTo.
func (h *handlerHarness) errorReply(t *testing.T) (code, inReplyTo string) {
	t.Helper()
	msgs := h.client.messages()
	if len(msgs) == 0 {
		return "", ""
	}
	if len(msgs) > 1 {
		t.Fatalf("handler sent %d messages, want at most 1", len(msgs))
	}
	var env struct {
		Type    string `json:"type"`
		Payload struct {
			Code      string `json:"code"`
			Message   string `json:"message"`
			InReplyTo string `json:"inReplyTo"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(msgs[0], &env); err != nil {
		t.Fatalf("unmarshalling reply %s: %v", msgs[0], err)
	}
	if env.Type != "error" {
		t.Fatalf("reply type = %q, want error", env.Type)
	}
	return env.Payload.Code, env.Payload.InReplyTo
}

func TestHandleMalformedEnvelope(t *testing.T) {
	h := newHandlerHarness()

	h.handler.Handle("player-1", []byte("this is not json"))

	if got := h.errorCode(t); got != "invalid_message" {
		t.Errorf("code = %q, want invalid_message", got)
	}
}

func TestHandleMalformedPayload(t *testing.T) {
	h := newHandlerHarness()

	// Valid envelope, but payload is a string where an object belongs.
	h.handler.Handle("player-1", []byte(`{"type":"game.submit_move","payload":"nonsense"}`))

	if got := h.errorCode(t); got != "invalid_message" {
		t.Errorf("code = %q, want invalid_message", got)
	}
	if len(h.moves.submitted) != 0 {
		t.Error("an unparseable payload reached the service")
	}
}

// A client sending a type this server doesn't handle yet is being
// forward-compatible, not misbehaving.
func TestHandleUnknownTypeIsIgnored(t *testing.T) {
	h := newHandlerHarness()

	for _, typ := range []string{"game.subscribe_game", "game.offer_draw", "chat.send", ""} {
		h.handler.Handle("player-1", []byte(`{"type":"`+typ+`","payload":{}}`))
	}

	if msgs := h.client.messages(); len(msgs) != 0 {
		t.Errorf("handler replied %q to unhandled types, want silence", msgs)
	}
}

// Player identity comes from the authenticated connection. A payload
// claiming to be someone else must not be believed.
func TestHandleSubmitMoveTakesPlayerFromTheConnection(t *testing.T) {
	h := newHandlerHarness()

	h.handler.Handle("player-1",
		[]byte(`{"type":"game.submit_move","payload":{"gameId":"g1","from":"e2","to":"e4","playerId":"someone-else"}}`))

	if len(h.moves.submitted) != 1 {
		t.Fatalf("submitted %d moves, want 1", len(h.moves.submitted))
	}
	in := h.moves.submitted[0]
	if in.PlayerID != "player-1" {
		t.Errorf("PlayerID = %q, want player-1 from the connection, not the payload", in.PlayerID)
	}
	if in.ID != "g1" || in.From != "e2" || in.To != "e4" {
		t.Errorf("input = %+v", in)
	}
}

func TestHandleSubmitMoveSuccessSendsNoReply(t *testing.T) {
	h := newHandlerHarness()

	h.handler.Handle("player-1", []byte(`{"type":"game.submit_move","payload":{"gameId":"g1","from":"e2","to":"e4"}}`))

	// The broadcast happens inside SubmitMove via the publisher, so the
	// handler itself has nothing to say.
	if msgs := h.client.messages(); len(msgs) != 0 {
		t.Errorf("handler replied %q on success, want nothing", msgs)
	}
}

// This table is the wire contract in websocket-protocol.md. Adding a
// domain error without a code here means clients get internal_error.
func TestHandleSubmitMoveMapsDomainErrorsToWireCodes(t *testing.T) {
	cases := []struct {
		err      error
		wantCode string
	}{
		{domain.ErrIllegalMove, "illegal_move"},
		{domain.ErrNotYourTurn, "not_your_turn"},
		{domain.ErrGameNotFound, "game_not_found"},
		{domain.ErrGameAlreadyFinished, "game_already_finished"},
		{errors.New("something unexpected"), "internal_error"},
	}

	for _, c := range cases {
		t.Run(c.wantCode, func(t *testing.T) {
			h := newHandlerHarness()
			h.moves.submitErr = c.err

			h.handler.Handle("player-1", []byte(`{"type":"game.submit_move","payload":{"gameId":"g1","from":"e2","to":"e4"}}`))

			if got := h.errorCode(t); got != c.wantCode {
				t.Errorf("%v mapped to %q, want %q", c.err, got, c.wantCode)
			}
		})
	}
}

// Wrapped errors must still map, since the services wrap with %w.
func TestHandleSubmitMoveMapsWrappedErrors(t *testing.T) {
	h := newHandlerHarness()
	h.moves.submitErr = errors.Join(errors.New("recording move"), domain.ErrIllegalMove)

	h.handler.Handle("player-1", []byte(`{"type":"game.submit_move","payload":{"gameId":"g1","from":"e2","to":"e4"}}`))

	if got := h.errorCode(t); got != "illegal_move" {
		t.Errorf("code = %q, want illegal_move through the wrapper", got)
	}
}

func TestHandleResign(t *testing.T) {
	h := newHandlerHarness()

	h.handler.Handle("player-1", []byte(`{"type":"game.resign","payload":{"gameId":"g1"}}`))

	if len(h.moves.resigned) != 1 || h.moves.resigned[0] != "g1:player-1" {
		t.Errorf("resigned = %v, want [g1:player-1]", h.moves.resigned)
	}
}

func TestHandleResignMapsErrors(t *testing.T) {
	h := newHandlerHarness()
	h.moves.resignErr = domain.ErrGameAlreadyFinished

	h.handler.Handle("player-1", []byte(`{"type":"game.resign","payload":{"gameId":"g1"}}`))

	if got := h.errorCode(t); got != "game_already_finished" {
		t.Errorf("code = %q, want game_already_finished", got)
	}
}

func TestHandleMatchmakingJoin(t *testing.T) {
	h := newHandlerHarness()

	h.handler.Handle("player-1", []byte(`{"type":"matchmaking.join","payload":{"preset":"3+2"}}`))

	if len(h.queue.joined) != 1 || h.queue.joined[0] != "player-1:3+2" {
		t.Errorf("joined = %v, want [player-1:3+2]", h.queue.joined)
	}
}

func TestHandleMatchmakingJoinMapsErrors(t *testing.T) {
	cases := []struct {
		err      error
		wantCode string
	}{
		{domain.ErrUnknownPreset, "unknown_preset"},
		{domain.ErrAlreadyInGame, "already_in_game"},
		{errors.New("queue exploded"), "internal_error"},
	}

	for _, c := range cases {
		t.Run(c.wantCode, func(t *testing.T) {
			h := newHandlerHarness()
			h.queue.joinErr = c.err

			h.handler.Handle("player-1", []byte(`{"type":"matchmaking.join","payload":{"preset":"3+2"}}`))

			if got := h.errorCode(t); got != c.wantCode {
				t.Errorf("%v mapped to %q, want %q", c.err, got, c.wantCode)
			}
		})
	}
}

func TestHandleMatchmakingLeave(t *testing.T) {
	h := newHandlerHarness()

	h.handler.Handle("player-1", []byte(`{"type":"matchmaking.leave","payload":{"preset":"3+2"}}`))

	if len(h.queue.left) != 1 || h.queue.left[0] != "player-1:3+2" {
		t.Errorf("left = %v, want [player-1:3+2]", h.queue.left)
	}
}

func TestParsePromotion(t *testing.T) {
	cases := map[string]domain.PieceType{
		"q":     domain.Queen,
		"r":     domain.Rook,
		"b":     domain.Bishop,
		"n":     domain.Knight,
		"":      "",
		"k":     "", // king is not a promotion target
		"queen": "", // the wire uses single letters
		"Q":     "", // and is case-sensitive
	}

	for input, want := range cases {
		if got := parsePromotion(input); got != want {
			t.Errorf("parsePromotion(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestHandleSubmitMovePassesPromotionThrough(t *testing.T) {
	h := newHandlerHarness()

	h.handler.Handle("player-1",
		[]byte(`{"type":"game.submit_move","payload":{"gameId":"g1","from":"a7","to":"a8","promotion":"n"}}`))

	if len(h.moves.submitted) != 1 {
		t.Fatalf("submitted %d moves, want 1", len(h.moves.submitted))
	}
	if got := h.moves.submitted[0].Promotion; got != domain.Knight {
		t.Errorf("Promotion = %q, want knight", got)
	}
}

// --- invites ---

// The sender is always the authenticated connection, never a payload field.
func TestHandleInviteMessagesDispatch(t *testing.T) {
	cases := []struct {
		msg  string
		want string
	}{
		{`{"type":"invite.offer","payload":{"toUserId":"player-2","preset":"3+2"}}`, "offer:player-1:player-2:3+2"},
		{`{"type":"invite.accept","payload":{"gameId":"game-9"}}`, "accept:player-1:game-9"},
		{`{"type":"invite.decline","payload":{"gameId":"game-9"}}`, "decline:player-1:game-9"},
		{`{"type":"invite.cancel","payload":{"gameId":"game-9"}}`, "cancel:player-1:game-9"},
	}

	for _, c := range cases {
		t.Run(c.want, func(t *testing.T) {
			h := newHandlerHarness()

			h.handler.Handle("player-1", []byte(c.msg))

			if len(h.invites.calls) != 1 || h.invites.calls[0] != c.want {
				t.Errorf("calls = %v, want [%s]", h.invites.calls, c.want)
			}
			// Success replies come from the service's notifier, not here.
			if got := h.errorCode(t); got != "" {
				t.Errorf("success sent an error %q", got)
			}
		})
	}
}

func TestHandleInviteMalformedPayloads(t *testing.T) {
	for _, msg := range []string{
		`{"type":"invite.offer","payload":{"preset":"3+2"}}`,
		`{"type":"invite.offer","payload":"nope"}`,
		`{"type":"invite.accept","payload":{}}`,
		`{"type":"invite.decline","payload":[]}`,
		`{"type":"invite.cancel","payload":{"gameId":""}}`,
	} {
		t.Run(msg, func(t *testing.T) {
			h := newHandlerHarness()

			h.handler.Handle("player-1", []byte(msg))

			if got := h.errorCode(t); got != "invalid_message" {
				t.Errorf("code = %q, want invalid_message", got)
			}
			if len(h.invites.calls) != 0 {
				t.Errorf("service was called with a malformed payload: %v", h.invites.calls)
			}
		})
	}
}

func TestHandleInviteMapsErrors(t *testing.T) {
	cases := []struct {
		err      error
		wantCode string
	}{
		{domain.ErrInviteNotFound, "invite_not_found"},
		{domain.ErrInviteAlreadyPending, "invite_already_pending"},
		{domain.ErrCannotInviteSelf, "cannot_invite_self"},
		{domain.ErrUserNotFound, "user_not_found"},
		{domain.ErrUserNotConnected, "user_not_connected"},
		{domain.ErrUnknownPreset, "unknown_preset"},
		{fmt.Errorf("wrapped: %w", domain.ErrAlreadyInGame), "already_in_game"},
		{errors.New("db on fire"), "internal_error"},
	}

	for _, c := range cases {
		t.Run(c.wantCode, func(t *testing.T) {
			h := newHandlerHarness()
			h.invites.err = c.err

			h.handler.Handle("player-1", []byte(`{"type":"invite.offer","payload":{"toUserId":"player-2","preset":"3+2"}}`))

			if got := h.errorCode(t); got != c.wantCode {
				t.Errorf("%v mapped to %q, want %q", c.err, got, c.wantCode)
			}
		})
	}
}

// --- chat ---

func TestHandleChatMessagesDispatch(t *testing.T) {
	cases := []struct{ msg, want string }{
		{`{"type":"chat.send_direct","payload":{"toUserId":"player-2","body":"hi"}}`, "direct:player-1:player-2:hi"},
		{`{"type":"chat.send_game","payload":{"gameId":"game-9","body":"gg"}}`, "game:player-1:game-9:gg"},
	}
	for _, c := range cases {
		t.Run(c.want, func(t *testing.T) {
			h := newHandlerHarness()

			h.handler.Handle("player-1", []byte(c.msg))

			if len(h.chat.calls) != 1 || h.chat.calls[0] != c.want {
				t.Errorf("calls = %v, want [%s]", h.chat.calls, c.want)
			}
			// The message itself comes back through ChatNotifier, not here.
			if got := h.errorCode(t); got != "" {
				t.Errorf("success sent an error %q", got)
			}
		})
	}
}

func TestHandleChatMalformedPayloads(t *testing.T) {
	for _, msg := range []string{
		`{"type":"chat.send_direct","payload":{"body":"hi"}}`,
		`{"type":"chat.send_direct","payload":"nope"}`,
		`{"type":"chat.send_game","payload":{"body":"hi"}}`,
	} {
		t.Run(msg, func(t *testing.T) {
			h := newHandlerHarness()

			h.handler.Handle("player-1", []byte(msg))

			if got := h.errorCode(t); got != "invalid_message" {
				t.Errorf("code = %q, want invalid_message", got)
			}
			if len(h.chat.calls) != 0 {
				t.Errorf("service called with a malformed payload: %v", h.chat.calls)
			}
		})
	}
}

func TestHandleChatMapsErrors(t *testing.T) {
	cases := []struct {
		err      error
		wantCode string
	}{
		{domain.ErrInvalidMessageBody, "invalid_message_body"},
		{domain.ErrCannotMessageSelf, "cannot_message_self"},
		{domain.ErrRateLimited, "rate_limited"},
		{domain.ErrConversationNotFound, "conversation_not_found"},
		{domain.ErrUserNotFound, "user_not_found"},
		{domain.ErrGameAlreadyFinished, "game_already_finished"},
	}
	for _, c := range cases {
		t.Run(c.wantCode, func(t *testing.T) {
			h := newHandlerHarness()
			h.chat.err = c.err

			h.handler.Handle("player-1", []byte(`{"type":"chat.send_direct","payload":{"toUserId":"player-2","body":"hi"}}`))

			if got := h.errorCode(t); got != c.wantCode {
				t.Errorf("%v mapped to %q, want %q", c.err, got, c.wantCode)
			}
		})
	}
}

// Every error names the message type that caused it, so a client can tell
// a failed DM's user_not_found from a failed invite's.
func TestErrorsCarryInReplyTo(t *testing.T) {
	cases := []struct{ msg, wantType string }{
		{`{"type":"chat.send_direct","payload":{"toUserId":"p2","body":"hi"}}`, "chat.send_direct"},
		{`{"type":"invite.offer","payload":{"toUserId":"p2","preset":"3+2"}}`, "invite.offer"},
		{`{"type":"game.resign","payload":{"gameId":"g"}}`, "game.resign"},
		{`{"type":"invite.accept","payload":{}}`, "invite.accept"}, // malformed payload, still attributed
	}
	for _, c := range cases {
		t.Run(c.wantType, func(t *testing.T) {
			h := newHandlerHarness()
			h.chat.err = domain.ErrUserNotFound
			h.invites.err = domain.ErrUserNotFound
			h.moves.resignErr = domain.ErrGameNotFound

			h.handler.Handle("player-1", []byte(c.msg))

			if _, got := h.errorReply(t); got != c.wantType {
				t.Errorf("inReplyTo = %q, want %q", got, c.wantType)
			}
		})
	}
}

func TestUnparseableEnvelopeHasNoInReplyTo(t *testing.T) {
	h := newHandlerHarness()

	h.handler.Handle("player-1", []byte("not json"))

	if code, inReplyTo := h.errorReply(t); code != "invalid_message" || inReplyTo != "" {
		t.Errorf("got code %q inReplyTo %q, want invalid_message and none", code, inReplyTo)
	}
}
