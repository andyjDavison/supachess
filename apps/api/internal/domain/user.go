package domain

import (
	"time"
)

type User struct {
	UserID string
	Username string
	Email string
	PasswordHash string
	Rating int32
	CreatedAt time.Time
}

func (u *User) ID() string {
	return u.UserID
}

// constructor factory
func NewUser(id string, email string, passwordHash string, rating int32) (*User, error) {
	return &User {
		UserID: id,
		Email: email,
		PasswordHash: passwordHash,
		CreatedAt: time.Now(),
		Rating: rating,
	}, nil
}

type Profile struct {
	ID string
	Email string
	Username string
}