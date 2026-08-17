package domain

import (
	"errors"
	"time"
)

type User struct {
	id int64
	username string
	email string
	passwordHash string
	rating int32
	createdAt time.Time
}

func (u *User) ID() int64 {
	return u.id
}

// constructor factory
func NewUser(id int64, email string, passwordHash string, rating int32) (*User, error) {
	if id == 0 {
		return nil, errors.New("User ID cannot be empty")
	}
	if email == "" {
		return nil, errors.New("User email cannot be empty")
	}

	return &User {
		id: id,
		email: email,
		passwordHash: passwordHash,
		createdAt: time.Now(),
		rating: rating,
	}, nil
}