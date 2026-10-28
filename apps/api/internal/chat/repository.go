package chat

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"

	"api/internal/domain"

	"github.com/jackc/pgx/v5/pgconn"
)

const (
	foreignKeyViolation       = "23503"
	invalidTextRepresentation = "22P02"
)

// ChatRepository is the persistence seam ChatService depends on, so the
// service can run against an in-memory implementation in tests.
type ChatRepository interface {
	// AppendDirectMessage stores a message from fromID to toID, creating
	// the pair's conversation on first contact. In the same transaction it
	// marks the sender as having replied (moving the conversation to their
	// Inbox), marks it read for them, and un-archives it for both members.
	// The returned message carries the conversation's ID.
	AppendDirectMessage(ctx context.Context, fromID, toID, body string) (*domain.DirectMessage, error)
	// ListConversations returns userID's conversations, newest activity
	// first, leaving out any with a user userID has blocked.
	ListConversations(ctx context.Context, userID string, limit int) ([]domain.ConversationSummary, error)
	// GetConversation returns userID's view of one conversation, or
	// ErrConversationNotFound if userID isn't in it (or it's hidden by a block).
	GetConversation(ctx context.Context, convID, userID string) (*domain.ConversationSummary, error)
	// ListDirectMessages returns up to limit messages older than beforeID
	// (0 for the newest), oldest first.
	ListDirectMessages(ctx context.Context, convID string, beforeID int64, limit int) ([]domain.DirectMessage, error)
	MarkRead(ctx context.Context, convID, userID string) error
	SetArchived(ctx context.Context, convID, userID string, archived bool) error
	AppendGameMessage(ctx context.Context, gameID, senderID, body string) (*domain.GameMessage, error)
	ListGameMessages(ctx context.Context, gameID string, limit int) ([]domain.GameMessage, error)
}

type PostgresChatRepository struct {
	db *sql.DB
}

func NewPostgresChatRepository(db *sql.DB) *PostgresChatRepository {
	return &PostgresChatRepository{db: db}
}

// isBadReference reports whether err means an id didn't refer to anything:
// a malformed uuid (22P02) or a foreign key to a missing row (23503).
func isBadReference(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && (pgErr.Code == invalidTextRepresentation || pgErr.Code == foreignKeyViolation)
}

func (r *PostgresChatRepository) AppendDirectMessage(ctx context.Context, fromID, toID, body string) (*domain.DirectMessage, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("beginning message transaction: %w", err)
	}
	defer tx.Rollback()

	const upsertConversation = `
		INSERT INTO conversations (user_lo, user_hi)
		VALUES (LEAST($1::uuid, $2::uuid), GREATEST($1::uuid, $2::uuid))
		ON CONFLICT (user_lo, user_hi) DO UPDATE SET last_message_at = now()
		RETURNING id
	`
	var convID string
	if err := tx.QueryRowContext(ctx, upsertConversation, fromID, toID).Scan(&convID); err != nil {
		if isBadReference(err) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("upserting conversation: %w", err)
	}

	const upsertMembers = `
		INSERT INTO conversation_members (conversation_id, user_id)
		VALUES ($1, $2), ($1, $3)
		ON CONFLICT DO NOTHING
	`
	if _, err := tx.ExecContext(ctx, upsertMembers, convID, fromID, toID); err != nil {
		return nil, fmt.Errorf("upserting conversation members: %w", err)
	}

	const insertMessage = `
		INSERT INTO direct_messages (conversation_id, sender_id, body)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`
	msg := domain.DirectMessage{ConversationID: convID, SenderID: fromID, Body: body}
	if err := tx.QueryRowContext(ctx, insertMessage, convID, fromID, body).Scan(&msg.ID, &msg.SentAt); err != nil {
		return nil, fmt.Errorf("inserting message: %w", err)
	}

	// Sending is replying, and you've obviously read up to your own message.
	const updateSender = `
		UPDATE conversation_members
		SET replied = true, archived_at = NULL, last_read_at = $3
		WHERE conversation_id = $1 AND user_id = $2
	`
	if _, err := tx.ExecContext(ctx, updateSender, convID, fromID, msg.SentAt); err != nil {
		return nil, fmt.Errorf("updating sender state: %w", err)
	}

	const unarchiveRecipient = `
		UPDATE conversation_members SET archived_at = NULL
		WHERE conversation_id = $1 AND user_id = $2
	`
	if _, err := tx.ExecContext(ctx, unarchiveRecipient, convID, toID); err != nil {
		return nil, fmt.Errorf("updating recipient state: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("committing message: %w", err)
	}
	return &msg, nil
}

// summaryQuery is one member's view of their conversations. $1 is the
// member; callers append further conditions and the ORDER/LIMIT.
const summaryQuery = `
	SELECT c.id, u.id, u.username, u.rating,
		CASE
			WHEN m.archived_at IS NOT NULL THEN 'archived'
			WHEN m.replied                 THEN 'inbox'
			ELSE 'unread'
		END,
		(lm.id IS NOT NULL AND lm.sender_id <> $1
			AND lm.created_at > coalesce(m.last_read_at, '-infinity'::timestamptz)),
		lm.id, lm.sender_id, lm.body, lm.created_at
	FROM conversation_members m
	JOIN conversations c ON c.id = m.conversation_id
	JOIN users u ON u.id = CASE WHEN c.user_lo = $1 THEN c.user_hi ELSE c.user_lo END
	LEFT JOIN LATERAL (
		SELECT d.id, d.sender_id, d.body, d.created_at
		FROM direct_messages d
		WHERE d.conversation_id = c.id
		ORDER BY d.id DESC
		LIMIT 1
	) lm ON true
	WHERE m.user_id = $1
	  AND NOT EXISTS (SELECT 1 FROM blocks b WHERE b.blocker_id = $1 AND b.blocked_id = u.id)
`

type rowScanner interface {
	Scan(dest ...any) error
}

func scanSummary(row rowScanner) (*domain.ConversationSummary, error) {
	var (
		s        domain.ConversationSummary
		lastID   sql.NullInt64
		lastFrom sql.NullString
		lastBody sql.NullString
		lastAt   sql.NullTime
	)
	err := row.Scan(&s.ID, &s.Other.UserID, &s.Other.Username, &s.Other.Rating,
		&s.Folder, &s.Unread, &lastID, &lastFrom, &lastBody, &lastAt)
	if err != nil {
		return nil, err
	}
	if lastID.Valid {
		s.LastMessage = &domain.DirectMessage{
			ID: lastID.Int64, ConversationID: s.ID, SenderID: lastFrom.String,
			Body: lastBody.String, SentAt: lastAt.Time,
		}
	}
	return &s, nil
}

func (r *PostgresChatRepository) ListConversations(ctx context.Context, userID string, limit int) ([]domain.ConversationSummary, error) {
	rows, err := r.db.QueryContext(ctx, summaryQuery+` ORDER BY c.last_message_at DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("listing conversations: %w", err)
	}
	defer rows.Close()

	summaries := []domain.ConversationSummary{}
	for rows.Next() {
		s, err := scanSummary(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning conversation: %w", err)
		}
		summaries = append(summaries, *s)
	}
	return summaries, rows.Err()
}

func (r *PostgresChatRepository) GetConversation(ctx context.Context, convID, userID string) (*domain.ConversationSummary, error) {
	s, err := scanSummary(r.db.QueryRowContext(ctx, summaryQuery+` AND c.id = $2`, userID, convID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || isBadReference(err) {
			return nil, domain.ErrConversationNotFound
		}
		return nil, fmt.Errorf("finding conversation: %w", err)
	}
	return s, nil
}

func (r *PostgresChatRepository) ListDirectMessages(ctx context.Context, convID string, beforeID int64, limit int) ([]domain.DirectMessage, error) {
	const query = `
		SELECT id, conversation_id, sender_id, body, created_at
		FROM direct_messages
		WHERE conversation_id = $1 AND ($2::bigint = 0 OR id < $2::bigint)
		ORDER BY id DESC
		LIMIT $3
	`

	rows, err := r.db.QueryContext(ctx, query, convID, beforeID, limit)
	if err != nil {
		if isBadReference(err) {
			return nil, domain.ErrConversationNotFound
		}
		return nil, fmt.Errorf("listing messages: %w", err)
	}
	defer rows.Close()

	msgs := []domain.DirectMessage{}
	for rows.Next() {
		var m domain.DirectMessage
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.SentAt); err != nil {
			return nil, fmt.Errorf("scanning message: %w", err)
		}
		msgs = append(msgs, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// Fetched newest-first so LIMIT keeps the most recent page; handed back
	// oldest-first, the order a thread renders in.
	slices.Reverse(msgs)
	return msgs, nil
}

func (r *PostgresChatRepository) MarkRead(ctx context.Context, convID, userID string) error {
	const query = `
		UPDATE conversation_members SET last_read_at = now()
		WHERE conversation_id = $1 AND user_id = $2
	`
	return r.updateMember(ctx, "marking read", query, convID, userID)
}

func (r *PostgresChatRepository) SetArchived(ctx context.Context, convID, userID string, archived bool) error {
	const query = `
		UPDATE conversation_members
		SET archived_at = CASE WHEN $3 THEN now() ELSE NULL END
		WHERE conversation_id = $1 AND user_id = $2
	`
	return r.updateMember(ctx, "setting archived", query, convID, userID, archived)
}

// updateMember runs an UPDATE on one member row, answering
// ErrConversationNotFound when the user isn't a member.
func (r *PostgresChatRepository) updateMember(ctx context.Context, what, query string, args ...any) error {
	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		if isBadReference(err) {
			return domain.ErrConversationNotFound
		}
		return fmt.Errorf("%s: %w", what, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", what, err)
	}
	if n == 0 {
		return domain.ErrConversationNotFound
	}
	return nil
}

func (r *PostgresChatRepository) AppendGameMessage(ctx context.Context, gameID, senderID, body string) (*domain.GameMessage, error) {
	const query = `
		INSERT INTO game_messages (game_id, sender_id, body)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`

	msg := domain.GameMessage{GameID: gameID, SenderID: senderID, Body: body}
	if err := r.db.QueryRowContext(ctx, query, gameID, senderID, body).Scan(&msg.ID, &msg.SentAt); err != nil {
		if isBadReference(err) {
			return nil, domain.ErrGameNotFound
		}
		return nil, fmt.Errorf("inserting game message: %w", err)
	}
	return &msg, nil
}

func (r *PostgresChatRepository) ListGameMessages(ctx context.Context, gameID string, limit int) ([]domain.GameMessage, error) {
	const query = `
		SELECT id, game_id, sender_id, body, created_at
		FROM game_messages
		WHERE game_id = $1
		ORDER BY id
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, gameID, limit)
	if err != nil {
		if isBadReference(err) {
			return nil, domain.ErrGameNotFound
		}
		return nil, fmt.Errorf("listing game messages: %w", err)
	}
	defer rows.Close()

	msgs := []domain.GameMessage{}
	for rows.Next() {
		var m domain.GameMessage
		if err := rows.Scan(&m.ID, &m.GameID, &m.SenderID, &m.Body, &m.SentAt); err != nil {
			return nil, fmt.Errorf("scanning game message: %w", err)
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}
