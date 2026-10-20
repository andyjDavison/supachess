package game

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

// isNoSuchGame is the fix for the malformed-id bug: /api/games/not-a-uuid
// used to answer 500, because games.id is a uuid column and a value that
// cannot be coerced to one raises 22P02 rather than matching zero rows.
func TestIsNoSuchGame(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"no rows", sql.ErrNoRows, true},
		{"wrapped no rows", fmt.Errorf("querying: %w", sql.ErrNoRows), true},
		{"malformed uuid", &pgconn.PgError{Code: "22P02"}, true},
		{"wrapped malformed uuid", fmt.Errorf("finding game: %w", &pgconn.PgError{Code: "22P02"}), true},

		// Everything else is a real failure and must keep its 500.
		{"unique violation", &pgconn.PgError{Code: "23505"}, false},
		{"undefined table", &pgconn.PgError{Code: "42P01"}, false},
		{"connection failure", errors.New("connection refused"), false},
		{"nil", nil, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isNoSuchGame(c.err); got != c.want {
				t.Errorf("isNoSuchGame(%v) = %v, want %v", c.err, got, c.want)
			}
		})
	}
}
