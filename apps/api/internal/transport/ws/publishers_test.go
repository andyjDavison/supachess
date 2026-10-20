package ws

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"api/internal/domain"
)

// These assert the exact envelope shapes in websocket-protocol.md. Nothing
// compiler-enforced keeps the Go side and the frontend's zod schemas in
// step, so the shape is pinned here deliberately.

func sampleGame() *domain.Game {
	return &domain.Game{
		ID: "game-1", WhiteID: "white-1", BlackID: "black-1",
		FEN: domain.StartingFEN, Status: domain.GameStatusActive,
		WhiteTimeRemaining: time.Minute, BlackTimeRemaining: time.Minute,
		LastMoveAt: time.Unix(0, 0).UTC(),
	}
}

func decodeEnvelope(t *testing.T, raw []byte) (string, map[string]any) {
	t.Helper()
	var env struct {
		Type    string         `json:"type"`
		Payload map[string]any `json:"payload"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("unmarshalling envelope %s: %v", raw, err)
	}
	return env.Type, env.Payload
}

func TestPublishMoveMadeEnvelope(t *testing.T) {
	h := NewHub()
	white := &fakeClient{id: "white-1"}
	black := &fakeClient{id: "black-1"}
	h.Register(white)
	h.Register(black)

	p := NewGameEventPublisher(h)
	move := &domain.Move{GameID: "game-1", Ply: 1, Color: domain.White, From: "e2", To: "e4", SAN: "e4"}

	if err := p.PublishMoveMade(context.Background(), sampleGame(), move); err != nil {
		t.Fatalf("PublishMoveMade: %v", err)
	}

	// Both participants hear about it.
	for name, c := range map[string]*fakeClient{"white": white, "black": black} {
		msgs := c.messages()
		if len(msgs) != 1 {
			t.Fatalf("%s received %d messages, want 1", name, len(msgs))
		}
		typ, payload := decodeEnvelope(t, msgs[0])
		if typ != "game.move_made" {
			t.Errorf("%s got type %q, want game.move_made", name, typ)
		}
		if _, ok := payload["game"]; !ok {
			t.Errorf("%s payload has no \"game\" key: %v", name, payload)
		}
		if _, ok := payload["move"]; !ok {
			t.Errorf("%s payload has no \"move\" key: %v", name, payload)
		}
	}
}

// Full state, not a delta - a client that missed a frame resyncs from the
// next message it receives.
func TestPublishMoveMadeCarriesFullGameState(t *testing.T) {
	h := NewHub()
	white := &fakeClient{id: "white-1"}
	h.Register(white)

	p := NewGameEventPublisher(h)
	p.PublishMoveMade(context.Background(), sampleGame(),
		&domain.Move{GameID: "game-1", Ply: 1, Color: domain.White, From: "e2", To: "e4", SAN: "e4"})

	_, payload := decodeEnvelope(t, white.messages()[0])
	game, ok := payload["game"].(map[string]any)
	if !ok {
		t.Fatalf("game is not an object: %v", payload["game"])
	}

	for _, key := range []string{"id", "fen", "status", "whiteTimeRemainingMs", "blackTimeRemainingMs", "lastMoveAt"} {
		if _, present := game[key]; !present {
			t.Errorf("game payload is missing %q - a resyncing client needs it", key)
		}
	}
}

func TestPublishGameOverEnvelope(t *testing.T) {
	h := NewHub()
	white := &fakeClient{id: "white-1"}
	h.Register(white)

	g := sampleGame()
	g.Status = domain.GameStatusFinished
	g.Result = domain.ResultWhiteWins
	g.ResultReason = domain.ReasonCheckmate

	if err := NewGameEventPublisher(h).PublishGameOver(context.Background(), g); err != nil {
		t.Fatalf("PublishGameOver: %v", err)
	}

	typ, payload := decodeEnvelope(t, white.messages()[0])
	if typ != "game.game_over" {
		t.Errorf("type = %q, want game.game_over", typ)
	}
	game := payload["game"].(map[string]any)
	if game["result"] != "white_wins" || game["resultReason"] != "checkmate" {
		t.Errorf("game payload = %v, want the result and reason included", game)
	}
}

// A player with no socket open is expected, not an error - they resync on
// reconnect.
func TestPublishSucceedsWhenAPlayerIsOffline(t *testing.T) {
	h := NewHub()
	white := &fakeClient{id: "white-1"}
	h.Register(white) // black never connects

	err := NewGameEventPublisher(h).PublishMoveMade(context.Background(), sampleGame(),
		&domain.Move{GameID: "game-1", Ply: 1, Color: domain.White, From: "e2", To: "e4", SAN: "e4"})

	if err != nil {
		t.Errorf("got %v, want nil - an absent opponent must not fail the publish", err)
	}
	if len(white.messages()) != 1 {
		t.Error("the connected player did not receive the message")
	}
}

func TestNotifyMatchedEnvelope(t *testing.T) {
	h := NewHub()
	white := &fakeClient{id: "white-1"}
	h.Register(white)

	opponent := &domain.User{UserID: "black-1", Username: "bob", Rating: 1337}

	if err := NewMatchmakingNotifier(h).NotifyMatched(context.Background(), "white-1", sampleGame(), opponent); err != nil {
		t.Fatalf("NotifyMatched: %v", err)
	}

	typ, payload := decodeEnvelope(t, white.messages()[0])
	if typ != "matchmaking.matched" {
		t.Errorf("type = %q, want matchmaking.matched", typ)
	}
	if payload["gameId"] != "game-1" {
		t.Errorf("gameId = %v, want game-1", payload["gameId"])
	}

	opp, ok := payload["opponent"].(map[string]any)
	if !ok {
		t.Fatalf("opponent is not an object: %v", payload["opponent"])
	}
	if opp["username"] != "bob" {
		t.Errorf("opponent.username = %v, want bob", opp["username"])
	}
	if opp["rating"] != float64(1337) {
		t.Errorf("opponent.rating = %v, want 1337", opp["rating"])
	}
}

func TestNotifyMatchedToOfflinePlayerReportsNotConnected(t *testing.T) {
	err := NewMatchmakingNotifier(NewHub()).NotifyMatched(
		context.Background(), "nobody", sampleGame(), &domain.User{UserID: "x", Username: "y"})

	if err == nil {
		t.Error("expected ErrUserNotConnected for an offline player")
	}
}
