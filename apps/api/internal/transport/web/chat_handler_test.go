package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"api/internal/auth"
	"api/internal/chat"
	"api/internal/chat/chattest"
	"api/internal/domain"
)

type noBlocks struct{}

func (noBlocks) IsBlocked(context.Context, string, string) (bool, error) { return false, nil }

type chatGames map[string]*domain.Game

func (g chatGames) GetGame(_ context.Context, id string) (*domain.Game, error) {
	if game, ok := g[id]; ok {
		return game, nil
	}
	return nil, domain.ErrGameNotFound
}

type quietChatNotifier struct{}

func (quietChatNotifier) NotifyDirectMessage(context.Context, string, *domain.ConversationSummary, *domain.DirectMessage) error {
	return nil
}
func (quietChatNotifier) NotifyGameMessage(context.Context, string, *domain.GameMessage) error {
	return nil
}

// callAsPath runs handler as userID through RequireAuth, like callAs, but
// takes any path values (chat routes use {id}, not {userId}).
func callAsPath(t *testing.T, userID string, handler http.HandlerFunc, method, target string, path map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	token, err := auth.GenerateToken(userID, friendsTestSecret, time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	req := httptest.NewRequest(method, target, nil)
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: token})
	for k, v := range path {
		req.SetPathValue(k, v)
	}
	rec := httptest.NewRecorder()
	auth.RequireAuth(friendsTestSecret)(handler).ServeHTTP(rec, req)
	return rec
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

// newChatHandler returns a handler plus its service, so tests can seed
// messages through the service's send path (which is a WebSocket message,
// not a route).
func newChatHandler() (*ChatHandler, *chat.ChatService) {
	svc := chat.NewChatService(
		chattest.NewRepo("alice", "bob", "carol"),
		noBlocks{},
		chatGames{"game-1": {ID: "game-1", WhiteID: "alice", BlackID: "bob", Status: domain.GameStatusActive}},
		quietChatNotifier{},
	)
	return NewChatHandler(svc), svc
}

func TestListConversationsShape(t *testing.T) {
	h, svc := newChatHandler()
	msg, err := svc.SendDirect(context.Background(), "alice", "bob", "hi bob")
	if err != nil {
		t.Fatalf("seeding: %v", err)
	}

	rec := callAsPath(t, "bob", h.ListConversationsHandler, http.MethodGet, "/api/conversations", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}
	got := decode[ConversationsResponse](t, rec)
	if len(got.Conversations) != 1 {
		t.Fatalf("conversations = %+v, want 1", got.Conversations)
	}
	c := got.Conversations[0]
	if c.ID != msg.ConversationID || c.Other.UserID != "alice" || c.Folder != domain.FolderUnread || !c.Unread {
		t.Errorf("bob's conversation = %+v", c)
	}
	if c.LastMessage == nil || c.LastMessage.Body != "hi bob" || c.LastMessage.SentAt == 0 {
		t.Errorf("lastMessage = %+v, want the message with a unix-ms sentAt", c.LastMessage)
	}
}

func TestListConversationsEmptyIsAnArray(t *testing.T) {
	h, _ := newChatHandler()

	rec := callAsPath(t, "carol", h.ListConversationsHandler, http.MethodGet, "/api/conversations", nil)

	if got := decode[ConversationsResponse](t, rec); got.Conversations == nil {
		t.Error("conversations encoded as null; the client expects []")
	}
}

func TestListMessagesAndPaging(t *testing.T) {
	h, svc := newChatHandler()
	first, _ := svc.SendDirect(context.Background(), "alice", "bob", "one")
	second, _ := svc.SendDirect(context.Background(), "bob", "alice", "two")
	path := map[string]string{"id": first.ConversationID}

	rec := callAsPath(t, "alice", h.ListMessagesHandler, http.MethodGet, "/x", path)
	got := decode[DirectMessagesResponse](t, rec)
	if len(got.Messages) != 2 || got.Messages[0].Body != "one" || got.Messages[1].Body != "two" {
		t.Fatalf("messages = %+v, want [one two] oldest first", got.Messages)
	}

	rec = callAsPath(t, "alice", h.ListMessagesHandler, http.MethodGet, "/x?before="+itoa(second.ID), path)
	if got := decode[DirectMessagesResponse](t, rec); len(got.Messages) != 1 || got.Messages[0].Body != "one" {
		t.Errorf("page before %d = %+v, want [one]", second.ID, got.Messages)
	}
}

func TestChatHandlerStatuses(t *testing.T) {
	quietLogs(t)
	h, svc := newChatHandler()
	msg, _ := svc.SendDirect(context.Background(), "alice", "bob", "hi")
	conv := map[string]string{"id": msg.ConversationID}

	cases := []struct {
		name    string
		as      string
		handler http.HandlerFunc
		method  string
		target  string
		path    map[string]string
		want    int
	}{
		{"non-member reads messages", "carol", h.ListMessagesHandler, http.MethodGet, "/x", conv, http.StatusNotFound},
		{"bad cursor", "alice", h.ListMessagesHandler, http.MethodGet, "/x?before=abc", conv, http.StatusBadRequest},
		{"negative cursor", "alice", h.ListMessagesHandler, http.MethodGet, "/x?before=-1", conv, http.StatusBadRequest},
		{"mark read", "bob", h.MarkReadHandler, http.MethodPost, "/x", conv, http.StatusNoContent},
		{"non-member marks read", "carol", h.MarkReadHandler, http.MethodPost, "/x", conv, http.StatusNotFound},
		{"archive", "bob", h.ArchiveHandler, http.MethodPut, "/x", conv, http.StatusNoContent},
		{"unarchive", "bob", h.UnarchiveHandler, http.MethodDelete, "/x", conv, http.StatusNoContent},
		{"unknown conversation", "bob", h.ArchiveHandler, http.MethodPut, "/x", map[string]string{"id": "nope"}, http.StatusNotFound},
		{"player reads game chat", "alice", h.GameMessagesHandler, http.MethodGet, "/x", map[string]string{"id": "game-1"}, http.StatusOK},
		{"non-player reads game chat", "carol", h.GameMessagesHandler, http.MethodGet, "/x", map[string]string{"id": "game-1"}, http.StatusNotFound},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if rec := callAsPath(t, c.as, c.handler, c.method, c.target, c.path); rec.Code != c.want {
				t.Errorf("status = %d, want %d: %s", rec.Code, c.want, rec.Body.String())
			}
		})
	}
}

func TestArchiveMovesFolder(t *testing.T) {
	h, svc := newChatHandler()
	msg, _ := svc.SendDirect(context.Background(), "alice", "bob", "hi")

	callAsPath(t, "bob", h.ArchiveHandler, http.MethodPut, "/x", map[string]string{"id": msg.ConversationID})

	rec := callAsPath(t, "bob", h.ListConversationsHandler, http.MethodGet, "/api/conversations", nil)
	if got := decode[ConversationsResponse](t, rec); got.Conversations[0].Folder != domain.FolderArchived {
		t.Errorf("folder = %q, want archived", got.Conversations[0].Folder)
	}
}

func TestGameMessagesShape(t *testing.T) {
	h, svc := newChatHandler()
	if _, err := svc.SendGame(context.Background(), "bob", "game-1", "gg"); err != nil {
		t.Fatalf("seeding: %v", err)
	}

	rec := callAsPath(t, "alice", h.GameMessagesHandler, http.MethodGet, "/x", map[string]string{"id": "game-1"})
	got := decode[GameMessagesResponse](t, rec)
	if len(got.Messages) != 1 || got.Messages[0].SenderID != "bob" || got.Messages[0].Body != "gg" || got.Messages[0].GameID != "game-1" {
		t.Errorf("messages = %+v", got.Messages)
	}
}
