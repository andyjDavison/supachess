package chat

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

// A malformed id or a reference to a missing row is the caller's mistake
// (a 404), not the server's (a 500).
func TestIsBadReference(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"malformed uuid", &pgconn.PgError{Code: "22P02"}, true},
		{"missing referenced row", &pgconn.PgError{Code: "23503"}, true},
		{"wrapped", fmt.Errorf("exec: %w", &pgconn.PgError{Code: "23503"}), true},
		{"unique violation", &pgconn.PgError{Code: "23505"}, false},
		{"no rows", sql.ErrNoRows, false},
		{"connection failure", errors.New("connection refused"), false},
		{"nil", nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isBadReference(c.err); got != c.want {
				t.Errorf("isBadReference(%v) = %v, want %v", c.err, got, c.want)
			}
		})
	}
}
