package web

import (
	"api/internal/domain"
	"api/internal/game"
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

type GameHandler struct {
	gameService *game.GameService
}

func NewGameHandler(gameService *game.GameService) *GameHandler {
	return &GameHandler{gameService: gameService}
}

type FoundGameResponse struct {
	ID    string  `json:"id"`
	WhiteID    string `json:"whiteID"`
	BlackID            string `json:"blackID"`
	FEN                string `json:"fen"`
	Status             domain.GameStatus `json:"status"`
	Result             domain.GameResult `json:"result"`
	ResultReason       domain.ResultReason `json:"resultReason"`
	TimeControl        domain.TimeControl `json:"timeControl"`
	WhiteTimeRemaining time.Duration `json:"whiteTimeRemaining"`
	BlackTimeRemaining time.Duration `json:"blackTimeRemaining"`
	LastMoveAt         time.Time `json:"lastMoveAt"`
	CreatedAt          time.Time `json:"createdAt"`
	FinishedAt         *time.Time `json:"finishedAt"`
}

type MoveHistoryResponse struct {
	Moves []domain.Move `json:"moves"`
}

func (handler *GameHandler) FindGameByIdHandler(writer http.ResponseWriter, request *http.Request) {
	id := request.PathValue("id")
	if id == "" {
		http.Error(writer, "Missing game id", http.StatusBadRequest)
		return
	}

	foundGame, err := handler.gameService.GetGame(request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrGameNotFound) {
			http.Error(writer, "game not found", http.StatusNotFound)
			return
		}
		http.Error(writer, "internal server error", http.StatusInternalServerError)
		return 
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK) // HTTP 200

	resp := FoundGameResponse{
		ID:    foundGame.ID,
		WhiteID: foundGame.WhiteID,
		BlackID: foundGame.BlackID,
		FEN: foundGame.FEN,
		Status: foundGame.Status,
		Result: foundGame.Result,
		ResultReason: foundGame.ResultReason,
		TimeControl: foundGame.TimeControl,
		WhiteTimeRemaining: foundGame.WhiteTimeRemaining,
		BlackTimeRemaining: foundGame.BlackTimeRemaining,
		LastMoveAt: foundGame.LastMoveAt,
		CreatedAt: foundGame.CreatedAt,
		FinishedAt: foundGame.FinishedAt,
	}
	json.NewEncoder(writer).Encode(resp)
}

func (handler *GameHandler) FindMovesByGameIdHandler(writer http.ResponseWriter, request *http.Request) {
	id := request.PathValue("id")
	if id == "" {
		http.Error(writer, "Missing game id", http.StatusBadRequest)
		return
	}

	foundMoves, err := handler.gameService.GetMoveHistory(request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrGameNotFound) {
			http.Error(writer, "game not found", http.StatusNotFound)
			return
		}
		http.Error(writer, "internal server error", http.StatusInternalServerError)
		return 
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK) // HTTP 200

	resp := MoveHistoryResponse{
		Moves:    foundMoves,
	}
	json.NewEncoder(writer).Encode(resp)
}
