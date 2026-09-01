package domain

import "time"

type PieceType string

const (
	Queen  PieceType = "queen"
	Rook   PieceType = "rook"
	Bishop PieceType = "bishop"
	Knight PieceType = "knight"
)

// Square is algebraic square notation, e.g. "e2", "e4".
type Square string

type Move struct {
	GameID    GameID
	Ply       int
	Color     Color
	From      Square
	To        Square
	Promotion PieceType // empty unless this move is a pawn promotion
	SAN       string
	FENAfter  string
	PlayedAt  time.Time
}