package game

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"api/internal/domain"
	"api/internal/rating"

	"github.com/jackc/pgx/v5/pgconn"
)

// invalidTextRepresentation is the Postgres code for a value that cannot
// be coerced to a column's type.
const invalidTextRepresentation = "22P02"

// isNoSuchGame reports whether err means "there is no game with that id".
//
// Two distinct database outcomes mean the same thing to a caller. The
// obvious one is no rows. The other is a malformed id: games.id is a uuid
// column, so a path value like "not-a-uuid" fails type conversion with
// 22P02 rather than matching zero rows. Treating only ErrNoRows as
// not-found meant such a request answered 500 instead of 404 - the caller
// asked for a game that cannot exist, which is a 404 either way.
func isNoSuchGame(err error) bool {
	if errors.Is(err, sql.ErrNoRows) {
		return true
	}
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == invalidTextRepresentation
}

// GameRepository is the persistence seam GameService depends on. Declared
// here as an interface so the service can be exercised against an
// in-memory implementation in tests, with PostgresGameRepository below as
// the production one.
type GameRepository interface {
	Create(ctx context.Context, g *domain.Game) (*domain.Game, error)
	FindByID(ctx context.Context, id string) (*domain.Game, error)
	FindActiveGameByPlayerID(ctx context.Context, playerID string) (*domain.Game, error)
	Update(ctx context.Context, g *domain.Game) error
	// FinishGame records a game's end and applies its rating deltas
	// (g.Ratings.*.Delta) to both players, atomically. It only finishes a
	// game that's still active, answering ErrGameAlreadyFinished otherwise,
	// so two racing endings can't both move the ratings.
	FinishGame(ctx context.Context, g *domain.Game) error
	// RatingFor is a player's current rating in one category.
	RatingFor(ctx context.Context, userID string, category domain.RatingCategory) (domain.PlayerRating, error)
	AppendMove(ctx context.Context, gameID string, m *domain.Move) error
	FindMovesByGameID(ctx context.Context, gameID string) ([]domain.Move, error)
	// DeleteWaitingGame removes a never-started invite game. It returns
	// ErrGameNotFound unless the row exists and is still waiting, so it
	// can never delete a game that's been played.
	DeleteWaitingGame(ctx context.Context, id string) error
	DeleteAllWaitingGames(ctx context.Context) (int64, error)
}

type PostgresGameRepository struct {
	db *sql.DB
}

func NewPostgresGameRepository(db *sql.DB) *PostgresGameRepository {
	return &PostgresGameRepository{db: db}
}

func (r *PostgresGameRepository) Create(ctx context.Context, g *domain.Game) (*domain.Game, error) {
	query := `
		INSERT INTO games (
			white_id, black_id, fen, game_status, result, result_reason,
			white_time, black_time, initial, increment,
			white_rating, black_rating, white_k, black_k
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING ` + gameColumns

	var whiteRating, blackRating, whiteK, blackK sql.NullInt32
	if g.Ratings != nil {
		whiteRating = sql.NullInt32{Int32: g.Ratings.White.Before, Valid: true}
		blackRating = sql.NullInt32{Int32: g.Ratings.Black.Before, Valid: true}
		whiteK = sql.NullInt32{Int32: g.Ratings.White.K, Valid: true}
		blackK = sql.NullInt32{Int32: g.Ratings.Black.K, Valid: true}
	}

	row := r.db.QueryRowContext(ctx, query,
		g.WhiteID, g.BlackID, g.FEN, g.Status, g.Result, g.ResultReason,
		int64(g.WhiteTimeRemaining), int64(g.BlackTimeRemaining),
		int64(g.TimeControl.Initial), int64(g.TimeControl.Increment),
		whiteRating, blackRating, whiteK, blackK,
	)

	created, err := scanGame(row)
	if err != nil {
		return nil, fmt.Errorf("inserting game: %w", err)
	}
	return created, nil
}

func (r *PostgresGameRepository) FindByID(ctx context.Context, id string) (*domain.Game, error) {
	query := `SELECT ` + gameColumns + ` FROM games WHERE id = $1`

	g, err := scanGame(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if isNoSuchGame(err) {
			return nil, domain.ErrGameNotFound
		}
		return nil, fmt.Errorf("finding game by id: %w", err)
	}
	return g, nil
}

func (r *PostgresGameRepository) FindActiveGameByPlayerID(ctx context.Context, playerID string) (*domain.Game, error) {
	query := `SELECT ` + gameColumns + `
		FROM games
		WHERE game_status = 'active' AND (white_id = $1 OR black_id = $1)
		LIMIT 1`

	g, err := scanGame(r.db.QueryRowContext(ctx, query, playerID))
	if err != nil {
		if isNoSuchGame(err) {
			return nil, domain.ErrGameNotFound
		}
		return nil, fmt.Errorf("finding active game for player: %w", err)
	}
	return g, nil
}

func (r *PostgresGameRepository) Update(ctx context.Context, g *domain.Game) error {
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

func (r *PostgresGameRepository) AppendMove(ctx context.Context, gameID string, m *domain.Move) error {
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

func (r *PostgresGameRepository) FindMovesByGameID(ctx context.Context, gameID string) ([]domain.Move, error) {
	const query = `
		SELECT game_id, ply, color, from_square, to_square, promotion, san, fen_after, played_at
		FROM moves
		WHERE game_id = $1
		ORDER BY ply ASC
	`

	rows, err := r.db.QueryContext(ctx, query, gameID)
	if err != nil {
		if isNoSuchGame(err) {
			return nil, domain.ErrGameNotFound
		}
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

// gameColumns is the column list scanGame reads, shared by every query
// that returns a game so it's only written out once.
const gameColumns = `
	id, white_id, black_id, fen, game_status, result, result_reason,
	white_time, black_time, initial, increment,
	last_move_at, created_at, finished_at,
	white_rating, black_rating, white_k, black_k,
	white_rating_delta, black_rating_delta`

// scanGame reads one row of gameColumns.
func scanGame(row *sql.Row) (*domain.Game, error) {
	var g domain.Game
	var whiteRemainingNS, blackRemainingNS, initialNS, incrementNS int64
	var finishedAt sql.NullTime
	var whiteRating, blackRating, whiteK, blackK, whiteDelta, blackDelta sql.NullInt32

	err := row.Scan(
		&g.ID, &g.WhiteID, &g.BlackID, &g.FEN, &g.Status, &g.Result, &g.ResultReason,
		&whiteRemainingNS, &blackRemainingNS, &initialNS, &incrementNS,
		&g.LastMoveAt, &g.CreatedAt, &finishedAt,
		&whiteRating, &blackRating, &whiteK, &blackK,
		&whiteDelta, &blackDelta,
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
	if whiteRating.Valid && blackRating.Valid {
		g.Ratings = &domain.GameRatings{
			Category: rating.CategoryFor(g.TimeControl),
			White:    domain.SideRating{Before: whiteRating.Int32, K: whiteK.Int32, Delta: nullableInt32(whiteDelta)},
			Black:    domain.SideRating{Before: blackRating.Int32, K: blackK.Int32, Delta: nullableInt32(blackDelta)},
		}
	}

	return &g, nil
}

func nullableInt32(n sql.NullInt32) *int32 {
	if !n.Valid {
		return nil
	}
	return &n.Int32
}

func nullableTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}
func (r *PostgresGameRepository) DeleteWaitingGame(ctx context.Context, id string) error {
	const query = `DELETE FROM games WHERE id = $1 AND game_status = 'waiting'`

	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		if isNoSuchGame(err) {
			return domain.ErrGameNotFound
		}
		return fmt.Errorf("deleting waiting game: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking delete result: %w", err)
	}
	if n == 0 {
		return domain.ErrGameNotFound
	}
	return nil
}

func (r *PostgresGameRepository) DeleteAllWaitingGames(ctx context.Context) (int64, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM games WHERE game_status = 'waiting'`)
	if err != nil {
		return 0, fmt.Errorf("deleting waiting games: %w", err)
	}
	return res.RowsAffected()
}

func (r *PostgresGameRepository) FinishGame(ctx context.Context, g *domain.Game) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning finish: %w", err)
	}
	defer tx.Rollback()

	var whiteDelta, blackDelta sql.NullInt32
	if g.Ratings != nil && g.Ratings.White.Delta != nil && g.Ratings.Black.Delta != nil {
		whiteDelta = sql.NullInt32{Int32: *g.Ratings.White.Delta, Valid: true}
		blackDelta = sql.NullInt32{Int32: *g.Ratings.Black.Delta, Valid: true}
	}

	res, err := tx.ExecContext(ctx, `
		UPDATE games
		SET fen = $2, game_status = $3, result = $4, result_reason = $5,
		    white_time = $6, black_time = $7,
		    last_move_at = $8, finished_at = $9,
		    white_rating_delta = $10, black_rating_delta = $11
		WHERE id = $1 AND game_status = 'active'`,
		g.ID, g.FEN, g.Status, g.Result, g.ResultReason,
		int64(g.WhiteTimeRemaining), int64(g.BlackTimeRemaining),
		g.LastMoveAt, nullableTime(g.FinishedAt),
		whiteDelta, blackDelta,
	)
	if err != nil {
		if isNoSuchGame(err) {
			return domain.ErrGameNotFound
		}
		return fmt.Errorf("finishing game: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking finish result: %w", err)
	}
	if n == 0 {
		return domain.ErrGameAlreadyFinished
	}

	if whiteDelta.Valid {
		category := g.Ratings.Category
		for _, side := range []struct {
			userID string
			delta  int32
		}{{g.WhiteID, whiteDelta.Int32}, {g.BlackID, blackDelta.Int32}} {
			if _, err := tx.ExecContext(ctx, `
				UPDATE user_ratings
				SET rating = rating + $3, games_played = games_played + 1
				WHERE user_id = $1 AND category = $2`,
				side.userID, category, side.delta,
			); err != nil {
				return fmt.Errorf("applying rating: %w", err)
			}
		}
		// users.rating is the display rating everywhere a single one is
		// shown, and that's Blitz.
		if category == domain.CategoryBlitz {
			if _, err := tx.ExecContext(ctx, `
				UPDATE users u SET rating = ur.rating
				FROM user_ratings ur
				WHERE ur.user_id = u.id AND ur.category = 'blitz' AND u.id IN ($1, $2)`,
				g.WhiteID, g.BlackID,
			); err != nil {
				return fmt.Errorf("syncing display rating: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing finish: %w", err)
	}
	return nil
}

func (r *PostgresGameRepository) RatingFor(ctx context.Context, userID string, category domain.RatingCategory) (domain.PlayerRating, error) {
	var pr domain.PlayerRating
	err := r.db.QueryRowContext(ctx,
		`SELECT rating, games_played FROM user_ratings WHERE user_id = $1 AND category = $2`,
		userID, category,
	).Scan(&pr.Rating, &pr.GamesPlayed)
	if err != nil {
		if isNoSuchGame(err) { // no row or a malformed id: no such player either way
			return pr, domain.ErrUserNotFound
		}
		return pr, fmt.Errorf("finding rating: %w", err)
	}
	return pr, nil
}
