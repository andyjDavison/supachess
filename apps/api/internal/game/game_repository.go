package game

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"api/internal/domain"
)

type GameRepository struct {
	db *sql.DB
}

func NewPostgresGameRepository(db *sql.DB) *GameRepository {
	return &GameRepository{db: db}
}

func (r *GameRepository) Create(ctx context.Context, g *domain.Game) (*domain.Game, error) {
	const query = `
		INSERT INTO games (
			white_id, black_id, fen, game_status, result, result_reason,
			white_time, black_time, initial, increment
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING
			id, white_id, black_id, fen, game_status, result, result_reason,
			white_time, black_time, initial, increment,
			last_move_at, created_at, finished_at
	`

	row := r.db.QueryRowContext(ctx, query,
		g.WhiteID, g.BlackID, g.FEN, g.Status, g.Result, g.ResultReason,
		int64(g.WhiteTimeRemaining), int64(g.BlackTimeRemaining),
		int64(g.TimeControl.Initial), int64(g.TimeControl.Increment),
	)

	created, err := scanGame(row)
	if err != nil {
		return nil, fmt.Errorf("inserting game: %w", err)
	}
	return created, nil
}

func (r *GameRepository) FindByID(ctx context.Context, id string) (*domain.Game, error) {
	const query = `
		SELECT
			id, white_id, black_id, fen, game_status, result, result_reason,
			white_time, black_time, initial, increment,
			last_move_at, created_at, finished_at
		FROM games
		WHERE id = $1
	`

	g, err := scanGame(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrGameNotFound
		}
		return nil, fmt.Errorf("finding game by id: %w", err)
	}
	return g, nil
}

func (r *GameRepository) FindActiveGameByPlayerID(ctx context.Context, playerID string) (*domain.Game, error) {
	const query = `
		SELECT
			id, white_id, black_id, fen, game_status, result, result_reason,
			white_time, black_time, initial, increment,
			last_move_at, created_at, finished_at
		FROM games
		WHERE game_status = 'active' AND (white_id = $1 OR black_id = $1)
		LIMIT 1
	`

	g, err := scanGame(r.db.QueryRowContext(ctx, query, playerID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrGameNotFound
		}
		return nil, fmt.Errorf("finding active game for player: %w", err)
	}
	return g, nil
}

func (r *GameRepository) Update(ctx context.Context, g *domain.Game) error {
	const query = `
		UPDATE games
		SET fen = $2, game_status = $3, result = $4, result_reason = $5,
		    white_time = $6, black_time = $7,
		    last_move_at = $8, finished_at = $9
		WHERE id = $1
	`

	res, err := r.db.ExecContext(ctx, query,
		g.ID, g.FEN, g.Status, g.Result, g.ResultReason,
		int64(g.WhiteTimeRemaining), int64(g.BlackTimeRemaining),
		g.LastMoveAt, nullableTime(g.FinishedAt),
	)
	if err != nil {
		return fmt.Errorf("updating game: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking update result: %w", err)
	}
	if n == 0 {
		return domain.ErrGameNotFound
	}
	return nil
}

func (r *GameRepository) AppendMove(ctx context.Context, gameID string, m *domain.Move) error {
	const query = `
		INSERT INTO moves (
			game_id, ply, color, from_square, to_square, promotion, san, fen_after, played_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.ExecContext(ctx, query,
		gameID, m.Ply, m.Color, m.From, m.To, m.Promotion, m.SAN, m.FENAfter, m.PlayedAt,
	)
	if err != nil {
		return fmt.Errorf("appending move: %w", err)
	}
	return nil
}

func (r *GameRepository) FindMovesByGameID(ctx context.Context, gameID string) ([]domain.Move, error) {
	const query = `
		SELECT game_id, ply, color, from_square, to_square, promotion, san, fen_after, played_at
		FROM moves
		WHERE game_id = $1
		ORDER BY ply ASC
	`

	rows, err := r.db.QueryContext(ctx, query, gameID)
	if err != nil {
		return nil, fmt.Errorf("finding moves: %w", err)
	}
	defer rows.Close()

	var moves []domain.Move
	for rows.Next() {
		var m domain.Move
		if err := rows.Scan(
			&m.GameID, &m.Ply, &m.Color, &m.From, &m.To, &m.Promotion, &m.SAN, &m.FENAfter, &m.PlayedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning move: %w", err)
		}
		moves = append(moves, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating moves: %w", err)
	}

	return moves, nil
}

// scanGame centralizes the column list shared by Create/FindByID/
// FindActiveGameByPlayerID so it's only written out once. row can be the
// result of either QueryRowContext call — both satisfy the same Scan
// interface.
func scanGame(row *sql.Row) (*domain.Game, error) {
	var g domain.Game
	var whiteRemainingNS, blackRemainingNS, initialNS, incrementNS int64
	var finishedAt sql.NullTime

	err := row.Scan(
		&g.ID, &g.WhiteID, &g.BlackID, &g.FEN, &g.Status, &g.Result, &g.ResultReason,
		&whiteRemainingNS, &blackRemainingNS, &initialNS, &incrementNS,
		&g.LastMoveAt, &g.CreatedAt, &finishedAt,
	)
	if err != nil {
		return nil, err
	}

	g.WhiteTimeRemaining = time.Duration(whiteRemainingNS)
	g.BlackTimeRemaining = time.Duration(blackRemainingNS)
	g.TimeControl = domain.TimeControl{
		Initial:   time.Duration(initialNS),
		Increment: time.Duration(incrementNS),
	}
	if finishedAt.Valid {
		g.FinishedAt = &finishedAt.Time
	}

	return &g, nil
}

func nullableTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}