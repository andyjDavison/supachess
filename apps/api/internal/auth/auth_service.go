package auth

import (
	"api/internal/domain"
	"api/internal/user"
	"context"
	"time"
)

type AuthService struct {
	repo *user.UserRepository
	hasher user.PasswordHasher
}

func NewAuthService(repo *user.UserRepository, hasher user.PasswordHasher) *AuthService {
	return &AuthService{repo: repo, hasher: hasher}
}

func (a *AuthService) VerifyCredentials(ctx context.Context, email string, password string, secret []byte, ttl time.Duration) (string, error) {
	u, err := a.repo.FindUserByEmail(ctx, email);
	if err != nil {
		return "", domain.ErrInvalidCredentials
	}

	if a.hasher.Compare(u.PasswordHash, password) != nil {
		return "", domain.ErrInvalidCredentials
	}

	token, err := GenerateToken(u.UserID, secret, ttl);
	if err != nil {
		return "", domain.ErrTokenGeneration
	}

	return token, nil
}

func (a *AuthService) FindByID(ctx context.Context, userID string) (*domain.Profile, error) {
	u, err := a.repo.FindUserById(ctx, userID);
	if err != nil {
		return nil, domain.ErrUserNotFound
	}

	return MapUserToProfile(u), nil
}

func MapUserToProfile(user *domain.User) *domain.Profile {
	return &domain.Profile{ID: user.UserID, Email: user.Email, Username: user.Username}
}