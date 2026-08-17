package user

import (
	"api/internal/domain"
	"context"
	"errors"
	"fmt"
	"strings"
)

type UserRegistrationInput struct {
	Username string
	Email string
	Password string
	Rating int32
}

type UserService struct {
	repo *UserRepository
	hasher PasswordHasher
}

func NewUserService(repo *UserRepository, hasher PasswordHasher) *UserService {
	return &UserService{repo: repo, hasher: hasher}
}

func (s *UserService) CreateUser(ctx context.Context, input UserRegistrationInput) (*domain.User, error) {
	if err := validateRegisterInput(input); err != nil {
		return nil, err
	}
	// if _, err := s.repo.FindByUsername(ctx, input.Username); err == nil {
	// 	return nil, domain.ErrUsernameTaken
	// } else if !errors.Is(err, domain.ErrUserNotFound) {
	// 	return nil, fmt.Errorf("checking username: %w", err)
	// }

	// if _, err := s.repo.FindByEmail(ctx, in.Email); err == nil {
	// 	return nil, domain.ErrEmailTaken
	// } else if !errors.Is(err, domain.ErrUserNotFound) {
	// 	return nil, fmt.Errorf("checking email: %w", err)
	// }

	hash, err := s.hasher.Hash(input.Password)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	user := &domain.User{
		Username: input.Username,
		Email: input.Email,
		PasswordHash: hash,
		Rating: input.Rating,
	}
	
	created, err := s.repo.Create(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}
	return created, nil
}

func validateRegisterInput(user UserRegistrationInput) error {
	switch {
	case len(user.Username) < 3:
		return errors.New("username must be at least 3 characters")
	case len(user.Password) < 8:
		return errors.New("password must be at least 8 characters")
	case !strings.ContainsRune(user.Email, '@'):
		return errors.New("email is invalid")
	}
	return nil
}