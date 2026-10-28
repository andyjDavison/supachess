package ws

import (
	"context"
	"encoding/json"
	"fmt"

	"api/internal/domain"
	"api/internal/transport/dto"
)

// ChatNotifier implements chat.Notifier, the same structural relationship
// MatchmakingNotifier and InviteNotifier have with their services.
type ChatNotifier struct {
	hub *Hub
}

func NewChatNotifier(hub *Hub) *ChatNotifier {
	return &ChatNotifier{hub: hub}
}

type directMessagePayload struct {
	Conversation dto.ConversationDTO  `json:"conversation"`
	Message      dto.DirectMessageDTO `json:"message"`
}

type gameMessagePayload struct {
	GameID  string             `json:"gameId"`
	Message dto.GameMessageDTO `json:"message"`
}

func (n *ChatNotifier) NotifyDirectMessage(_ context.Context, userID string, conv *domain.ConversationSummary, msg *domain.DirectMessage) error {
	return n.send(userID, "chat.direct_message", directMessagePayload{
		Conversation: dto.NewConversationDTO(conv),
		Message:      dto.NewDirectMessageDTO(msg),
	})
}

func (n *ChatNotifier) NotifyGameMessage(_ context.Context, userID string, msg *domain.GameMessage) error {
	return n.send(userID, "chat.game_message", gameMessagePayload{
		GameID:  msg.GameID,
		Message: dto.NewGameMessageDTO(msg),
	})
}

func (n *ChatNotifier) send(userID, msgType string, payload any) error {
	data, err := json.Marshal(envelope{Type: msgType, Payload: payload})
	if err != nil {
		return fmt.Errorf("marshaling %s envelope: %w", msgType, err)
	}
	return n.hub.SendToUser(userID, data)
}
