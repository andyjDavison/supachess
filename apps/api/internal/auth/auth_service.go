package auth

import (
	"api/internal/domain"
	"api/internal/user"
	"context"
	"time"
)

// userLookup is the slice of the user repository this service actually
// uses - two reads, no writes. Narrow by intent, matching how
// matchmaking declares gameService/userLookup rather than depending on
// a whole concrete service.
type userLookup interface {
	FindUserByEmail(ctx context.Context, email string) (*domain.User, error)
	FindUserById(ctx context.Context, id string) (*domain.User, error)
}

type AuthService struct {
	repo   userLookup
	hasher user.PasswordHasher
}

func NewAuthService(repo userLookup, hasher user.PasswordHasher) *AuthService {
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