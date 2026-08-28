package domain

import "time"

type Game struct {
	GameID string
	WhitePlayerID string
	BlackPlayerID string
	CurrentPosition string
	Status GameStatus
	Result ResultStatus
	WhiteTimeRemaining time.Duration
	BlackTimeRemaining time.Duration
	TimeControl time.Duration
	LastMoveAt time.Time
	CreatedAt time.Time
	FinishedAt *time.Time
}

type GameStatus int
const (
	Waiting GameStatus = iota
	Active
	Finished
)

var gameStatus = map[GameStatus]string{
	Waiting: "waiting",
	Active: "active",
	Finished: "finished",
}

func (gs GameStatus) String() string {
	return gameStatus[gs]
}

type ResultStatus int
const (
	WhiteWonCM ResultStatus = iota
	WhiteWonTime
	BlackWonCM
	BlackWonTime
	Draw
	Aborted
)

var resultStatus = map[ResultStatus]string{
	WhiteWonCM: "wcm",
	WhiteWonTime: "wt",
	BlackWonCM: "bcm",
	BlackWonTime: "bt",
	Draw: "draw",
	Aborted: "abort",
}

func (rs ResultStatus) String() string {
	return resultStatus[rs]
}