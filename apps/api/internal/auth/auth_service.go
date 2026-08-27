package auth

import (
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
		return "", ErrInvalidCredentials
	}

	if a.hasher.Compare(u.PasswordHash, password); err != nil {
		return "", ErrInvalidCredentials
	}

	token, err := GenerateToken(u.UserID, secret, ttl);
	if err != nil {
		return "", ErrTokenGeneration
	}

	return token, nil
}