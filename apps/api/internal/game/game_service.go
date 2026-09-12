package game

import (
	"api/internal/domain"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

type CreateGameInput struct {
	WhiteID string
	BlackID string
	TimeControl domain.TimeControl
}

type SubmitMoveInput struct {
	ID string
	PlayerID string
	From domain.Square
	To domain.Square
	Promotion domain.PieceType
}

type GameService struct {
	repo GameRepository
	validator MoveValidator
	events GameEventPublisher
}

func NewGameService(repo GameRepository, validator MoveValidator, events GameEventPublisher) *GameService {
	return &GameService{repo: repo, validator: validator, events: events}
}

func (g *GameService) CreateGame(ctx context.Context, input CreateGameInput) (*domain.Game, error) {
	if input.WhiteID == input.BlackID {
		return nil, errors.New("a player cannot play themselves")
	}
	if input.TimeControl.Initial <= 0 {
		return nil, errors.New("time control initial duration must be positive")
	}
	if input.TimeControl.Increment < 0 {
		return nil, errors.New("time control increment cannot be negative")
	}

	gameInput := domain.Game{
		WhiteID: input.WhiteID, 
		BlackID: input.BlackID, 
		FEN: domain.StartingFEN, 
		Status: domain.GameStatusActive, 
		TimeControl: input.TimeControl,
	}

	game, err := g.repo.Create(ctx, &gameInput)
	if err != nil {
		return nil, fmt.Errorf("creating game: %w", err)
	}

	return game, nil
}

func (g *GameService) SubmitMove(ctx context.Context, input SubmitMoveInput) (*domain.Move, *domain.Game, error) {
	game, err := g.repo.FindByID(ctx, input.ID)
	if err != nil {
		return nil, nil, err
	}

	if game.Status != domain.GameStatusActive {
		return nil, nil, domain.ErrGameAlreadyFinished
	}

	if input.PlayerID != game.WhiteID && input.PlayerID != game.BlackID {
		return nil, nil, domain.ErrNotYourTurn
	}

	turn, err := activeColorFromFEN(game.FEN)
	if err != nil {
		return nil, nil, fmt.Errorf("determining turn: %w", err)
	}

	movingColor := domain.White
	if input.PlayerID == game.BlackID {
		movingColor = domain.Black
	}
	if movingColor != turn {
		return nil, nil, domain.ErrNotYourTurn
	}

	history, err := g.repo.FindMovesByGameID(ctx, input.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("loading move history: %w", err)
	}

	result, err := g.validator.ValidateMove(history, game.FEN, input.From, input.To, input.Promotion)
	if err != nil {
		return nil, nil, domain.ErrIllegalMove
	}

	now := time.Now().UTC()
	elapsed := now.Sub(game.LastMoveAt)

	move := &domain.Move{
		GameID:    game.ID,
		Ply:       len(history) + 1,
		Color:     movingColor,
		From:      input.From,
		To:        input.To,
		Promotion: input.Promotion,
		SAN:       result.SAN,
		FENAfter:  result.FEN,
		PlayedAt:  now,
	}

	game.FEN = result.FEN
	game.LastMoveAt = now
	applyClock(game, movingColor, elapsed)
	applyOutcome(game, movingColor, result.Status, now)

	if err := g.repo.AppendMove(ctx, game.ID, move); err != nil {
		return nil, nil, fmt.Errorf("recording move: %w", err)
	}
	if err := g.repo.Update(ctx, game); err != nil {
		return nil, nil, fmt.Errorf("updating game: %w", err)
	}

	// A broadcast failure shouldn't undo an already-persisted move —
	// logged, not returned as an error to the caller.
	if err := g.events.PublishMoveMade(ctx, game, move); err != nil {
		slog.Error("publishing move event", "game", game.ID, "error", err)
	}
	if game.Status == domain.GameStatusFinished {
		if err := g.events.PublishGameOver(ctx, game); err != nil {
			slog.Error("publishing game-over event", "game", game.ID, "error", err)
		}
	}

	return move, game, nil
}

func applyClock(g *domain.Game, mover domain.Color, elapsed time.Duration) {
	if mover == domain.White {
		g.WhiteTimeRemaining -= elapsed
		g.WhiteTimeRemaining += g.TimeControl.Increment
	} else {
		g.BlackTimeRemaining -= elapsed
		g.BlackTimeRemaining += g.TimeControl.Increment
	}
}

func applyOutcome(g *domain.Game, mover domain.Color, status PositionStatus, now time.Time) {
	switch status {
	case PositionCheckmate:
		g.ResultReason = domain.ReasonCheckmate
		if mover == domain.White {
			g.Result = domain.ResultWhiteWins
		} else {
			g.Result = domain.ResultBlackWins
		}
	case PositionStalemate:
		g.Result = domain.ResultDraw
		g.ResultReason = domain.ReasonStalemate
	case PositionDrawRepetition:
		g.Result = domain.ResultDraw
		g.ResultReason = domain.ReasonThreefoldRepetition
	case PositionDrawFiftyMoveRule:
		g.Result = domain.ResultDraw
		g.ResultReason = domain.ReasonFiftyMoveRule
	case PositionDrawInsufficientMaterial:
		g.Result = domain.ResultDraw
		g.ResultReason = domain.ReasonInsufficientMaterial
	default:
		return // ongoing or check — game isn't over
	}
	g.Status = domain.GameStatusFinished
	g.FinishedAt = &now
}

func (g *GameService) Resign(ctx context.Context, gameID string, resignationID string) (*domain.Game, error) {
	game, err := g.repo.FindByID(ctx, gameID)
	if err != nil {
		return nil, err
	}
	if game.Status != domain.GameStatusActive {
		return nil, domain.ErrGameAlreadyFinished
	}
	if resignationID != game.WhiteID && resignationID != game.BlackID {
		return nil, errors.New("player is not a participant in this game")
	}

	now := time.Now().UTC()
	game.Status = domain.GameStatusFinished
	game.ResultReason = domain.ReasonResignation
	game.FinishedAt = &now
	if resignationID == game.WhiteID {
		game.Result = domain.ResultBlackWins
	} else {
		game.Result = domain.ResultWhiteWins
	}

	if err := g.repo.Update(ctx, game); err != nil {
		return nil, fmt.Errorf("updating game: %w", err)
	}
	if err := g.events.PublishGameOver(ctx, game); err != nil {
		slog.Error("publishing game-over event", "game", game.ID, "error", err)
	}
	return game, nil
}

func (g *GameService) OfferDraw(ctx context.Context, gameID string, playerID string) (*domain.Game, error) {
	return nil, nil
}

func (g *GameService) RespondToDraw(ctx context.Context, gameID string, playerID string, accept bool) (*domain.Game, error) {
	return nil, nil
}

func (g *GameService) GetGame(ctx context.Context, gameID string) (*domain.Game, error) {
	return g.repo.FindByID(ctx, gameID)
}

func (g *GameService) GetMoveHistory(ctx context.Context, gameID string) ([]domain.Move, error) {
	return g.repo.FindMovesByGameID(ctx, gameID)
}

func (g *GameService) HasActiveGame(ctx context.Context, playerID string) (bool, error) {
	_, err := g.repo.FindActiveGameByPlayerID(ctx, playerID)
	if err != nil {
		if errors.Is(err, domain.ErrGameNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (g *GameService) HandleTimeExpired(ctx context.Context, gameID string, expiredColor domain.Color) (*domain.Game, error) {
	game, err := g.repo.FindByID(ctx, gameID)
	if err != nil {
		return nil, err
	}
	if game.Status != domain.GameStatusActive {
		return nil, domain.ErrGameAlreadyFinished
	}

	now := time.Now().UTC()
	game.Status = domain.GameStatusFinished
	game.ResultReason = domain.ReasonTimeout
	game.FinishedAt = &now
	if expiredColor == domain.White {
		game.Result = domain.ResultBlackWins
	} else {
		game.Result = domain.ResultWhiteWins
	}

	if err := g.repo.Update(ctx, game); err != nil {
		return nil, fmt.Errorf("updating game: %w", err)
	}
	if err := g.events.PublishGameOver(ctx, game); err != nil {
		slog.Error("publishing game-over event", "game", game.ID, "error", err)
	}
	return game, nil
}

// func GetDrawGame(game *domain.Game) (*domain.Game) {
// 	ret := game
// 	now := time.Now()
// 	ret.Status = domain.GameStatusFinished
// 	ret.Result = domain.ResultDraw
// 	ret.FinishedAt = &now

// 	return ret
// }