package user

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"api/internal/domain"
)

// memUserRepo is an in-memory UserRepository for service tests.
type memUserRepo struct {
	mu        sync.Mutex
	byID      map[string]*domain.User
	nextID    int
	createErr error
	lookupErr error
}

func newMemUserRepo() *memUserRepo {
	return &memUserRepo{byID: map[string]*domain.User{}}
}

func (r *memUserRepo) Create(_ context.Context, input *domain.User) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.createErr != nil {
		return nil, r.createErr
	}
	r.nextID++
	stored := *input
	stored.UserID = "user-" + string(rune('0'+r.nextID))
	r.byID[stored.UserID] = &stored
	return &stored, nil
}

func (r *memUserRepo) find(match func(*domain.User) bool) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.lookupErr != nil {
		return nil, r.lookupErr
	}
	for _, u := range r.byID {
		if match(u) {
			return u, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func (r *memUserRepo) FindUserById(_ context.Context, id string) (*domain.User, error) {
	return r.find(func(u *domain.User) bool { return u.UserID == id })
}

func (r *memUserRepo) FindUserByEmail(_ context.Context, email string) (*domain.User, error) {
	return r.find(func(u *domain.User) bool { return u.Email == email })
}

func (r *memUserRepo) FindUserByUsername(_ context.Context, username string) (*domain.User, error) {
	return r.find(func(u *domain.User) bool { return u.Username == username })
}

func validInput() UserRegistrationInput {
	return UserRegistrationInput{Username: "alice", Email: "alice@example.com", Password: "password123", Rating: 800}
}

func TestValidateRegisterInput(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*UserRegistrationInput)
		wantErr string
	}{
		{"valid", func(*UserRegistrationInput) {}, ""},
		{"username too short", func(i *UserRegistrationInput) { i.Username = "ab" }, "username"},
		{"username at the boundary", func(i *UserRegistrationInput) { i.Username = "abc" }, ""},
		{"password too short", func(i *UserRegistrationInput) { i.Password = "1234567" }, "password"},
		{"password at the boundary", func(i *UserRegistrationInput) { i.Password = "12345678" }, ""},
		{"email without @", func(i *UserRegistrationInput) { i.Email = "no-at-sign" }, "email"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in := validInput()
			c.mutate(&in)

			err := validateRegisterInput(in)

			if c.wantErr == "" {
				if err != nil {
					t.Fatalf("want no error, got %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), c.wantErr) {
				t.Fatalf("error = %v, want one mentioning %q", err, c.wantErr)
			}
		})
	}
}

func TestCreateUserStoresAHashNotThePassword(t *testing.T) {
	repo := newMemUserRepo()
	hasher := NewBcryptHasher(4)
	svc := NewUserService(repo, hasher)

	created, err := svc.CreateUser(context.Background(), validInput())
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	if created.PasswordHash == "password123" {
		t.Fatal("the plaintext password was stored")
	}
	if err := hasher.Compare(created.PasswordHash, "password123"); err != nil {
		t.Errorf("stored hash does not verify the original password: %v", err)
	}
	if created.Rating != 800 {
		t.Errorf("rating = %d, want the 800 passed in", created.Rating)
	}
	if created.Username != "alice" || created.Email != "alice@example.com" {
		t.Errorf("created = %+v", created)
	}
}

func TestCreateUserRejectsTakenUsername(t *testing.T) {
	repo := newMemUserRepo()
	svc := NewUserService(repo, NewBcryptHasher(4))
	ctx := context.Background()
	if _, err := svc.CreateUser(ctx, validInput()); err != nil {
		t.Fatalf("seeding first user: %v", err)
	}

	in := validInput()
	in.Email = "different@example.com" // same username, new email
	_, err := svc.CreateUser(ctx, in)

	if !errors.Is(err, domain.ErrUsernameTaken) {
		t.Errorf("got %v, want ErrUsernameTaken", err)
	}
}

func TestCreateUserRejectsTakenEmail(t *testing.T) {
	repo := newMemUserRepo()
	svc := NewUserService(repo, NewBcryptHasher(4))
	ctx := context.Background()
	if _, err := svc.CreateUser(ctx, validInput()); err != nil {
		t.Fatalf("seeding first user: %v", err)
	}

	in := validInput()
	in.Username = "bob" // same email, new username
	_, err := svc.CreateUser(ctx, in)

	if !errors.Is(err, domain.ErrEmailTaken) {
		t.Errorf("got %v, want ErrEmailTaken", err)
	}
}

func TestCreateUserRejectsInvalidInputBeforeTouchingTheRepo(t *testing.T) {
	repo := newMemUserRepo()
	repo.lookupErr = errors.New("repo should not have been consulted")
	svc := NewUserService(repo, NewBcryptHasher(4))

	in := validInput()
	in.Password = "short"

	if _, err := svc.CreateUser(context.Background(), in); err == nil {
		t.Error("expected a validation error")
	} else if strings.Contains(err.Error(), "should not have been consulted") {
		t.Error("validation ran after the repository lookup, not before")
	}
}

// A lookup failure is not "name is free" - it must not be swallowed into a
// successful registration.
func TestCreateUserPropagatesLookupFailure(t *testing.T) {
	repo := newMemUserRepo()
	repo.lookupErr = errors.New("database is down")
	svc := NewUserService(repo, NewBcryptHasher(4))

	_, err := svc.CreateUser(context.Background(), validInput())

	if err == nil {
		t.Fatal("expected the lookup failure to surface")
	}
	if errors.Is(err, domain.ErrUsernameTaken) || errors.Is(err, domain.ErrEmailTaken) {
		t.Errorf("a database error was reported as a taken name: %v", err)
	}
}

func TestFindUserByIDNotFound(t *testing.T) {
	svc := NewUserService(newMemUserRepo(), NewBcryptHasher(4))

	if _, err := svc.FindUserById(context.Background(), "nope"); !errors.Is(err, domain.ErrUserNotFound) {
		t.Errorf("got %v, want it to wrap ErrUserNotFound", err)
	}
}
