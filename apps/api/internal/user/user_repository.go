package user

import (
	"api/internal/domain"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

const (
	usernameUniqueConstraint = "users_username_key"
	emailUniqueConstraint    = "users_email_key"
)

type User struct {
	ID string
	Username string
	Email string
	PasswordHash string
	Rating int32
	CreatedAt time.Time
}

// UserRepository is the persistence seam UserService depends on, so the
// service can run against an in-memory implementation in tests.
//
// This replaces an earlier IUserRepository that no type satisfied: it
// declared FindById/FindByEmail/FindByUsername against a repository whose
// methods are FindUserById/FindUserByEmail/FindUserByUsername, and typed
// Create against the package-local User rather than domain.User. It was
// referenced nowhere, so nothing caught the drift.
type UserRepository interface {
	Create(ctx context.Context, input *domain.User) (*domain.User, error)
	FindUserById(ctx context.Context, id string) (*domain.User, error)
	FindUserByEmail(ctx context.Context, email string) (*domain.User, error)
	FindUserByUsername(ctx context.Context, username string) (*domain.User, error)
}

type PostgresUserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

func (repo *PostgresUserRepository) Create(ctx context.Context, input *domain.User) (*domain.User, error) {
	user := MapUserToDB(input)
	// The signup rating seeds all three speeds, in the same statement so
	// a user never exists without them.
	const query = `
		WITH created AS (
			INSERT INTO users (username, email, password_hash, rating, created_at)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id, username, email, password_hash, rating, created_at
		), seeded AS (
			INSERT INTO user_ratings (user_id, category, rating)
			SELECT created.id, c.category, created.rating
			FROM created CROSS JOIN (VALUES ('bullet'), ('blitz'), ('rapid')) AS c(category)
		)
		SELECT id, username, email, password_hash, rating, created_at FROM created;
	`
	var createdUser domain.User
	err := repo.db.QueryRowContext(ctx, query, user.Username, user.Email, user.PasswordHash, user.Rating, time.Now().UTC()).
		Scan(&createdUser.UserID, &createdUser.Username, &createdUser.Email, &createdUser.PasswordHash, &createdUser.Rating, &createdUser.CreatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			switch pgErr.ConstraintName {
			case usernameUniqueConstraint:
				return nil, domain.ErrUsernameTaken
			case emailUniqueConstraint:
				return nil, domain.ErrEmailTaken
			}
		}
		return nil, fmt.Errorf("inserting user: %w", err)
	}

	return &createdUser, nil
}

func (repo *PostgresUserRepository) FindUserById(ctx context.Context, id string) (*domain.User, error) {
	const query = `
		SELECT * FROM users WHERE id = $1
	`

	var foundUser domain.User
	err := repo.db.QueryRowContext(ctx, query, id).Scan(&foundUser.UserID, &foundUser.Username, &foundUser.Email, &foundUser.PasswordHash, &foundUser.Rating, &foundUser.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("finding user by id: %w", err)
	}

	return &foundUser, nil
}

func (repo *PostgresUserRepository) FindUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	const query = `
		SELECT * FROM users WHERE email = $1
	`

	var foundUser domain.User
	err := repo.db.QueryRowContext(ctx, query, email).Scan(&foundUser.UserID, &foundUser.Username, &foundUser.Email, &foundUser.PasswordHash, &foundUser.Rating, &foundUser.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("finding user by email: %w", err)
	}

	return &foundUser, nil
}

func (repo *PostgresUserRepository) FindUserByUsername(ctx context.Context, username string) (*domain.User, error) {
	const query = `
		SELECT * FROM users WHERE username = $1
	`

	var foundUser domain.User
	err := repo.db.QueryRowContext(ctx, query, username).Scan(&foundUser.UserID, &foundUser.Username, &foundUser.Email, &foundUser.PasswordHash, &foundUser.Rating, &foundUser.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("finding user by username: %w", err)
	}

	return &foundUser, nil
}

func MapUserToDB(input *domain.User) User {
	return User{
		Username: input.Username,
		Email: input.Email,
		PasswordHash: input.PasswordHash,
		Rating: input.Rating,
	}
}