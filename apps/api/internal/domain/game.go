package domain

import "time"

type Color string
const (
	White Color = "white"
	Black Color = "black"
)

type GameStatus string

const (
	GameStatusActive   GameStatus = "active"
	GameStatusFinished GameStatus = "finished"
)

type GameResult string

const (
	ResultWhiteWins GameResult = "white_wins"
	ResultBlackWins GameResult = "black_wins"
	ResultDraw      GameResult = "draw"
)

type ResultReason string

const (
	ReasonCheckmate            ResultReason = "checkmate"
	ReasonResignation          ResultReason = "resignation"
	ReasonTimeout              ResultReason = "timeout"
	ReasonStalemate            ResultReason = "stalemate"
	ReasonThreefoldRepetition  ResultReason = "threefold_repetition"
	ReasonFiftyMoveRule        ResultReason = "fifty_move_rule"
	ReasonInsufficientMaterial ResultReason = "insufficient_material"
	ReasonDrawAgreement        ResultReason = "draw_agreement"
)

type TimeControl struct {
	Initial   time.Duration
	Increment time.Duration
}

// StartingFEN is the standard chess starting position.
const StartingFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

type Game struct {
	ID                 string
	WhiteID            string
	BlackID            string
	FEN                string
	Status             GameStatus
	Result             GameResult
	ResultReason       ResultReason
	TimeControl        TimeControl
	WhiteTimeRemaining time.Duration
	BlackTimeRemaining time.Duration
	LastMoveAt         time.Time
	CreatedAt          time.Time
	FinishedAt         *time.Time
}