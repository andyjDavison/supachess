package game

import (
	"testing"

	"api/internal/domain"
)

func TestActiveColorFromFEN(t *testing.T) {
	cases := []struct {
		name string
		fen  string
		want domain.Color
	}{
		{"starting position is white to move", domain.StartingFEN, domain.White},
		{"black to move", "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1", domain.Black},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := activeColorFromFEN(c.fen)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestActiveColorFromFENRejectsMalformed(t *testing.T) {
	cases := []struct {
		name string
		fen  string
	}{
		{"empty", ""},
		{"board only, no colour field", "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR"},
		{"colour field is not w or b", "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR x KQkq - 0 1"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := activeColorFromFEN(c.fen); err == nil {
				t.Errorf("activeColorFromFEN(%q) returned no error", c.fen)
			}
		})
	}
}
