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

type IUserRepository interface {
	Create(ctx context.Context, user *User) (*domain.User, error)
	FindById(ctx context.Context, id string) (*domain.User, error)
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindByUsername(ctx context.Context, username string) (*domain.User, error)
}

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (repo *UserRepository) Create(ctx context.Context, input *domain.User) (*domain.User, error) {
	user := MapUserToDB(input)
	const query = `
		INSERT INTO users (username, email, password_hash, rating, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, username, email, password_hash, rating, created_at;
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

func (repo *UserRepository) FindUserById(ctx context.Context, id string) (*domain.User, error) {
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

func (repo *UserRepository) FindUserByEmail(ctx context.Context, email string) (*domain.User, error) {
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

func (repo *UserRepository) FindUserByUsername(ctx context.Context, username string) (*domain.User, error) {
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