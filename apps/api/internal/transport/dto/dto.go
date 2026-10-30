package dto

import (
	"api/internal/domain"
	"api/internal/rating"
)

type GameDTO struct {
	ID                   string          `json:"id"`
	WhiteID              string          `json:"whiteId"`
	BlackID              string          `json:"blackId"`
	FEN                  string          `json:"fen"`
	Status               string          `json:"status"`
	Result               string          `json:"result,omitempty"`
	ResultReason         string          `json:"resultReason,omitempty"`
	WhiteTimeRemainingMs int64           `json:"whiteTimeRemainingMs"`
	BlackTimeRemainingMs int64           `json:"blackTimeRemainingMs"`
	InitialMs            int64           `json:"initialMs"`   // time control: starting clock
	IncrementMs          int64           `json:"incrementMs"` // time control: added per move
	LastMoveAt           int64           `json:"lastMoveAt"`  // unix millis
	FinishedAt           *int64          `json:"finishedAt,omitempty"`
	Ratings              *GameRatingsDTO `json:"ratings,omitempty"` // nil for games from before ratings
}

type GameRatingsDTO struct {
	Category domain.RatingCategory `json:"category"`
	White    SideRatingDTO         `json:"white"`
	Black    SideRatingDTO         `json:"black"`
}

// SideRatingDTO is one player's rating at the start of the game, what
// each result would change it by, and - once finished - what it did.
type SideRatingDTO struct {
	Rating int32  `json:"rating"`
	Win    int32  `json:"win"`
	Draw   int32  `json:"draw"`
	Loss   int32  `json:"loss"`
	Delta  *int32 `json:"delta,omitempty"`
}

func newSideRatingDTO(me, opp domain.SideRating) SideRatingDTO {
	stakes := rating.StakesFor(me.Before, opp.Before, me.K)
	return SideRatingDTO{Rating: me.Before, Win: stakes.Win, Draw: stakes.Draw, Loss: stakes.Loss, Delta: me.Delta}
}

func NewGameDTO(g *domain.Game) GameDTO {
	dto := GameDTO{
		ID:                   string(g.ID),
		WhiteID:              string(g.WhiteID),
		BlackID:              string(g.BlackID),
		FEN:                  g.FEN,
		Status:               string(g.Status),
		Result:               string(g.Result),
		ResultReason:         string(g.ResultReason),
		WhiteTimeRemainingMs: g.WhiteTimeRemaining.Milliseconds(),
		BlackTimeRemainingMs: g.BlackTimeRemaining.Milliseconds(),
		InitialMs:            g.TimeControl.Initial.Milliseconds(),
		IncrementMs:          g.TimeControl.Increment.Milliseconds(),
		LastMoveAt:           g.LastMoveAt.UnixMilli(),
	}
	if g.FinishedAt != nil {
		ms := g.FinishedAt.UnixMilli()
		dto.FinishedAt = &ms
	}
	if r := g.Ratings; r != nil {
		dto.Ratings = &GameRatingsDTO{
			Category: r.Category,
			White:    newSideRatingDTO(r.White, r.Black),
			Black:    newSideRatingDTO(r.Black, r.White),
		}
	}
	return dto
}

type MoveDTO struct {
	GameID    string `json:"gameId"`
	Ply       int    `json:"ply"`
	Color     string `json:"color"`
	From      string `json:"from"`
	To        string `json:"to"`
	Promotion string `json:"promotion,omitempty"`
	SAN       string `json:"san"`
	FENAfter  string `json:"fenAfter"`
	PlayedAt  int64  `json:"playedAt"`
}

func NewMoveDTO(m *domain.Move) MoveDTO {
	return MoveDTO{
		GameID:    string(m.GameID),
		Ply:       m.Ply,
		Color:     string(m.Color),
		From:      string(m.From),
		To:        string(m.To),
		Promotion: string(m.Promotion),
		SAN:       m.SAN,
		FENAfter:  m.FENAfter,
		PlayedAt:  m.PlayedAt.UnixMilli(),
	}
}
