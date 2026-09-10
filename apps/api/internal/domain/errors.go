package domain

import "errors"

// Sentinel errors let callers use errors.Is instead of matching on string
// messages or a specific repository implementation's error types. Any
// UserRepository implementation (in-memory, Postgres, ...) is expected to
// return these where they apply.
var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUsernameTaken      = errors.New("username already taken")
	ErrEmailTaken         = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrTokenGeneration = errors.New("error generating a token")
	ErrGameCantBeCreated = errors.New("game cant be created")
	ErrGameNotFound = errors.New("game not found")
	ErrNotYourTurn = errors.New("not your turn")
	ErrIllegalMove = errors.New("illegal move")
	ErrGameAlreadyFinished = errors.New("game already finished")
	ErrUserNotConnected = errors.New("user not connected")
	ErrSendBufferFull = errors.New("client send buffer full, disconnecting")
)