package game

import (
	"errors"
	"strings"
	"testing"

	"api/internal/domain"
)

// These run against the real notnil/chess engine - the point is to pin the
// translation layer (UCI encoding, SAN, PositionStatus mapping), not to
// re-test the library's rules.

func TestValidateMoveLegalOpening(t *testing.T) {
	v := NewNotnilMoveValidator()

	res, err := v.ValidateMove(nil, domain.StartingFEN, "e2", "e4", "")
	if err != nil {
		t.Fatalf("e2e4 from the starting position: %v", err)
	}
	if res.SAN != "e4" {
		t.Errorf("SAN = %q, want e4", res.SAN)
	}
	if res.Status != PositionOngoing {
		t.Errorf("Status = %q, want ongoing", res.Status)
	}
	// The returned FEN must be the position *after* the move, with black to play.
	if !strings.Contains(res.FEN, " b ") {
		t.Errorf("FEN after white's move = %q, want black to move", res.FEN)
	}
}

func TestValidateMoveRejectsIllegalAndMalformed(t *testing.T) {
	v := NewNotnilMoveValidator()

	cases := []struct {
		name     string
		from, to domain.Square
	}{
		{"pawn cannot jump three", "e2", "e5"},
		{"no piece on the from square", "e4", "e5"},
		{"moving onto your own piece", "a1", "a2"},
		{"nonsense squares", "zz", "99"},
		{"empty squares", "", ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := v.ValidateMove(nil, domain.StartingFEN, c.from, c.to, "")
			if !errors.Is(err, domain.ErrIllegalMove) {
				t.Errorf("got %v, want ErrIllegalMove", err)
			}
		})
	}
}

func TestValidateMoveRejectsUnparseableFEN(t *testing.T) {
	v := NewNotnilMoveValidator()

	if _, err := v.ValidateMove(nil, "this is not a fen", "e2", "e4", ""); err == nil {
		t.Error("expected an error for an unparseable FEN")
	}
}

func TestValidateMovePromotion(t *testing.T) {
	// White pawn on a7, both kings clear of it.
	const fen = "8/P6k/8/8/8/8/8/K7 w - - 0 1"
	v := NewNotnilMoveValidator()

	cases := []struct {
		piece   domain.PieceType
		wantSAN string
	}{
		{domain.Queen, "a8=Q"},
		{domain.Rook, "a8=R"},
		{domain.Bishop, "a8=B"},
		{domain.Knight, "a8=N"},
	}

	for _, c := range cases {
		t.Run(string(c.piece), func(t *testing.T) {
			res, err := v.ValidateMove(nil, fen, "a7", "a8", c.piece)
			if err != nil {
				t.Fatalf("promoting to %s: %v", c.piece, err)
			}
			if res.SAN != c.wantSAN {
				t.Errorf("SAN = %q, want %q", res.SAN, c.wantSAN)
			}
		})
	}
}

func TestValidateMoveCastling(t *testing.T) {
	const fen = "r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1"
	v := NewNotnilMoveValidator()

	res, err := v.ValidateMove(nil, fen, "e1", "g1", "")
	if err != nil {
		t.Fatalf("kingside castle: %v", err)
	}
	if res.SAN != "O-O" {
		t.Errorf("SAN = %q, want O-O", res.SAN)
	}
}

func TestValidateMoveEnPassant(t *testing.T) {
	// Black just played d7d5; e5 pawn can take en passant on d6.
	const fen = "rnbqkbnr/ppp1pppp/8/3pP3/8/8/PPPP1PPP/RNBQKBNR w KQkq d6 0 3"
	v := NewNotnilMoveValidator()

	res, err := v.ValidateMove(nil, fen, "e5", "d6", "")
	if err != nil {
		t.Fatalf("en passant capture: %v", err)
	}
	if res.SAN != "exd6" {
		t.Errorf("SAN = %q, want exd6", res.SAN)
	}
}

// The Check tag is only populated during real move generation, which is
// why ValidateMove applies the matched legal move rather than the decoded
// one. This test fails if that ever regresses.
func TestValidateMoveDetectsCheck(t *testing.T) {
	const fen = "4k3/8/8/8/8/8/8/4K2R w K - 0 1"
	v := NewNotnilMoveValidator()

	res, err := v.ValidateMove(nil, fen, "h1", "h8", "")
	if err != nil {
		t.Fatalf("Rh8+: %v", err)
	}
	if res.Status != PositionCheck {
		t.Errorf("Status = %q, want check", res.Status)
	}
}

func TestValidateMoveDetectsCheckmate(t *testing.T) {
	// Fool's mate, one move from completion: Qh4#.
	const fen = "rnbqkbnr/pppp1ppp/8/4p3/6P1/5P2/PPPPP2P/RNBQKBNR b KQkq - 0 2"
	v := NewNotnilMoveValidator()

	res, err := v.ValidateMove(nil, fen, "d8", "h4", "")
	if err != nil {
		t.Fatalf("Qh4#: %v", err)
	}
	if res.Status != PositionCheckmate {
		t.Errorf("Status = %q, want checkmate", res.Status)
	}
}

func TestValidateMoveDetectsStalemate(t *testing.T) {
	// Black king on a8; white queen to b6 leaves black with no legal move.
	const fen = "k7/8/1Q6/8/8/8/8/K6R w - - 0 1"
	v := NewNotnilMoveValidator()

	res, err := v.ValidateMove(nil, fen, "b6", "c7", "")
	if err != nil {
		t.Fatalf("Qc7: %v", err)
	}
	if res.Status != PositionStalemate {
		t.Errorf("Status = %q, want stalemate (got FEN %q)", res.Status, res.FEN)
	}
}

func TestValidateMoveDetectsInsufficientMaterial(t *testing.T) {
	// White king on a1 takes the lone black knight on b2; the black king
	// is on h8 and does not defend it. What remains is king vs king.
	const fen = "7k/8/8/8/8/8/1n6/K7 w - - 0 1"
	v := NewNotnilMoveValidator()

	res, err := v.ValidateMove(nil, fen, "a1", "b2", "")
	if err != nil {
		t.Fatalf("Kxb2: %v", err)
	}
	if res.Status != PositionDrawInsufficientMaterial {
		t.Errorf("Status = %q, want insufficient material (FEN %q)", res.Status, res.FEN)
	}
}
