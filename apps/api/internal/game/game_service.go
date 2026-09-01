package game

import (
	"api/internal/domain"
	"context"
	"time"
)

type GameService struct {
	repo *GameRepository
}

func NewGameService(repo *GameRepository) *GameService {
	return &GameService{repo: repo}
}

func (g *GameService) CreateGame(ctx context.Context, whiteId string, blackId string, startingFen string, timeControl domain.TimeControl) (*domain.Game, error) {

	gameInput := domain.Game{WhiteID: whiteId, BlackID: blackId, FEN: startingFen, Status: domain.GameStatusActive, TimeControl: timeControl}

	game, err := g.repo.Create(ctx, &gameInput)
	if err != nil {
		return nil, domain.ErrGameCantBeCreated
	}

	return game, nil
}

func (g *GameService) SubmitMove(ctx context.Context, gameID string, playerID string, from string, to string, promotion string)

func (g *GameService) Resign(ctx context.Context, id string, winner domain.Color) (*domain.Game, error) {

	game, err := g.repo.FindByID(ctx, id)
	if err != nil {
		return nil, domain.ErrGameNotFound
	}

	return GetResignationGame(game, winner), nil
}

func (g *GameService) OfferDraw(ctx context.Context, gameID string, playerID string) (*domain.Game, error) {
	game, err := g.repo.FindByID(ctx, gameID)
	if err != nil {
		return nil, domain.ErrGameNotFound
	}

	return GetDrawGame(game), nil
}

func (g *GameService) RespondToDraw(ctx context.Context, gameID string, playerID string, accept bool) (*domain.Game, error) {
	return nil, nil
}

func (g *GameService) GetGame(ctx context.Context, gameID string) (*domain.Game, error) {
	return nil, nil
}

func (g *GameService) GetMoveHistory(ctx context.Context, gameID string) ([]*domain.Move, error) {
	return nil, nil
}

func (g *GameService) HasActiveGame(ctx context.Context, playerID string) (bool, error) {
	return false, nil
}

func (g *GameService) HandleTimeExpired(ctx context.Context, gameID string) (*domain.Game, error) {
	return nil, nil
}

func GetResignationGame(game *domain.Game, winner domain.Color) (*domain.Game) {
	ret := game
	now := time.Now()
	ret.Status = domain.GameStatusFinished
	if winner == domain.White {
		ret.Result = domain.ResultWhiteWins
	} else {
		ret.Result = domain.ResultBlackWins
	}
	ret.FinishedAt = &now

	return ret
}

func GetDrawGame(game *domain.Game) (*domain.Game) {
	ret := game
	now := time.Now()
	ret.Status = domain.GameStatusFinished
	ret.Result = domain.ResultDraw
	ret.FinishedAt = &now

	return ret
}