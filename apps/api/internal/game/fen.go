// internal/game/fen.go
package game

import (
	"fmt"
	"strings"

	"api/internal/domain"
)

func activeColorFromFEN(fen string) (domain.Color, error) {
	fields := strings.Fields(fen)
	if len(fields) < 2 {
		return "", fmt.Errorf("invalid FEN: %q", fen)
	}
	switch fields[1] {
	case "w":
		return domain.White, nil
	case "b":
		return domain.Black, nil
	default:
		return "", fmt.Errorf("invalid FEN active color: %q", fields[1])
	}
}