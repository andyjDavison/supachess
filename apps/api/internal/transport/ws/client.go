package ws

type Client interface {
	ID() string // the user's domain.UserID, as a string
	Send(message []byte) error
}