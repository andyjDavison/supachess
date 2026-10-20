package ws

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"api/internal/domain"
	"api/internal/game"
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

type handlerHarness struct {
	handler *GameMessageHandler
	moves   *fakeMoves
	queue   *fakeQueue
	client  *fakeClient
}

func newHandlerHarness() *handlerHarness {
	hub := NewHub()
	client := &fakeClient{id: "player-1"}
	hub.Register(client)
	moves := &fakeMoves{}
	queue := &fakeQueue{}
	return &handlerHarness{
		handler: NewGameMessageHandler(moves, queue, hub),
		moves:   moves,
		queue:   queue,
		client:  client,
	}
}

// errorCode returns the code from the single error envelope sent back, or
// "" if the handler replied with nothing.
func (h *handlerHarness) errorCode(t *testing.T) string {
	t.Helper()
	msgs := h.client.messages()
	if len(msgs) == 0 {
		return ""
	}
	if len(msgs) > 1 {
		t.Fatalf("handler sent %d messages, want at most 1", len(msgs))
	}
	var env struct {
		Type    string `json:"type"`
		Payload struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(msgs[0], &env); err != nil {
		t.Fatalf("unmarshalling reply %s: %v", msgs[0], err)
	}
	if env.Type != "error" {
		t.Fatalf("reply type = %q, want error", env.Type)
	}
	return env.Payload.Code
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
