package ws

import "api/internal/domain"

type gameDTO struct {
	ID                    string  `json:"id"`
	WhiteID               string  `json:"whiteId"`
	BlackID               string  `json:"blackId"`
	FEN                   string  `json:"fen"`
	Status                string  `json:"status"`
	Result                string  `json:"result,omitempty"`
	ResultReason          string  `json:"resultReason,omitempty"`
	WhiteTimeRemainingMs  int64   `json:"whiteTimeRemainingMs"`
	BlackTimeRemainingMs  int64   `json:"blackTimeRemainingMs"`
	LastMoveAt            int64   `json:"lastMoveAt"` // unix millis
	FinishedAt            *int64  `json:"finishedAt,omitempty"`
}

func newGameDTO(g *domain.Game) gameDTO {
	dto := gameDTO{
		ID:                   string(g.ID),
		WhiteID:              string(g.WhiteID),
		BlackID:              string(g.BlackID),
		FEN:                  g.FEN,
		Status:               string(g.Status),
		Result:               string(g.Result),
		ResultReason:         string(g.ResultReason),
		WhiteTimeRemainingMs: g.WhiteTimeRemaining.Milliseconds(),
		BlackTimeRemainingMs: g.BlackTimeRemaining.Milliseconds(),
		LastMoveAt:           g.LastMoveAt.UnixMilli(),
	}
	if g.FinishedAt != nil {
		ms := g.FinishedAt.UnixMilli()
		dto.FinishedAt = &ms
	}
	return dto
}

type moveDTO struct {
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

func newMoveDTO(m *domain.Move) moveDTO {
	return moveDTO{
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