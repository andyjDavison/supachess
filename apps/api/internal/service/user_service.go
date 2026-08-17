package service

import (
	"api/internal/repository"
	"context"
	"errors"
)

type UserService struct {
	repo *repository.UserRepository
}

func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) CreateUser(ctx context.Context, user *repository.User) error {
	if user.Username == "" {
		return errors.New("username required")
	}
	if user.Email == "" {
		return errors.New("email is required")
	}
	if user.PasswordHash == "" {
		return errors.New("password is required")
	}
	if user.Rating == 0 || user.Rating < 0 {
		return errors.New("rating is required")
	}
	
	return s.repo.Create(ctx, user)
}

