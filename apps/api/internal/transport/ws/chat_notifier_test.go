package ws

import (
	"context"
	"testing"
	"time"

	"api/internal/domain"
)

// Pin the chat.* envelope shapes from websocket-protocol.md.

func TestDirectMessageEnvelope(t *testing.T) {
	h := NewHub()
	bob := &fakeClient{id: "bob-id"}
	h.Register(bob)

	msg := &domain.DirectMessage{ID: 42, ConversationID: "conv-1", SenderID: "alice-id", Body: "hi", SentAt: time.UnixMilli(1_700_000_000_000)}
	conv := &domain.ConversationSummary{
		ID:          "conv-1",
		Other:       domain.ChatUser{UserID: "alice-id", Username: "alice", Rating: 1500},
		Folder:      domain.FolderUnread,
		Unread:      true,
		LastMessage: msg,
	}

	if err := NewChatNotifier(h).NotifyDirectMessage(context.Background(), "bob-id", conv, msg); err != nil {
		t.Fatalf("NotifyDirectMessage: %v", err)
	}

	typ, p := onlyMessage(t, bob)
	if typ != "chat.direct_message" {
		t.Fatalf("type = %q", typ)
	}
	c, _ := p["conversation"].(map[string]any)
	if c["id"] != "conv-1" || c["folder"] != "unread" || c["unread"] != true {
		t.Errorf("conversation = %v", c)
	}
	other, _ := c["other"].(map[string]any)
	if other["userId"] != "alice-id" || other["username"] != "alice" || other["rating"] != float64(1500) {
		t.Errorf("conversation.other = %v", other)
	}
	if last, _ := c["lastMessage"].(map[string]any); last["id"] != float64(42) {
		t.Errorf("conversation.lastMessage = %v", c["lastMessage"])
	}
	m, _ := p["message"].(map[string]any)
	if m["id"] != float64(42) || m["conversationId"] != "conv-1" || m["senderId"] != "alice-id" ||
		m["body"] != "hi" || m["sentAt"] != float64(1_700_000_000_000) {
		t.Errorf("message = %v", m)
	}
}

func TestGameMessageEnvelope(t *testing.T) {
	h := NewHub()
	white := &fakeClient{id: "white-1"}
	h.Register(white)

	msg := &domain.GameMessage{ID: 7, GameID: "game-1", SenderID: "black-1", Body: "gg", SentAt: time.UnixMilli(1_700_000_000_000)}
	if err := NewChatNotifier(h).NotifyGameMessage(context.Background(), "white-1", msg); err != nil {
		t.Fatalf("NotifyGameMessage: %v", err)
	}

	typ, p := onlyMessage(t, white)
	if typ != "chat.game_message" || p["gameId"] != "game-1" {
		t.Errorf("got %s %v", typ, p)
	}
	m, _ := p["message"].(map[string]any)
	if m["id"] != float64(7) || m["gameId"] != "game-1" || m["senderId"] != "black-1" || m["body"] != "gg" || m["sentAt"] != float64(1_700_000_000_000) {
		t.Errorf("message = %v", m)
	}
}
