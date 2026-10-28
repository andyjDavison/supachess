package dto

import "api/internal/domain"

// Chat payloads, shared by the REST handlers and the WS notifier so a
// conversation looks the same whichever way it arrives.

type DirectMessageDTO struct {
	ID             int64  `json:"id"`
	ConversationID string `json:"conversationId"`
	SenderID       string `json:"senderId"`
	Body           string `json:"body"`
	SentAt         int64  `json:"sentAt"` // unix millis
}

func NewDirectMessageDTO(m *domain.DirectMessage) DirectMessageDTO {
	return DirectMessageDTO{
		ID:             m.ID,
		ConversationID: m.ConversationID,
		SenderID:       m.SenderID,
		Body:           m.Body,
		SentAt:         m.SentAt.UnixMilli(),
	}
}

type ChatUserDTO struct {
	UserID   string `json:"userId"`
	Username string `json:"username"`
	Rating   int32  `json:"rating"`
}

type ConversationDTO struct {
	ID          string            `json:"id"`
	Other       ChatUserDTO       `json:"other"`
	Folder      domain.Folder     `json:"folder"`
	Unread      bool              `json:"unread"`
	LastMessage *DirectMessageDTO `json:"lastMessage"` // null for a conversation with no messages
}

func NewConversationDTO(c *domain.ConversationSummary) ConversationDTO {
	out := ConversationDTO{
		ID:     c.ID,
		Other:  ChatUserDTO{UserID: c.Other.UserID, Username: c.Other.Username, Rating: c.Other.Rating},
		Folder: c.Folder,
		Unread: c.Unread,
	}
	if c.LastMessage != nil {
		last := NewDirectMessageDTO(c.LastMessage)
		out.LastMessage = &last
	}
	return out
}

type GameMessageDTO struct {
	ID       int64  `json:"id"`
	GameID   string `json:"gameId"`
	SenderID string `json:"senderId"`
	Body     string `json:"body"`
	SentAt   int64  `json:"sentAt"` // unix millis
}

func NewGameMessageDTO(m *domain.GameMessage) GameMessageDTO {
	return GameMessageDTO{ID: m.ID, GameID: m.GameID, SenderID: m.SenderID, Body: m.Body, SentAt: m.SentAt.UnixMilli()}
}
