package domain

import "time"

type FriendshipStatus string

const (
	FriendshipPending  FriendshipStatus = "pending"
	FriendshipAccepted FriendshipStatus = "accepted"
)

// Friendship is one row per unordered pair of users. RequesterID is
// whoever sent the request; only AddresseeID may accept it.
type Friendship struct {
	RequesterID string
	AddresseeID string
	Status      FriendshipStatus
	CreatedAt   time.Time
	AcceptedAt  *time.Time
}

type Friend struct {
	UserID   string
	Username string
	Rating   int32
	Since    time.Time
}

// FriendRequest describes the other side of a pending request, from the
// point of view of whoever is listing it.
type FriendRequest struct {
	UserID    string
	Username  string
	Rating    int32
	CreatedAt time.Time
}

type BlockedUser struct {
	UserID    string
	Username  string
	BlockedAt time.Time
}

// Relationship is how a search result relates to the user searching.
type Relationship string

const (
	RelationshipNone     Relationship = "none"
	RelationshipFriend   Relationship = "friend"
	RelationshipOutgoing Relationship = "outgoing"
	RelationshipIncoming Relationship = "incoming"
)

type UserSearchResult struct {
	UserID       string
	Username     string
	Rating       int32
	Relationship Relationship
}
