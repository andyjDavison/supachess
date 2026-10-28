package friends

import (
	"errors"
	"fmt"
	"testing"

	"api/internal/domain"
	"api/internal/friends/friendstest"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestMapPgError(t *testing.T) {
	plain := errors.New("connection refused")
	cases := []struct {
		name string
		err  error
		want error
	}{
		{"malformed uuid", &pgconn.PgError{Code: "22P02"}, domain.ErrUserNotFound},
		{"unknown user (fk)", &pgconn.PgError{Code: "23503"}, domain.ErrUserNotFound},
		{"same-direction duplicate", &pgconn.PgError{Code: "23505", ConstraintName: "friendships_pkey"}, domain.ErrFriendRequestExists},
		{"reverse-direction duplicate", &pgconn.PgError{Code: "23505", ConstraintName: "friendships_pair_key"}, domain.ErrFriendRequestExists},
		{"self reference", &pgconn.PgError{Code: "23514"}, domain.ErrCannotFriendSelf},
		{"wrapped", fmt.Errorf("exec: %w", &pgconn.PgError{Code: "23503"}), domain.ErrUserNotFound},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := mapPgError(c.err); !errors.Is(got, c.want) {
				t.Errorf("mapPgError = %v, want %v", got, c.want)
			}
		})
	}

	// Anything else passes through untouched so callers can wrap it.
	for _, err := range []error{plain, &pgconn.PgError{Code: "42P01"}} {
		if got := mapPgError(err); got != err {
			t.Errorf("mapPgError(%v) = %v, want it unchanged", err, got)
		}
	}
}

// A username containing LIKE wildcards must match literally, or searching
// "a_" would also match "ab".
func TestPrefixPattern(t *testing.T) {
	cases := map[string]string{
		"Alice":  "alice%",
		"a_b":    `a\_b%`,
		"100%":   `100\%%`,
		`back\s`: `back\\s%`,
	}
	for in, want := range cases {
		if got := prefixPattern(in); got != want {
			t.Errorf("prefixPattern(%q) = %q, want %q", in, got, want)
		}
	}
}

var (
	_ FriendRepository = (*PostgresFriendRepository)(nil)
	_ FriendRepository = (*friendstest.Repo)(nil)
)
