package chattest

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"sync"
	"time"

	"api/internal/domain"
)

// Repo is an in-memory chat.ChatRepository, shared by the service and
// handler tests. It derives folders and unread flags the same way the
// Postgres summary query does. Users are seeded by username, which doubles
// as their ID.
type Repo struct {
	mu            sync.Mutex
	users         map[string]*domain.User
	blocks        map[[2]string]bool // {blocker, blocked}
	conversations map[string]*conversation
	byPair        map[[2]string]string
	dms           map[string][]domain.DirectMessage
	gameMsgs      map[string][]domain.GameMessage
	nextID        int64
	now           time.Time
}

type conversation struct {
	id      string
	members map[string]*member
}

type member struct {
	replied    bool
	archivedAt *time.Time
	lastReadAt *time.Time
}

func NewRepo(usernames ...string) *Repo {
	r := &Repo{
		users:         map[string]*domain.User{},
		blocks:        map[[2]string]bool{},
		conversations: map[string]*conversation{},
		byPair:        map[[2]string]string{},
		dms:           map[string][]domain.DirectMessage{},
		gameMsgs:      map[string][]domain.GameMessage{},
		now:           time.Unix(1_700_000_000, 0).UTC(),
	}
	for i, name := range usernames {
		r.users[name] = &domain.User{UserID: name, Username: name, Rating: int32(1000 + i)}
	}
	return r
}

// Block records that blocker has blocked blocked, which hides their
// conversation from blocker's list (as the Postgres query does).
func (r *Repo) Block(blocker, blocked string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.blocks[[2]string{blocker, blocked}] = true
}

func (r *Repo) tick() time.Time {
	r.now = r.now.Add(time.Second)
	return r.now
}

func pair(a, b string) [2]string {
	if a > b {
		a, b = b, a
	}
	return [2]string{a, b}
}

func (r *Repo) AppendDirectMessage(_ context.Context, fromID, toID, body string) (*domain.DirectMessage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.users[fromID] == nil || r.users[toID] == nil {
		return nil, domain.ErrUserNotFound
	}

	id, ok := r.byPair[pair(fromID, toID)]
	if !ok {
		id = fmt.Sprintf("conv-%d", len(r.conversations)+1)
		r.byPair[pair(fromID, toID)] = id
		r.conversations[id] = &conversation{id: id, members: map[string]*member{fromID: {}, toID: {}}}
	}
	conv := r.conversations[id]

	r.nextID++
	at := r.tick()
	msg := domain.DirectMessage{ID: r.nextID, ConversationID: id, SenderID: fromID, Body: body, SentAt: at}
	r.dms[id] = append(r.dms[id], msg)

	sender := conv.members[fromID]
	sender.replied = true
	sender.archivedAt = nil
	sender.lastReadAt = &at
	conv.members[toID].archivedAt = nil
	return &msg, nil
}

func (r *Repo) summaryLocked(conv *conversation, userID string) (*domain.ConversationSummary, bool) {
	m, ok := conv.members[userID]
	if !ok {
		return nil, false
	}
	var otherID string
	for id := range conv.members {
		if id != userID {
			otherID = id
		}
	}
	if r.blocks[[2]string{userID, otherID}] {
		return nil, false
	}

	other := r.users[otherID]
	s := &domain.ConversationSummary{
		ID:     conv.id,
		Other:  domain.ChatUser{UserID: other.UserID, Username: other.Username, Rating: other.Rating},
		Folder: domain.FolderUnread,
	}
	switch {
	case m.archivedAt != nil:
		s.Folder = domain.FolderArchived
	case m.replied:
		s.Folder = domain.FolderInbox
	}
	if msgs := r.dms[conv.id]; len(msgs) > 0 {
		last := msgs[len(msgs)-1]
		s.LastMessage = &last
		s.Unread = last.SenderID != userID && (m.lastReadAt == nil || last.SentAt.After(*m.lastReadAt))
	}
	return s, true
}

func (r *Repo) ListConversations(_ context.Context, userID string, limit int) ([]domain.ConversationSummary, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []domain.ConversationSummary{}
	for _, conv := range r.conversations {
		if s, ok := r.summaryLocked(conv, userID); ok {
			out = append(out, *s)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].LastMessage.ID > out[j].LastMessage.ID })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (r *Repo) GetConversation(_ context.Context, convID, userID string) (*domain.ConversationSummary, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	conv, ok := r.conversations[convID]
	if !ok {
		return nil, domain.ErrConversationNotFound
	}
	s, ok := r.summaryLocked(conv, userID)
	if !ok {
		return nil, domain.ErrConversationNotFound
	}
	return s, nil
}

func (r *Repo) ListDirectMessages(_ context.Context, convID string, beforeID int64, limit int) ([]domain.DirectMessage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var page []domain.DirectMessage
	msgs := r.dms[convID]
	for i := len(msgs) - 1; i >= 0 && len(page) < limit; i-- {
		if beforeID == 0 || msgs[i].ID < beforeID {
			page = append(page, msgs[i])
		}
	}
	slices.Reverse(page)
	if page == nil {
		page = []domain.DirectMessage{}
	}
	return page, nil
}

func (r *Repo) memberLocked(convID, userID string) (*member, error) {
	conv, ok := r.conversations[convID]
	if !ok {
		return nil, domain.ErrConversationNotFound
	}
	m, ok := conv.members[userID]
	if !ok {
		return nil, domain.ErrConversationNotFound
	}
	return m, nil
}

func (r *Repo) MarkRead(_ context.Context, convID, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	m, err := r.memberLocked(convID, userID)
	if err != nil {
		return err
	}
	at := r.tick()
	m.lastReadAt = &at
	return nil
}

func (r *Repo) SetArchived(_ context.Context, convID, userID string, archived bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	m, err := r.memberLocked(convID, userID)
	if err != nil {
		return err
	}
	if archived {
		at := r.tick()
		m.archivedAt = &at
	} else {
		m.archivedAt = nil
	}
	return nil
}

func (r *Repo) AppendGameMessage(_ context.Context, gameID, senderID, body string) (*domain.GameMessage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID++
	msg := domain.GameMessage{ID: r.nextID, GameID: gameID, SenderID: senderID, Body: body, SentAt: r.tick()}
	r.gameMsgs[gameID] = append(r.gameMsgs[gameID], msg)
	return &msg, nil
}

func (r *Repo) ListGameMessages(_ context.Context, gameID string, limit int) ([]domain.GameMessage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	msgs := append([]domain.GameMessage{}, r.gameMsgs[gameID]...)
	if len(msgs) > limit {
		msgs = msgs[:limit]
	}
	return msgs, nil
}
