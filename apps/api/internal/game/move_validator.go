package game

import (
	"api/internal/domain"
)

type PositionStatus string

const (
	PositionOngoing                  PositionStatus = "ongoing"
	PositionCheck                    PositionStatus = "check"
	PositionCheckmate                PositionStatus = "checkmate"
	PositionStalemate                PositionStatus = "stalemate"
	PositionDrawRepetition           PositionStatus = "draw_repetition"
	PositionDrawFiftyMoveRule        PositionStatus = "draw_fifty_move_rule"
	PositionDrawInsufficientMaterial PositionStatus = "draw_insufficient_material"
)

// MoveResult is what ValidateMove returns for a legal move.
type MoveResult struct {
	FEN    string
	SAN    string
	Status PositionStatus
}

// MoveValidator is the seam between Service and whichever chess rules
// engine actually implements it (e.g. a wrapper around notnil/chess).
// Service depends only on this interface, never on a concrete rules
// library directly — same reasoning as PasswordHasher wrapping bcrypt.
type MoveValidator interface {
	// ValidateMove checks whether from -> to (with an optional promotion
	// piece, only meaningful when a pawn reaches the back rank) is legal
	// given the game's move history and current position. history is
	// required, not optional — draw-by-repetition can't be determined
	// from a single FEN alone, since it depends on prior positions.
	//
	// Returns domain.ErrIllegalMove if the move isn't legal here.
	ValidateMove(
		history []domain.Move,
		currentFEN string,
		from, to domain.Square,
		promotion domain.PieceType,
	) (MoveResult, error)
}