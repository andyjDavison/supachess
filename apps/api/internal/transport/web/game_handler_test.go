package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"api/internal/domain"
)

func getGame(h *GameHandler, id string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/games/"+id, nil)
	req.SetPathValue("id", id)
	rec := httptest.NewRecorder()
	h.FindGameByIdHandler(rec, req)
	return rec
}

func getMoves(h *GameHandler, id string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/games/"+id+"/moves", nil)
	req.SetPathValue("id", id)
	rec := httptest.NewRecorder()
	h.FindMovesByGameIdHandler(rec, req)
	return rec
}

func seededGame() *domain.Game {
	finished := time.Unix(1_700_000_100, 0).UTC()
	return &domain.Game{
		ID:      "11111111-1111-1111-1111-111111111111",
		WhiteID: "white-1", BlackID: "black-1",
		FEN: domain.StartingFEN, Status: domain.GameStatusFinished,
		Result: domain.ResultWhiteWins, ResultReason: domain.ReasonCheckmate,
		WhiteTimeRemaining: 90 * time.Second, BlackTimeRemaining: 45 * time.Second,
		LastMoveAt: time.Unix(1_700_000_000, 0).UTC(), FinishedAt: &finished,
	}
}

// --- bug 2 ---
//
// A malformed id used to reach the client as 500. The repository now maps
// Postgres 22P02 onto ErrGameNotFound (see isNoSuchGame, unit-tested in
// internal/game); this is the handler half - ErrGameNotFound must become a
// 404, whichever way the repository arrived at it.
func TestFindGameByIdNotFoundIsFourOhFour(t *testing.T) {
	h := NewGameHandler(newGameService(newFakeGameRepo()))

	for _, id := range []string{
		"00000000-0000-0000-0000-000000000000", // well-formed, absent
		"not-a-uuid",                           // malformed: used to be a 500
	} {
		t.Run(id, func(t *testing.T) {
			if rec := getGame(h, id); rec.Code != http.StatusNotFound {
				t.Errorf("status = %d, want 404: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestFindGameByIdRejectsEmptyId(t *testing.T) {
	h := NewGameHandler(newGameService(newFakeGameRepo()))

	if rec := getGame(h, ""); rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

// A genuine database failure is still a 500 - the 22P02 mapping must not
// have swallowed every error into "not found".
func TestFindGameByIdRepositoryFailureIsFiveHundred(t *testing.T) {
	repo := newFakeGameRepo()
	repo.findErr = errBoom
	h := NewGameHandler(newGameService(repo))

	if rec := getGame(h, "11111111-1111-1111-1111-111111111111"); rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
}

func TestFindGameByIdReturnsTheDTO(t *testing.T) {
	repo := newFakeGameRepo()
	g := seededGame()
	repo.games[g.ID] = g
	h := NewGameHandler(newGameService(repo))

	rec := getGame(h, g.ID)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not JSON: %v", err)
	}

	// Camel-cased, milliseconds - what the frontend's gameSchema parses.
	if body["id"] != g.ID {
		t.Errorf("id = %v, want %v", body["id"], g.ID)
	}
	if body["whiteTimeRemainingMs"] != float64(90_000) {
		t.Errorf("whiteTimeRemainingMs = %v, want 90000", body["whiteTimeRemainingMs"])
	}
	if body["result"] != "white_wins" || body["resultReason"] != "checkmate" {
		t.Errorf("result/reason = %v/%v", body["result"], body["resultReason"])
	}
	if _, present := body["timeControl"]; present {
		t.Error("the DTO leaked a timeControl field the wire format does not define")
	}
}

func TestFindMovesReturnsAnArray(t *testing.T) {
	repo := newFakeGameRepo()
	g := seededGame()
	repo.games[g.ID] = g
	repo.moves[g.ID] = []domain.Move{{
		GameID: g.ID, Ply: 1, Color: domain.White,
		From: "e2", To: "e4", SAN: "e4", FENAfter: "after",
		PlayedAt: time.Unix(1_700_000_000, 0).UTC(),
	}}
	h := NewGameHandler(newGameService(repo))

	rec := getMoves(h, g.ID)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	// A bare array, not an object wrapper - movesResponseSchema depends on it.
	var moves []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &moves); err != nil {
		t.Fatalf("response is not a JSON array: %v (%s)", err, rec.Body.String())
	}
	if len(moves) != 1 {
		t.Fatalf("got %d moves, want 1", len(moves))
	}
	if moves[0]["san"] != "e4" || moves[0]["ply"] != float64(1) || moves[0]["color"] != "white" {
		t.Errorf("move = %v", moves[0])
	}
	if _, present := moves[0]["promotion"]; present {
		t.Error("promotion is present on a non-promotion move, want omitted")
	}
}

func TestFindMovesForAGameWithNoneIsAnEmptyArray(t *testing.T) {
	repo := newFakeGameRepo()
	g := seededGame()
	repo.games[g.ID] = g
	h := NewGameHandler(newGameService(repo))

	rec := getMoves(h, g.ID)

	if body := rec.Body.String(); body != "[]\n" {
		t.Errorf("body = %q, want \"[]\" - the client parses this as an array", body)
	}
}
