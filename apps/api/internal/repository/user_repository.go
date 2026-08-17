package repository

import (
	"context"
	"database/sql"
)

type User struct {
	ID int64
	Username string
	Email string
	PasswordHash string
	Rating int32
}

type IUserRepository interface {
	Create(ctx context.Context, user *User) error
	// FindById(ctx context.Context, id string) error
	// FindByUsername(ctx context.Context, username string) error
	// FindByEmail(ctx context.Context, email string) error
}

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (repo *UserRepository) Create(ctx context.Context, user *User) error {
	query := `INSERT INTO users (username, email, password_hash, rating) VALUES ($1, $2, $3, $4) RETURNING id;`
	
	return repo.db.QueryRowContext(ctx, query, user.Username, user.Email, user.PasswordHash, user.Rating).Scan(&user.ID)
}