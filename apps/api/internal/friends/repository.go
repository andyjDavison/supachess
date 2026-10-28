package friends

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"api/internal/domain"

	"github.com/jackc/pgx/v5/pgconn"
)

const (
	uniqueViolation           = "23505"
	foreignKeyViolation       = "23503"
	checkViolation            = "23514"
	invalidTextRepresentation = "22P02"
)

// FriendRepository is the persistence seam FriendService depends on, so the
// service can run against an in-memory implementation in tests.
//
// Pair methods (FindFriendship, DeleteFriendship) match the pair in either
// direction. Directional ones (CreateRequest, AcceptRequest) do not.
type FriendRepository interface {
	// FindFriendship returns ErrFriendRequestNotFound when the pair has no row.
	FindFriendship(ctx context.Context, a, b string) (*domain.Friendship, error)
	CreateRequest(ctx context.Context, requesterID, addresseeID string) error
	// AcceptRequest returns ErrFriendRequestNotFound unless a pending
	// requester->addressee row exists.
	AcceptRequest(ctx context.Context, requesterID, addresseeID string) error
	// DeleteFriendship removes the pair's row only if it has the given
	// status, reporting whether a row was removed.
	DeleteFriendship(ctx context.Context, a, b string, status domain.FriendshipStatus) (bool, error)
	ListFriends(ctx context.Context, userID string) ([]domain.Friend, error)
	ListIncoming(ctx context.Context, userID string) ([]domain.FriendRequest, error)
	ListOutgoing(ctx context.Context, userID string) ([]domain.FriendRequest, error)
	IsBlockedEitherWay(ctx context.Context, a, b string) (bool, error)
	// Block clears any friendship or request between the pair and records
	// the block, atomically. Blocking twice is not an error.
	Block(ctx context.Context, blockerID, blockedID string) error
	Unblock(ctx context.Context, blockerID, blockedID string) error
	ListBlocked(ctx context.Context, userID string) ([]domain.BlockedUser, error)
	// SearchUsers matches a username prefix case-insensitively, excluding
	// the searcher and anyone blocked in either direction.
	SearchUsers(ctx context.Context, userID, prefix string, limit int) ([]domain.UserSearchResult, error)
}

type PostgresFriendRepository struct {
	db *sql.DB
}

func NewPostgresFriendRepository(db *sql.DB) *PostgresFriendRepository {
	return &PostgresFriendRepository{db: db}
}

// mapPgError turns the Postgres failures a caller can cause into sentinels.
// A malformed id (22P02) or a reference to a user that doesn't exist (23503)
// both mean "no such user" - the same reasoning as game.isNoSuchGame.
func mapPgError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	switch pgErr.Code {
	case invalidTextRepresentation, foreignKeyViolation:
		return domain.ErrUserNotFound
	case uniqueViolation:
		// Either the primary key (same direction) or friendships_pair_key
		// (the reverse direction, e.g. two users requesting each other at
		// once). Both mean a row for this pair already exists.
		return domain.ErrFriendRequestExists
	case checkViolation:
		return domain.ErrCannotFriendSelf
	}
	return err
}

// prefixPattern lowercases q, escapes LIKE's wildcards, and appends %, so a
// username containing _ or % is matched literally. Built here rather than in
// SQL so the planner sees a constant prefix it can use the index for.
func prefixPattern(q string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(strings.ToLower(q)) + "%"
}

func (r *PostgresFriendRepository) FindFriendship(ctx context.Context, a, b string) (*domain.Friendship, error) {
	const query = `
		SELECT requester_id, addressee_id, status, created_at, accepted_at
		FROM friendships
		WHERE (requester_id = $1 AND addressee_id = $2)
		   OR (requester_id = $2 AND addressee_id = $1)
	`

	var f domain.Friendship
	var acceptedAt sql.NullTime
	err := r.db.QueryRowContext(ctx, query, a, b).
		Scan(&f.RequesterID, &f.AddresseeID, &f.Status, &f.CreatedAt, &acceptedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrFriendRequestNotFound
		}
		if mapped := mapPgError(err); mapped != err {
			return nil, mapped
		}
		return nil, fmt.Errorf("finding friendship: %w", err)
	}
	if acceptedAt.Valid {
		f.AcceptedAt = &acceptedAt.Time
	}
	return &f, nil
}

func (r *PostgresFriendRepository) CreateRequest(ctx context.Context, requesterID, addresseeID string) error {
	const query = `
		INSERT INTO friendships (requester_id, addressee_id, status)
		VALUES ($1, $2, 'pending')
	`

	if _, err := r.db.ExecContext(ctx, query, requesterID, addresseeID); err != nil {
		if mapped := mapPgError(err); mapped != err {
			return mapped
		}
		return fmt.Errorf("creating friend request: %w", err)
	}
	return nil
}

func (r *PostgresFriendRepository) AcceptRequest(ctx context.Context, requesterID, addresseeID string) error {
	const query = `
		UPDATE friendships
		SET status = 'accepted', accepted_at = now()
		WHERE requester_id = $1 AND addressee_id = $2 AND status = 'pending'
	`

	res, err := r.db.ExecContext(ctx, query, requesterID, addresseeID)
	if err != nil {
		if errors.Is(mapPgError(err), domain.ErrUserNotFound) {
			return domain.ErrFriendRequestNotFound
		}
		return fmt.Errorf("accepting friend request: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking accept result: %w", err)
	}
	if n == 0 {
		return domain.ErrFriendRequestNotFound
	}
	return nil
}

func (r *PostgresFriendRepository) DeleteFriendship(ctx context.Context, a, b string, status domain.FriendshipStatus) (bool, error) {
	const query = `
		DELETE FROM friendships
		WHERE ((requester_id = $1 AND addressee_id = $2)
		    OR (requester_id = $2 AND addressee_id = $1))
		  AND status = $3
	`

	res, err := r.db.ExecContext(ctx, query, a, b, status)
	if err != nil {
		// A malformed id can't match a row; that's "nothing deleted".
		if errors.Is(mapPgError(err), domain.ErrUserNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("deleting friendship: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("checking delete result: %w", err)
	}
	return n > 0, nil
}

func (r *PostgresFriendRepository) ListFriends(ctx context.Context, userID string) ([]domain.Friend, error) {
	const query = `
		SELECT u.id, u.username, u.rating, f.accepted_at
		FROM friendships f
		JOIN users u ON u.id = CASE WHEN f.requester_id = $1 THEN f.addressee_id ELSE f.requester_id END
		WHERE f.status = 'accepted' AND (f.requester_id = $1 OR f.addressee_id = $1)
		ORDER BY lower(u.username)
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("listing friends: %w", err)
	}
	defer rows.Close()

	friends := []domain.Friend{}
	for rows.Next() {
		var f domain.Friend
		if err := rows.Scan(&f.UserID, &f.Username, &f.Rating, &f.Since); err != nil {
			return nil, fmt.Errorf("scanning friend: %w", err)
		}
		friends = append(friends, f)
	}
	return friends, rows.Err()
}

func (r *PostgresFriendRepository) ListIncoming(ctx context.Context, userID string) ([]domain.FriendRequest, error) {
	const query = `
		SELECT u.id, u.username, u.rating, f.created_at
		FROM friendships f
		JOIN users u ON u.id = f.requester_id
		WHERE f.status = 'pending' AND f.addressee_id = $1
		ORDER BY f.created_at DESC
	`
	return r.listRequests(ctx, query, userID)
}

func (r *PostgresFriendRepository) ListOutgoing(ctx context.Context, userID string) ([]domain.FriendRequest, error) {
	const query = `
		SELECT u.id, u.username, u.rating, f.created_at
		FROM friendships f
		JOIN users u ON u.id = f.addressee_id
		WHERE f.status = 'pending' AND f.requester_id = $1
		ORDER BY f.created_at DESC
	`
	return r.listRequests(ctx, query, userID)
}

func (r *PostgresFriendRepository) listRequests(ctx context.Context, query, userID string) ([]domain.FriendRequest, error) {
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("listing friend requests: %w", err)
	}
	defer rows.Close()

	requests := []domain.FriendRequest{}
	for rows.Next() {
		var fr domain.FriendRequest
		if err := rows.Scan(&fr.UserID, &fr.Username, &fr.Rating, &fr.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning friend request: %w", err)
		}
		requests = append(requests, fr)
	}
	return requests, rows.Err()
}

func (r *PostgresFriendRepository) IsBlockedEitherWay(ctx context.Context, a, b string) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1 FROM blocks
			WHERE (blocker_id = $1 AND blocked_id = $2)
			   OR (blocker_id = $2 AND blocked_id = $1)
		)
	`

	var blocked bool
	if err := r.db.QueryRowContext(ctx, query, a, b).Scan(&blocked); err != nil {
		if mapped := mapPgError(err); mapped != err {
			return false, mapped
		}
		return false, fmt.Errorf("checking blocks: %w", err)
	}
	return blocked, nil
}

func (r *PostgresFriendRepository) Block(ctx context.Context, blockerID, blockedID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning block transaction: %w", err)
	}
	defer tx.Rollback()

	const clearFriendship = `
		DELETE FROM friendships
		WHERE (requester_id = $1 AND addressee_id = $2)
		   OR (requester_id = $2 AND addressee_id = $1)
	`
	if _, err := tx.ExecContext(ctx, clearFriendship, blockerID, blockedID); err != nil {
		if mapped := mapPgError(err); mapped != err {
			return mapped
		}
		return fmt.Errorf("clearing friendship before block: %w", err)
	}

	const insertBlock = `
		INSERT INTO blocks (blocker_id, blocked_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`
	if _, err := tx.ExecContext(ctx, insertBlock, blockerID, blockedID); err != nil {
		if mapped := mapPgError(err); mapped != err {
			return mapped
		}
		return fmt.Errorf("inserting block: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing block: %w", err)
	}
	return nil
}

func (r *PostgresFriendRepository) Unblock(ctx context.Context, blockerID, blockedID string) error {
	const query = `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`

	if _, err := r.db.ExecContext(ctx, query, blockerID, blockedID); err != nil {
		// Unblocking a malformed id is a no-op, same as one never blocked.
		if errors.Is(mapPgError(err), domain.ErrUserNotFound) {
			return nil
		}
		return fmt.Errorf("unblocking: %w", err)
	}
	return nil
}

func (r *PostgresFriendRepository) ListBlocked(ctx context.Context, userID string) ([]domain.BlockedUser, error) {
	const query = `
		SELECT u.id, u.username, b.created_at
		FROM blocks b
		JOIN users u ON u.id = b.blocked_id
		WHERE b.blocker_id = $1
		ORDER BY lower(u.username)
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("listing blocked users: %w", err)
	}
	defer rows.Close()

	blocked := []domain.BlockedUser{}
	for rows.Next() {
		var b domain.BlockedUser
		if err := rows.Scan(&b.UserID, &b.Username, &b.BlockedAt); err != nil {
			return nil, fmt.Errorf("scanning blocked user: %w", err)
		}
		blocked = append(blocked, b)
	}
	return blocked, rows.Err()
}

func (r *PostgresFriendRepository) SearchUsers(ctx context.Context, userID, prefix string, limit int) ([]domain.UserSearchResult, error) {
	const query = `
		SELECT u.id, u.username, u.rating,
			CASE
				WHEN f.status = 'accepted' THEN 'friend'
				WHEN f.requester_id = $1   THEN 'outgoing'
				WHEN f.addressee_id = $1   THEN 'incoming'
				ELSE 'none'
			END
		FROM users u
		LEFT JOIN friendships f
			ON (f.requester_id = $1 AND f.addressee_id = u.id)
			OR (f.addressee_id = $1 AND f.requester_id = u.id)
		WHERE lower(u.username) LIKE $2 ESCAPE '\'
		  AND u.id <> $1
		  AND NOT EXISTS (
			SELECT 1 FROM blocks b
			WHERE (b.blocker_id = $1 AND b.blocked_id = u.id)
			   OR (b.blocker_id = u.id AND b.blocked_id = $1)
		  )
		ORDER BY lower(u.username)
		LIMIT $3
	`

	rows, err := r.db.QueryContext(ctx, query, userID, prefixPattern(prefix), limit)
	if err != nil {
		return nil, fmt.Errorf("searching users: %w", err)
	}
	defer rows.Close()

	results := []domain.UserSearchResult{}
	for rows.Next() {
		var res domain.UserSearchResult
		if err := rows.Scan(&res.UserID, &res.Username, &res.Rating, &res.Relationship); err != nil {
			return nil, fmt.Errorf("scanning search result: %w", err)
		}
		results = append(results, res)
	}
	return results, rows.Err()
}
