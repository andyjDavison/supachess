package domain

import "time"

// Folder is where a conversation sits for one of its two members.
type Folder string

const (
	// FolderInbox: this user has sent at least one message in it.
	FolderInbox Folder = "inbox"
	// FolderUnread: the other user started it and this one hasn't replied.
	FolderUnread Folder = "unread"
	// FolderArchived: this user archived it. Any new message un-archives it.
	FolderArchived Folder = "archived"
)

// MaxMessageRunes is the longest chat message accepted, in characters.
const MaxMessageRunes = 500

type DirectMessage struct {
	ID             int64
	ConversationID string
	SenderID       string
	Body           string
	SentAt         time.Time
}

type ChatUser struct {
	UserID   string
	Username string
	Rating   int32
}

// ConversationSummary is one conversation from one member's point of view:
// Other is the other member, and Folder/Unread are this member's.
type ConversationSummary struct {
	ID          string
	Other       ChatUser
	Folder      Folder
	Unread      bool
	LastMessage *DirectMessage
}

type GameMessage struct {
	ID       int64
	GameID   string
	SenderID string
	Body     string
	SentAt   time.Time
}
