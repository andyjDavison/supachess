package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"api/internal/domain"
	"api/internal/user"
)

type stubUsers struct {
	byEmail map[string]*domain.User
	byID    map[string]*domain.User
}

func (s *stubUsers) FindUserByEmail(_ context.Context, email string) (*domain.User, error) {
	u, ok := s.byEmail[email]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return u, nil
}

func (s *stubUsers) FindUserById(_ context.Context, id string) (*domain.User, error) {
	u, ok := s.byID[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return u, nil
}

func newAuthHarness(t *testing.T) (*AuthService, *domain.User) {
	t.Helper()
	hasher := user.NewBcryptHasher(4) // low cost: these tests hash a lot
	hash, err := hasher.Hash("correct-horse")
	if err != nil {
		t.Fatalf("hashing fixture password: %v", err)
	}
	u := &domain.User{UserID: "user-1", Username: "alice", Email: "alice@example.com", PasswordHash: hash}
	users := &stubUsers{
		byEmail: map[string]*domain.User{u.Email: u},
		byID:    map[string]*domain.User{u.UserID: u},
	}
	return NewAuthService(users, hasher), u
}

func TestVerifyCredentialsSuccess(t *testing.T) {
	svc, u := newAuthHarness(t)

	token, err := svc.VerifyCredentials(context.Background(), u.Email, "correct-horse", secret, time.Hour)
	if err != nil {
		t.Fatalf("VerifyCredentials: %v", err)
	}

	subject, err := ParseToken(token, secret)
	if err != nil {
		t.Fatalf("returned token did not parse: %v", err)
	}
	if subject != u.UserID {
		t.Errorf("token subject = %q, want %q", subject, u.UserID)
	}
}

// Both failure modes must be indistinguishable to the caller, or the error
// tells an attacker whether an email is registered.
func TestVerifyCredentialsFailuresAreIndistinguishable(t *testing.T) {
	svc, u := newAuthHarness(t)
	ctx := context.Background()

	_, unknownErr := svc.VerifyCredentials(ctx, "nobody@example.com", "correct-horse", secret, time.Hour)
	_, wrongPassErr := svc.VerifyCredentials(ctx, u.Email, "wrong", secret, time.Hour)

	if !errors.Is(unknownErr, domain.ErrInvalidCredentials) {
		t.Errorf("unknown email gave %v, want ErrInvalidCredentials", unknownErr)
	}
	if !errors.Is(wrongPassErr, domain.ErrInvalidCredentials) {
		t.Errorf("wrong password gave %v, want ErrInvalidCredentials", wrongPassErr)
	}
	if unknownErr.Error() != wrongPassErr.Error() {
		t.Errorf("the two failures differ (%q vs %q) - that leaks whether an account exists",
			unknownErr, wrongPassErr)
	}
}

func TestFindByIDReturnsProfile(t *testing.T) {
	svc, u := newAuthHarness(t)

	profile, err := svc.FindByID(context.Background(), u.UserID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if profile.ID != u.UserID || profile.Email != u.Email || profile.Username != u.Username {
		t.Errorf("profile = %+v, want id/email/username from %+v", profile, u)
	}
}

func TestFindByIDUnknownUser(t *testing.T) {
	svc, _ := newAuthHarness(t)

	if _, err := svc.FindByID(context.Background(), "nope"); !errors.Is(err, domain.ErrUserNotFound) {
		t.Errorf("got %v, want ErrUserNotFound", err)
	}
}

// The profile is what reaches the browser, so it must not carry the hash.
func TestMapUserToProfileDropsPasswordHash(t *testing.T) {
	p := MapUserToProfile(&domain.User{
		UserID: "u1", Email: "e@x.com", Username: "n", PasswordHash: "$2a$secret",
	})

	if p.ID != "u1" || p.Email != "e@x.com" || p.Username != "n" {
		t.Errorf("profile = %+v", p)
	}
}
