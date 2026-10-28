package game

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"api/internal/domain"
)

// --- doubles for the three seams GameService depends on ---

type memGameRepo struct {
	mu        sync.Mutex
	games     map[string]*domain.Game
	moves     map[string][]domain.Move
	nextID    int
	createErr error
	updateErr error
	appendErr error
}

func newMemGameRepo() *memGameRepo {
	return &memGameRepo{games: map[string]*domain.Game{}, moves: map[string][]domain.Move{}}
}

func (r *memGameRepo) Create(_ context.Context, g *domain.Game) (*domain.Game, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.createErr != nil {
		return nil, r.createErr
	}
	r.nextID++
	stored := *g
	stored.ID = "game-1"
	stored.CreatedAt = time.Now().UTC()
	stored.LastMoveAt = stored.CreatedAt
	r.games[stored.ID] = &stored
	copied := stored
	return &copied, nil
}

func (r *memGameRepo) FindByID(_ context.Context, id string) (*domain.Game, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	g, ok := r.games[id]
	if !ok {
		return nil, domain.ErrGameNotFound
	}
	// Hand back a copy, like a real query would - the service must persist
	// its mutations through Update rather than rely on a shared pointer.
	copied := *g
	return &copied, nil
}

func (r *memGameRepo) FindActiveGameByPlayerID(_ context.Context, playerID string) (*domain.Game, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, g := range r.games {
		if g.Status == domain.GameStatusActive && (g.WhiteID == playerID || g.BlackID == playerID) {
			copied := *g
			return &copied, nil
		}
	}
	return nil, domain.ErrGameNotFound
}

func (r *memGameRepo) Update(_ context.Context, g *domain.Game) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.updateErr != nil {
		return r.updateErr
	}
	if _, ok := r.games[g.ID]; !ok {
		return domain.ErrGameNotFound
	}
	copied := *g
	r.games[g.ID] = &copied
	return nil
}

func (r *memGameRepo) AppendMove(_ context.Context, gameID string, m *domain.Move) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.appendErr != nil {
		return r.appendErr
	}
	r.moves[gameID] = append(r.moves[gameID], *m)
	return nil
}

func (r *memGameRepo) FindMovesByGameID(_ context.Context, gameID string) ([]domain.Move, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]domain.Move(nil), r.moves[gameID]...), nil
}

func (r *memGameRepo) DeleteWaitingGame(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	g, ok := r.games[id]
	if !ok || g.Status != domain.GameStatusWaiting {
		return domain.ErrGameNotFound
	}
	delete(r.games, id)
	return nil
}

func (r *memGameRepo) DeleteAllWaitingGames(_ context.Context) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var n int64
	for id, g := range r.games {
		if g.Status == domain.GameStatusWaiting {
			delete(r.games, id)
			n++
		}
	}
	return n, nil
}

func (r *memGameRepo) stored(id string) *domain.Game {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.games[id]
}

func (r *memGameRepo) storedMoves(id string) []domain.Move {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]domain.Move(nil), r.moves[id]...)
}

// stubValidator returns a scripted result so service tests exercise the
// service, not the chess engine.
type stubValidator struct {
	result MoveResult
	err    error
	calls  int
}

func (v *stubValidator) ValidateMove(_ []domain.Move, _ string, _, _ domain.Square, _ domain.PieceType) (MoveResult, error) {
	v.calls++
	if v.err != nil {
		return MoveResult{}, v.err
	}
	return v.result, nil
}

type spyPublisher struct {
	moveMade []*domain.Move
	gameOver []*domain.Game
	moveErr  error
	overErr  error
}

func (p *spyPublisher) PublishMoveMade(_ context.Context, _ *domain.Game, m *domain.Move) error {
	p.moveMade = append(p.moveMade, m)
	return p.moveErr
}

func (p *spyPublisher) PublishGameOver(_ context.Context, g *domain.Game) error {
	p.gameOver = append(p.gameOver, g)
	return p.overErr
}

type scheduled struct {
	gameID    string
	remaining time.Duration
	color     domain.Color
}

type spyScheduler struct {
	schedules []scheduled
	cancels   []string
}

func (s *spyScheduler) Schedule(gameID string, remaining time.Duration, expiredColor domain.Color) {
	s.schedules = append(s.schedules, scheduled{gameID, remaining, expiredColor})
}

func (s *spyScheduler) Cancel(gameID string) { s.cancels = append(s.cancels, gameID) }

type harness struct {
	svc       *GameService
	repo      *memGameRepo
	validator *stubValidator
	events    *spyPublisher
	timers    *spyScheduler
}

func newHarness() *harness {
	repo := newMemGameRepo()
	validator := &stubValidator{result: MoveResult{
		FEN:    "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1",
		SAN:    "e4",
		Status: PositionOngoing,
	}}
	events := &spyPublisher{}
	timers := &spyScheduler{}
	return &harness{
		svc:       NewGameService(repo, validator, events, timers),
		repo:      repo,
		validator: validator,
		events:    events,
		timers:    timers,
	}
}

const (
	whiteID = "white-player"
	blackID = "black-player"
)

func (h *harness) seedActiveGame(t *testing.T, tc domain.TimeControl) *domain.Game {
	t.Helper()
	g, err := h.svc.CreateGame(context.Background(), CreateGameInput{
		WhiteID: whiteID, BlackID: blackID, TimeControl: tc,
	})
	if err != nil {
		t.Fatalf("seeding game: %v", err)
	}
	h.timers.schedules = nil // ignore the arming from CreateGame
	return g
}

func blitz() domain.TimeControl {
	return domain.TimeControl{Initial: 3 * time.Minute, Increment: 2 * time.Second}
}

// --- CreateGame ---

func TestCreateGameValidatesInput(t *testing.T) {
	cases := []struct {
		name  string
		input CreateGameInput
	}{
		{"a player cannot play themselves", CreateGameInput{WhiteID: "x", BlackID: "x", TimeControl: blitz()}},
		{"zero initial time", CreateGameInput{WhiteID: whiteID, BlackID: blackID, TimeControl: domain.TimeControl{}}},
		{"negative initial time", CreateGameInput{WhiteID: whiteID, BlackID: blackID,
			TimeControl: domain.TimeControl{Initial: -time.Minute}}},
		{"negative increment", CreateGameInput{WhiteID: whiteID, BlackID: blackID,
			TimeControl: domain.TimeControl{Initial: time.Minute, Increment: -time.Second}}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := newHarness()

			if _, err := h.svc.CreateGame(context.Background(), c.input); err == nil {
				t.Error("expected an error")
			}
			if len(h.timers.schedules) != 0 {
				t.Error("a timer was armed for a rejected game")
			}
		})
	}
}

func TestCreateGameSeedsBothClocksAndArmsWhitesTimer(t *testing.T) {
	h := newHarness()

	g, err := h.svc.CreateGame(context.Background(), CreateGameInput{
		WhiteID: whiteID, BlackID: blackID, TimeControl: blitz(),
	})
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}

	if g.FEN != domain.StartingFEN {
		t.Errorf("FEN = %q, want the starting position", g.FEN)
	}
	if g.Status != domain.GameStatusActive {
		t.Errorf("Status = %q, want active", g.Status)
	}
	if g.WhiteTimeRemaining != 3*time.Minute || g.BlackTimeRemaining != 3*time.Minute {
		t.Errorf("clocks = (%v, %v), want both at the 3m initial", g.WhiteTimeRemaining, g.BlackTimeRemaining)
	}

	if len(h.timers.schedules) != 1 {
		t.Fatalf("armed %d timers, want 1", len(h.timers.schedules))
	}
	armed := h.timers.schedules[0]
	if armed.color != domain.White || armed.remaining != 3*time.Minute {
		t.Errorf("armed %+v, want white for 3m - white moves first", armed)
	}
}

// --- SubmitMove rejections ---

func TestSubmitMoveRejectsFinishedGame(t *testing.T) {
	h := newHarness()
	g := h.seedActiveGame(t, blitz())
	stored := h.repo.stored(g.ID)
	stored.Status = domain.GameStatusFinished

	_, _, err := h.svc.SubmitMove(context.Background(), SubmitMoveInput{
		ID: g.ID, PlayerID: whiteID, From: "e2", To: "e4",
	})

	if !errors.Is(err, domain.ErrGameAlreadyFinished) {
		t.Errorf("got %v, want ErrGameAlreadyFinished", err)
	}
}

func TestSubmitMoveRejectsUnknownGame(t *testing.T) {
	h := newHarness()

	_, _, err := h.svc.SubmitMove(context.Background(), SubmitMoveInput{
		ID: "no-such-game", PlayerID: whiteID, From: "e2", To: "e4",
	})

	if !errors.Is(err, domain.ErrGameNotFound) {
		t.Errorf("got %v, want ErrGameNotFound", err)
	}
}

func TestSubmitMoveRejectsNonParticipant(t *testing.T) {
	h := newHarness()
	g := h.seedActiveGame(t, blitz())

	_, _, err := h.svc.SubmitMove(context.Background(), SubmitMoveInput{
		ID: g.ID, PlayerID: "a-stranger", From: "e2", To: "e4",
	})

	if !errors.Is(err, domain.ErrNotYourTurn) {
		t.Errorf("got %v, want ErrNotYourTurn", err)
	}
	if h.validator.calls != 0 {
		t.Error("the move was validated before the player was authorised")
	}
}

// Turn order comes from the FEN, not from a counter the client could skew.
func TestSubmitMoveRejectsMoveOutOfTurn(t *testing.T) {
	h := newHarness()
	g := h.seedActiveGame(t, blitz())

	// Starting position: white to move, so black moving is out of turn.
	_, _, err := h.svc.SubmitMove(context.Background(), SubmitMoveInput{
		ID: g.ID, PlayerID: blackID, From: "e7", To: "e5",
	})

	if !errors.Is(err, domain.ErrNotYourTurn) {
		t.Errorf("got %v, want ErrNotYourTurn", err)
	}
	if h.validator.calls != 0 {
		t.Error("an out-of-turn move reached the validator")
	}
}

func TestSubmitMoveMapsValidatorRejectionToIllegalMove(t *testing.T) {
	h := newHarness()
	g := h.seedActiveGame(t, blitz())
	h.validator.err = errors.New("some engine-specific complaint")

	_, _, err := h.svc.SubmitMove(context.Background(), SubmitMoveInput{
		ID: g.ID, PlayerID: whiteID, From: "e2", To: "e5",
	})

	if !errors.Is(err, domain.ErrIllegalMove) {
		t.Errorf("got %v, want ErrIllegalMove", err)
	}
	if len(h.repo.storedMoves(g.ID)) != 0 {
		t.Error("an illegal move was persisted")
	}
}

// --- SubmitMove happy path ---

func TestSubmitMovePersistsPublishesAndPassesTheClock(t *testing.T) {
	h := newHarness()
	g := h.seedActiveGame(t, blitz())

	move, updated, err := h.svc.SubmitMove(context.Background(), SubmitMoveInput{
		ID: g.ID, PlayerID: whiteID, From: "e2", To: "e4",
	})
	if err != nil {
		t.Fatalf("SubmitMove: %v", err)
	}

	if move.Ply != 1 {
		t.Errorf("Ply = %d, want 1", move.Ply)
	}
	if move.Color != domain.White {
		t.Errorf("Color = %q, want white", move.Color)
	}
	if move.SAN != "e4" {
		t.Errorf("SAN = %q, want the validator's e4", move.SAN)
	}
	if move.FENAfter != h.validator.result.FEN {
		t.Errorf("FENAfter = %q, want the validator's FEN", move.FENAfter)
	}

	if persisted := h.repo.storedMoves(g.ID); len(persisted) != 1 || persisted[0].Ply != 1 {
		t.Errorf("stored moves = %+v, want exactly ply 1", persisted)
	}
	if h.repo.stored(g.ID).FEN != h.validator.result.FEN {
		t.Error("the game's FEN was not advanced")
	}
	if updated.Status != domain.GameStatusActive {
		t.Errorf("Status = %q, want still active", updated.Status)
	}

	if len(h.events.moveMade) != 1 {
		t.Errorf("published %d move_made events, want 1", len(h.events.moveMade))
	}
	if len(h.events.gameOver) != 0 {
		t.Error("published game_over for an ongoing game")
	}

	// The clock now runs for black, armed with black's remaining time.
	if len(h.timers.schedules) != 1 {
		t.Fatalf("armed %d timers, want 1", len(h.timers.schedules))
	}
	if armed := h.timers.schedules[0]; armed.color != domain.Black {
		t.Errorf("armed for %q, want black - the turn passed", armed.color)
	}
	if len(h.timers.cancels) != 0 {
		t.Error("cancelled the timer on an ongoing game")
	}
}

func TestSubmitMoveSecondPlyIsBlack(t *testing.T) {
	h := newHarness()
	g := h.seedActiveGame(t, blitz())
	ctx := context.Background()

	if _, _, err := h.svc.SubmitMove(ctx, SubmitMoveInput{ID: g.ID, PlayerID: whiteID, From: "e2", To: "e4"}); err != nil {
		t.Fatalf("white's move: %v", err)
	}
	// The stub advanced the FEN to "black to move"; now script black's reply.
	h.validator.result = MoveResult{
		FEN:    "rnbqkbnr/pppp1ppp/8/4p3/4P3/8/PPPP1PPP/RNBQKBNR w KQkq e6 0 2",
		SAN:    "e5",
		Status: PositionOngoing,
	}

	move, _, err := h.svc.SubmitMove(ctx, SubmitMoveInput{ID: g.ID, PlayerID: blackID, From: "e7", To: "e5"})
	if err != nil {
		t.Fatalf("black's move: %v", err)
	}

	if move.Ply != 2 || move.Color != domain.Black {
		t.Errorf("move = ply %d / %q, want ply 2 / black", move.Ply, move.Color)
	}
	if armed := h.timers.schedules[len(h.timers.schedules)-1]; armed.color != domain.White {
		t.Errorf("armed for %q, want white", armed.color)
	}
}

// --- clock arithmetic ---

func TestSubmitMoveDeductsElapsedAndAddsIncrementForTheMoverOnly(t *testing.T) {
	h := newHarness()
	g := h.seedActiveGame(t, blitz())

	// Backdate LastMoveAt so a known amount of thinking time has passed.
	stored := h.repo.stored(g.ID)
	stored.LastMoveAt = time.Now().UTC().Add(-10 * time.Second)

	_, updated, err := h.svc.SubmitMove(context.Background(), SubmitMoveInput{
		ID: g.ID, PlayerID: whiteID, From: "e2", To: "e4",
	})
	if err != nil {
		t.Fatalf("SubmitMove: %v", err)
	}

	// 3m - 10s thought + 2s increment, with slack for real elapsed time.
	want := 3*time.Minute - 10*time.Second + 2*time.Second
	if diff := updated.WhiteTimeRemaining - want; diff > time.Second || diff < -time.Second {
		t.Errorf("white clock = %v, want about %v", updated.WhiteTimeRemaining, want)
	}
	if updated.BlackTimeRemaining != 3*time.Minute {
		t.Errorf("black clock = %v, want untouched 3m", updated.BlackTimeRemaining)
	}
}

func TestApplyClock(t *testing.T) {
	tc := domain.TimeControl{Initial: time.Minute, Increment: 3 * time.Second}

	t.Run("white", func(t *testing.T) {
		g := &domain.Game{TimeControl: tc, WhiteTimeRemaining: time.Minute, BlackTimeRemaining: time.Minute}
		applyClock(g, domain.White, 5*time.Second)
		if g.WhiteTimeRemaining != 58*time.Second {
			t.Errorf("white = %v, want 58s (60 - 5 + 3)", g.WhiteTimeRemaining)
		}
		if g.BlackTimeRemaining != time.Minute {
			t.Errorf("black = %v, want untouched", g.BlackTimeRemaining)
		}
	})

	t.Run("black", func(t *testing.T) {
		g := &domain.Game{TimeControl: tc, WhiteTimeRemaining: time.Minute, BlackTimeRemaining: time.Minute}
		applyClock(g, domain.Black, 5*time.Second)
		if g.BlackTimeRemaining != 58*time.Second {
			t.Errorf("black = %v, want 58s", g.BlackTimeRemaining)
		}
		if g.WhiteTimeRemaining != time.Minute {
			t.Errorf("white = %v, want untouched", g.WhiteTimeRemaining)
		}
	})
}

// --- outcomes ---

func TestApplyOutcome(t *testing.T) {
	cases := []struct {
		name       string
		status     PositionStatus
		mover      domain.Color
		wantResult domain.GameResult
		wantReason domain.ResultReason
		wantOver   bool
	}{
		{"white mates", PositionCheckmate, domain.White, domain.ResultWhiteWins, domain.ReasonCheckmate, true},
		{"black mates", PositionCheckmate, domain.Black, domain.ResultBlackWins, domain.ReasonCheckmate, true},
		{"stalemate", PositionStalemate, domain.White, domain.ResultDraw, domain.ReasonStalemate, true},
		{"repetition", PositionDrawRepetition, domain.White, domain.ResultDraw, domain.ReasonThreefoldRepetition, true},
		{"fifty-move rule", PositionDrawFiftyMoveRule, domain.White, domain.ResultDraw, domain.ReasonFiftyMoveRule, true},
		{"insufficient material", PositionDrawInsufficientMaterial, domain.White, domain.ResultDraw, domain.ReasonInsufficientMaterial, true},
		{"ongoing", PositionOngoing, domain.White, "", "", false},
		{"check is not an ending", PositionCheck, domain.White, "", "", false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			g := &domain.Game{Status: domain.GameStatusActive}
			now := time.Now().UTC()

			applyOutcome(g, c.mover, c.status, now)

			if g.Result != c.wantResult {
				t.Errorf("Result = %q, want %q", g.Result, c.wantResult)
			}
			if g.ResultReason != c.wantReason {
				t.Errorf("ResultReason = %q, want %q", g.ResultReason, c.wantReason)
			}
			if c.wantOver {
				if g.Status != domain.GameStatusFinished {
					t.Errorf("Status = %q, want finished", g.Status)
				}
				if g.FinishedAt == nil {
					t.Error("FinishedAt is nil on a finished game")
				}
			} else {
				if g.Status != domain.GameStatusActive {
					t.Errorf("Status = %q, want still active", g.Status)
				}
				if g.FinishedAt != nil {
					t.Error("FinishedAt was set on an ongoing game")
				}
			}
		})
	}
}

func TestSubmitMoveCheckmateFinishesCancelsAndAnnounces(t *testing.T) {
	h := newHarness()
	g := h.seedActiveGame(t, blitz())
	h.validator.result = MoveResult{FEN: "8/8/8/8/8/8/8/8 b - - 0 1", SAN: "Qh5#", Status: PositionCheckmate}

	_, updated, err := h.svc.SubmitMove(context.Background(), SubmitMoveInput{
		ID: g.ID, PlayerID: whiteID, From: "d1", To: "h5",
	})
	if err != nil {
		t.Fatalf("SubmitMove: %v", err)
	}

	if updated.Status != domain.GameStatusFinished {
		t.Errorf("Status = %q, want finished", updated.Status)
	}
	if updated.Result != domain.ResultWhiteWins || updated.ResultReason != domain.ReasonCheckmate {
		t.Errorf("outcome = %q/%q, want white_wins/checkmate", updated.Result, updated.ResultReason)
	}
	if len(h.events.gameOver) != 1 {
		t.Errorf("published %d game_over events, want 1", len(h.events.gameOver))
	}
	if len(h.timers.cancels) != 1 || h.timers.cancels[0] != g.ID {
		t.Errorf("cancels = %v, want one for %s", h.timers.cancels, g.ID)
	}
	if len(h.timers.schedules) != 0 {
		t.Error("armed a new timer on a finished game")
	}
}

// A broadcast failure must not roll back a move that is already persisted.
func TestSubmitMoveSurvivesPublisherFailure(t *testing.T) {
	h := newHarness()
	g := h.seedActiveGame(t, blitz())
	h.events.moveErr = errors.New("every socket is gone")

	move, _, err := h.svc.SubmitMove(context.Background(), SubmitMoveInput{
		ID: g.ID, PlayerID: whiteID, From: "e2", To: "e4",
	})

	if err != nil {
		t.Fatalf("a publish failure surfaced as an error: %v", err)
	}
	if move == nil {
		t.Fatal("no move returned")
	}
	if len(h.repo.storedMoves(g.ID)) != 1 {
		t.Error("the move was not persisted")
	}
}

func TestSubmitMovePropagatesPersistenceFailure(t *testing.T) {
	h := newHarness()
	g := h.seedActiveGame(t, blitz())
	h.repo.appendErr = errors.New("disk is full")

	if _, _, err := h.svc.SubmitMove(context.Background(), SubmitMoveInput{
		ID: g.ID, PlayerID: whiteID, From: "e2", To: "e4",
	}); err == nil {
		t.Error("a failed AppendMove was swallowed")
	}
}

// --- Resign ---

func TestResign(t *testing.T) {
	cases := []struct {
		name       string
		resigner   string
		wantResult domain.GameResult
	}{
		{"white resigns, black wins", whiteID, domain.ResultBlackWins},
		{"black resigns, white wins", blackID, domain.ResultWhiteWins},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := newHarness()
			g := h.seedActiveGame(t, blitz())

			updated, err := h.svc.Resign(context.Background(), g.ID, c.resigner)
			if err != nil {
				t.Fatalf("Resign: %v", err)
			}

			if updated.Result != c.wantResult {
				t.Errorf("Result = %q, want %q", updated.Result, c.wantResult)
			}
			if updated.ResultReason != domain.ReasonResignation {
				t.Errorf("ResultReason = %q, want resignation", updated.ResultReason)
			}
			if updated.Status != domain.GameStatusFinished || updated.FinishedAt == nil {
				t.Error("game not marked finished")
			}
			if len(h.timers.cancels) != 1 {
				t.Errorf("cancels = %v, want one", h.timers.cancels)
			}
			if len(h.events.gameOver) != 1 {
				t.Errorf("published %d game_over events, want 1", len(h.events.gameOver))
			}
			if h.repo.stored(g.ID).Status != domain.GameStatusFinished {
				t.Error("the finished state was not persisted")
			}
		})
	}
}

func TestResignRejectsNonParticipantAndFinishedGame(t *testing.T) {
	t.Run("non-participant", func(t *testing.T) {
		h := newHarness()
		g := h.seedActiveGame(t, blitz())

		if _, err := h.svc.Resign(context.Background(), g.ID, "a-stranger"); err == nil {
			t.Error("a stranger was allowed to resign someone else's game")
		}
		if h.repo.stored(g.ID).Status != domain.GameStatusActive {
			t.Error("the game was ended by a non-participant")
		}
	})

	t.Run("already finished", func(t *testing.T) {
		h := newHarness()
		g := h.seedActiveGame(t, blitz())
		h.repo.stored(g.ID).Status = domain.GameStatusFinished

		if _, err := h.svc.Resign(context.Background(), g.ID, whiteID); !errors.Is(err, domain.ErrGameAlreadyFinished) {
			t.Errorf("got %v, want ErrGameAlreadyFinished", err)
		}
	})
}

// --- HandleTimeExpired ---

func TestHandleTimeExpired(t *testing.T) {
	cases := []struct {
		name       string
		expired    domain.Color
		wantResult domain.GameResult
	}{
		{"white flags, black wins", domain.White, domain.ResultBlackWins},
		{"black flags, white wins", domain.Black, domain.ResultWhiteWins},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := newHarness()
			g := h.seedActiveGame(t, blitz())

			updated, err := h.svc.HandleTimeExpired(context.Background(), g.ID, c.expired)
			if err != nil {
				t.Fatalf("HandleTimeExpired: %v", err)
			}

			if updated.Result != c.wantResult {
				t.Errorf("Result = %q, want %q", updated.Result, c.wantResult)
			}
			if updated.ResultReason != domain.ReasonTimeout {
				t.Errorf("ResultReason = %q, want timeout", updated.ResultReason)
			}
			if updated.Status != domain.GameStatusFinished || updated.FinishedAt == nil {
				t.Error("game not marked finished")
			}
			if len(h.events.gameOver) != 1 {
				t.Errorf("published %d game_over events, want 1", len(h.events.gameOver))
			}
		})
	}
}

func TestHandleTimeExpiredOnFinishedGame(t *testing.T) {
	h := newHarness()
	g := h.seedActiveGame(t, blitz())
	h.repo.stored(g.ID).Status = domain.GameStatusFinished

	_, err := h.svc.HandleTimeExpired(context.Background(), g.ID, domain.White)

	if !errors.Is(err, domain.ErrGameAlreadyFinished) {
		t.Errorf("got %v, want ErrGameAlreadyFinished", err)
	}
	if len(h.events.gameOver) != 0 {
		t.Error("announced game_over twice for one game")
	}
}

// --- HasActiveGame ---

func TestHasActiveGame(t *testing.T) {
	h := newHarness()
	ctx := context.Background()

	// ErrGameNotFound is the normal "no game" answer, not a failure.
	has, err := h.svc.HasActiveGame(ctx, whiteID)
	if err != nil {
		t.Fatalf("with no games: %v", err)
	}
	if has {
		t.Error("reported an active game before any existed")
	}

	g := h.seedActiveGame(t, blitz())
	for _, id := range []string{whiteID, blackID} {
		has, err = h.svc.HasActiveGame(ctx, id)
		if err != nil || !has {
			t.Errorf("HasActiveGame(%s) = (%v, %v), want (true, nil)", id, has, err)
		}
	}

	h.repo.stored(g.ID).Status = domain.GameStatusFinished
	has, err = h.svc.HasActiveGame(ctx, whiteID)
	if err != nil || has {
		t.Errorf("after the game finished = (%v, %v), want (false, nil)", has, err)
	}
}

// --- read-throughs ---

func TestGetGameAndMoveHistory(t *testing.T) {
	h := newHarness()
	g := h.seedActiveGame(t, blitz())
	ctx := context.Background()
	if _, _, err := h.svc.SubmitMove(ctx, SubmitMoveInput{ID: g.ID, PlayerID: whiteID, From: "e2", To: "e4"}); err != nil {
		t.Fatalf("seeding a move: %v", err)
	}

	got, err := h.svc.GetGame(ctx, g.ID)
	if err != nil || got.ID != g.ID {
		t.Errorf("GetGame = (%v, %v)", got, err)
	}

	history, err := h.svc.GetMoveHistory(ctx, g.ID)
	if err != nil {
		t.Fatalf("GetMoveHistory: %v", err)
	}
	if len(history) != 1 || history[0].SAN != "e4" {
		t.Errorf("history = %+v, want one e4", history)
	}

	if _, err := h.svc.GetGame(ctx, "nope"); !errors.Is(err, domain.ErrGameNotFound) {
		t.Errorf("GetGame on an unknown id = %v, want ErrGameNotFound", err)
	}
}
