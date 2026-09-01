package game

import (
	"fmt"

	"api/internal/domain"

	"github.com/notnil/chess"
)

// NotnilMoveValidator implements MoveValidator using
// github.com/notnil/chess as the underlying rules engine. This is the
// only file in the game package allowed to import that library —
// everything else depends on the MoveValidator interface, never on
// chess.* types directly.
type NotnilMoveValidator struct{}

func NewNotnilMoveValidator() *NotnilMoveValidator {
	return &NotnilMoveValidator{}
}

func (v *NotnilMoveValidator) ValidateMove(
	history []domain.Move,
	currentFEN string,
	from, to domain.Square,
	promotion domain.PieceType,
) (MoveResult, error) {
	fenOpt, err := chess.FEN(currentFEN)
	if err != nil {
		return MoveResult{}, fmt.Errorf("parsing current FEN: %w", err)
	}
	g := chess.NewGame(fenOpt)

	// UCINotation is notnil/chess's from-square/to-square format
	// ("e2e4", "e7e8q" for promotion) — the right fit given our
	// interface takes from/to squares rather than SAN.
	uci := string(from) + string(to) + promotionChar(promotion)

	decoded, err := chess.UCINotation{}.Decode(g.Position(), uci)
	if err != nil {
		// Malformed square names, not a legality question — still just
		// "reject", per how we scoped this interface's error handling.
		return MoveResult{}, domain.ErrIllegalMove
	}

	// Decode alone only reconstructs enough of a Move to identify it
	// (from/to/promo, plus capture/castle/en-passant tags) — it does
	// NOT tell us whether it's actually legal, or compute tags like
	// Check, which only get set during real move generation. So the
	// decoded move gets matched against the position's actual legal
	// moves (same lookup notnil/chess itself uses internally), and
	// that matched move — not the decoded one — is what gets applied.
	legalMove := findMatchingMove(g, decoded)
	if legalMove == nil {
		return MoveResult{}, domain.ErrIllegalMove
	}

	// SAN must be encoded from the position BEFORE the move is applied
	// — algebraic notation is only meaningful relative to the position
	// it was played from.
	san := chess.AlgebraicNotation{}.Encode(g.Position(), legalMove)

	if err := g.Move(legalMove); err != nil {
		// Shouldn't actually happen, since legalMove came from
		// ValidMoves() — defensive rather than trusted blindly.
		return MoveResult{}, domain.ErrIllegalMove
	}

	return MoveResult{
		FEN:    g.Position().String(),
		SAN:    san,
		Status: classifyPosition(g, legalMove),
	}, nil
}

func findMatchingMove(g *chess.Game, target *chess.Move) *chess.Move {
	for _, m := range g.ValidMoves() {
		if m.String() == target.String() {
			return m
		}
	}
	return nil
}

func promotionChar(p domain.PieceType) string {
	switch p {
	case domain.Queen:
		return "q"
	case domain.Rook:
		return "r"
	case domain.Bishop:
		return "b"
	case domain.Knight:
		return "n"
	default:
		return ""
	}
}

// classifyPosition maps notnil/chess's Method/Outcome and the just-played
// move's tags onto our own PositionStatus — the translation boundary that
// keeps the library's types from leaking past this file. Must be called
// AFTER g.Move(move) so Method()/Outcome() reflect the resulting position.
func classifyPosition(g *chess.Game, playedMove *chess.Move) PositionStatus {
	switch g.Method() {
	case chess.Checkmate:
		return PositionCheckmate
	case chess.Stalemate:
		return PositionStalemate
	case chess.ThreefoldRepetition, chess.FivefoldRepetition:
		return PositionDrawRepetition
	case chess.FiftyMoveRule, chess.SeventyFiveMoveRule:
		return PositionDrawFiftyMoveRule
	case chess.InsufficientMaterial:
		return PositionDrawInsufficientMaterial
	}

	// Not a game-ending method — check whether this specific move put
	// the opponent in check. The Check tag is computed during move
	// generation (part of ValidMoves()), which is exactly why legalMove
	// — not the hand-decoded one — had to be the move actually applied.
	if playedMove.HasTag(chess.Check) {
		return PositionCheck
	}
	return PositionOngoing
}