package dto

import (
	"encoding/json"
	"testing"
	"time"

	"api/internal/domain"
)

// Durations are time.Duration in Go, nanoseconds in Postgres, and
// milliseconds on the wire. This file pins the last conversion, which is
// the one a client would misread silently.
func TestNewGameDTOConvertsDurationsToMilliseconds(t *testing.T) {
	g := &domain.Game{
		ID:                 "game-1",
		WhiteID:            "w",
		BlackID:            "b",
		FEN:                domain.StartingFEN,
		Status:             domain.GameStatusActive,
		WhiteTimeRemaining: 90 * time.Second,
		BlackTimeRemaining: 2500 * time.Millisecond,
	}

	got := NewGameDTO(g)

	if got.WhiteTimeRemainingMs != 90_000 {
		t.Errorf("WhiteTimeRemainingMs = %d, want 90000", got.WhiteTimeRemainingMs)
	}
	if got.BlackTimeRemainingMs != 2_500 {
		t.Errorf("BlackTimeRemainingMs = %d, want 2500", got.BlackTimeRemainingMs)
	}
}

func TestNewGameDTOConvertsTimestampsToUnixMillis(t *testing.T) {
	lastMove := time.Date(2026, 3, 1, 12, 0, 0, 500_000_000, time.UTC)
	g := &domain.Game{ID: "game-1", LastMoveAt: lastMove}

	if got := NewGameDTO(g).LastMoveAt; got != lastMove.UnixMilli() {
		t.Errorf("LastMoveAt = %d, want %d", got, lastMove.UnixMilli())
	}
}

func TestNewGameDTOFinishedAt(t *testing.T) {
	t.Run("nil stays nil and is omitted", func(t *testing.T) {
		got := NewGameDTO(&domain.Game{ID: "g", Status: domain.GameStatusActive})
		if got.FinishedAt != nil {
			t.Errorf("FinishedAt = %v, want nil", *got.FinishedAt)
		}

		raw, err := json.Marshal(got)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var decoded map[string]any
		if err := json.Unmarshal(raw, &decoded); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if _, present := decoded["finishedAt"]; present {
			t.Error("finishedAt is present in the JSON for an unfinished game")
		}
	})

	t.Run("set is converted", func(t *testing.T) {
		finished := time.Date(2026, 3, 1, 13, 0, 0, 0, time.UTC)
		got := NewGameDTO(&domain.Game{ID: "g", FinishedAt: &finished})

		if got.FinishedAt == nil {
			t.Fatal("FinishedAt is nil")
		}
		if *got.FinishedAt != finished.UnixMilli() {
			t.Errorf("FinishedAt = %d, want %d", *got.FinishedAt, finished.UnixMilli())
		}
	})
}

// The frontend's zod schema has result/resultReason optional, which only
// works because an unfinished game omits them rather than sending "".
func TestGameDTOOmitsEmptyResultFields(t *testing.T) {
	raw, err := json.Marshal(NewGameDTO(&domain.Game{ID: "g", Status: domain.GameStatusActive}))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	for _, key := range []string{"result", "resultReason"} {
		if _, present := decoded[key]; present {
			t.Errorf("%q is present for an in-progress game, want omitted", key)
		}
	}
	// These must always be sent, even at zero, or the client's clocks break.
	for _, key := range []string{"whiteTimeRemainingMs", "blackTimeRemainingMs", "lastMoveAt", "fen", "status"} {
		if _, present := decoded[key]; !present {
			t.Errorf("%q is missing from the payload", key)
		}
	}
}

func TestGameDTOJSONKeysMatchTheProtocol(t *testing.T) {
	finished := time.Unix(0, 0).UTC()
	raw, err := json.Marshal(NewGameDTO(&domain.Game{
		ID: "g", WhiteID: "w", BlackID: "b", FEN: "fen",
		Status: domain.GameStatusFinished, Result: domain.ResultDraw,
		ResultReason: domain.ReasonStalemate, FinishedAt: &finished,
	}))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Camel-cased keys, as the frontend's gameSchema expects.
	want := []string{"id", "whiteId", "blackId", "fen", "status", "result",
		"resultReason", "whiteTimeRemainingMs", "blackTimeRemainingMs", "lastMoveAt", "finishedAt"}
	for _, key := range want {
		if _, present := decoded[key]; !present {
			t.Errorf("payload is missing %q", key)
		}
	}
	if len(decoded) != len(want) {
		t.Errorf("payload has %d keys, want %d: %v", len(decoded), len(want), decoded)
	}
}

func TestNewMoveDTO(t *testing.T) {
	playedAt := time.Date(2026, 3, 1, 12, 0, 1, 0, time.UTC)
	m := &domain.Move{
		GameID: "game-1", Ply: 3, Color: domain.White,
		From: "e2", To: "e4", SAN: "e4", FENAfter: "after-fen", PlayedAt: playedAt,
	}

	got := NewMoveDTO(m)

	if got.Ply != 3 || got.Color != "white" || got.From != "e2" || got.To != "e4" || got.SAN != "e4" {
		t.Errorf("dto = %+v", got)
	}
	if got.PlayedAt != playedAt.UnixMilli() {
		t.Errorf("PlayedAt = %d, want %d", got.PlayedAt, playedAt.UnixMilli())
	}
	if got.Promotion != "" {
		t.Errorf("Promotion = %q, want empty for a non-promotion", got.Promotion)
	}
}

func TestMoveDTOOmitsEmptyPromotionButCarriesItWhenSet(t *testing.T) {
	base := domain.Move{GameID: "g", Ply: 1, Color: domain.White, From: "a7", To: "a8", SAN: "a8=Q"}

	raw, _ := json.Marshal(NewMoveDTO(&base))
	var decoded map[string]any
	json.Unmarshal(raw, &decoded)
	if _, present := decoded["promotion"]; present {
		t.Error("promotion is present on a non-promotion move, want omitted")
	}

	promoted := base
	promoted.Promotion = domain.Queen
	if got := NewMoveDTO(&promoted).Promotion; got != "queen" {
		t.Errorf("Promotion = %q, want queen", got)
	}
}
