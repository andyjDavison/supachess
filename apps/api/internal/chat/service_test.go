package chat

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"api/internal/chat/chattest"
	"api/internal/domain"
)

var ctx = context.Background()

var (
	_ ChatRepository = (*PostgresChatRepository)(nil)
	_ ChatRepository = (*chattest.Repo)(nil)
)

// --- doubles ---

type fakeBlocks map[[2]string]bool

func (b fakeBlocks) IsBlocked(_ context.Context, x, y string) (bool, error) {
	return b[[2]string{x, y}] || b[[2]string{y, x}], nil
}

type fakeGames map[string]*domain.Game

func (g fakeGames) GetGame(_ context.Context, id string) (*domain.Game, error) {
	if game, ok := g[id]; ok {
		copied := *game
		return &copied, nil
	}
	return nil, domain.ErrGameNotFound
}

type dmNote struct {
	to   string
	conv domain.ConversationSummary
	body string
}

type spyNotifier struct {
	mu    sync.Mutex
	dms   []dmNote
	games []string // "to:gameID:body"
}

func (n *spyNotifier) NotifyDirectMessage(_ context.Context, userID string, conv *domain.ConversationSummary, msg *domain.DirectMessage) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.dms = append(n.dms, dmNote{to: userID, conv: *conv, body: msg.Body})
	return nil
}

func (n *spyNotifier) NotifyGameMessage(_ context.Context, userID string, msg *domain.GameMessage) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.games = append(n.games, userID+":"+msg.GameID+":"+msg.Body)
	return nil
}

type harness struct {
	svc    *ChatService
	repo   *chattest.Repo
	blocks fakeBlocks
	games  fakeGames
	notify *spyNotifier
	clock  time.Time
}

func newHarness() *harness {
	h := &harness{
		repo:   chattest.NewRepo("alice", "bob", "carol"),
		blocks: fakeBlocks{},
		games: fakeGames{
			"active-game":   {ID: "active-game", WhiteID: "alice", BlackID: "bob", Status: domain.GameStatusActive},
			"finished-game": {ID: "finished-game", WhiteID: "alice", BlackID: "bob", Status: domain.GameStatusFinished},
		},
		notify: &spyNotifier{},
		clock:  time.Unix(1_700_000_000, 0).UTC(),
	}
	h.svc = NewChatService(h.repo, h.blocks, h.games, h.notify)
	h.svc.now = func() time.Time { return h.clock }
	return h
}

// send advances the clock past the rate limit first, so tests that send a
// lot aren't throttled unless they mean to be.
func (h *harness) send(t *testing.T, from, to, body string) *domain.DirectMessage {
	t.Helper()
	h.clock = h.clock.Add(time.Minute)
	msg, err := h.svc.SendDirect(ctx, from, to, body)
	if err != nil {
		t.Fatalf("SendDirect(%s -> %s): %v", from, to, err)
	}
	return msg
}

func (h *harness) conv(t *testing.T, convID, userID string) domain.ConversationSummary {
	t.Helper()
	convs, err := h.svc.ListConversations(ctx, userID)
	if err != nil {
		t.Fatalf("ListConversations(%s): %v", userID, err)
	}
	for _, c := range convs {
		if c.ID == convID {
			return c
		}
	}
	t.Fatalf("%s's list has no conversation %s: %+v", userID, convID, convs)
	return domain.ConversationSummary{}
}

// --- folders ---

func TestFirstMessageIsInboxForSenderAndUnreadForRecipient(t *testing.T) {
	h := newHarness()

	msg := h.send(t, "alice", "bob", "  hi bob  ")

	if msg.Body != "hi bob" {
		t.Errorf("body = %q, want it trimmed", msg.Body)
	}
	alice := h.conv(t, msg.ConversationID, "alice")
	if alice.Folder != domain.FolderInbox || alice.Unread {
		t.Errorf("alice: folder %q unread %v, want inbox and read", alice.Folder, alice.Unread)
	}
	if alice.Other.UserID != "bob" || alice.LastMessage == nil || alice.LastMessage.Body != "hi bob" {
		t.Errorf("alice's summary = %+v", alice)
	}
	bob := h.conv(t, msg.ConversationID, "bob")
	if bob.Folder != domain.FolderUnread || !bob.Unread || bob.Other.UserID != "alice" {
		t.Errorf("bob: %+v, want unread folder, unseen, other=alice", bob)
	}
}

func TestReplyMovesConversationToInbox(t *testing.T) {
	h := newHarness()
	msg := h.send(t, "alice", "bob", "hi")

	h.send(t, "bob", "alice", "hey")

	bob := h.conv(t, msg.ConversationID, "bob")
	if bob.Folder != domain.FolderInbox || bob.Unread {
		t.Errorf("bob after replying: folder %q unread %v, want inbox and read", bob.Folder, bob.Unread)
	}
	alice := h.conv(t, msg.ConversationID, "alice")
	if alice.Folder != domain.FolderInbox || !alice.Unread {
		t.Errorf("alice after bob's reply: folder %q unread %v, want inbox and unseen", alice.Folder, alice.Unread)
	}
}

func TestSamePairSharesOneConversation(t *testing.T) {
	h := newHarness()
	first := h.send(t, "alice", "bob", "1")
	second := h.send(t, "bob", "alice", "2")

	if first.ConversationID != second.ConversationID {
		t.Errorf("conversations %s and %s, want one per pair", first.ConversationID, second.ConversationID)
	}
}

func TestArchiveAndNewMessageUnarchives(t *testing.T) {
	h := newHarness()
	msg := h.send(t, "alice", "bob", "hi")

	if err := h.svc.Archive(ctx, "bob", msg.ConversationID); err != nil {
		t.Fatalf("Archive: %v", err)
	}
	if got := h.conv(t, msg.ConversationID, "bob").Folder; got != domain.FolderArchived {
		t.Fatalf("bob's folder = %q, want archived", got)
	}
	// Archiving is per member.
	if got := h.conv(t, msg.ConversationID, "alice").Folder; got != domain.FolderInbox {
		t.Errorf("alice's folder = %q, want it untouched", got)
	}

	h.send(t, "alice", "bob", "you there?")

	// Back where it was before archiving: bob still hasn't replied.
	if got := h.conv(t, msg.ConversationID, "bob").Folder; got != domain.FolderUnread {
		t.Errorf("bob's folder after a new message = %q, want unread", got)
	}
}

func TestSendingUnarchivesForTheSenderToo(t *testing.T) {
	h := newHarness()
	msg := h.send(t, "alice", "bob", "hi")
	h.svc.Archive(ctx, "alice", msg.ConversationID)

	h.send(t, "alice", "bob", "again")

	if got := h.conv(t, msg.ConversationID, "alice").Folder; got != domain.FolderInbox {
		t.Errorf("alice's folder = %q, want inbox", got)
	}
}

func TestUnarchive(t *testing.T) {
	h := newHarness()
	msg := h.send(t, "alice", "bob", "hi")
	h.send(t, "bob", "alice", "hey")
	h.svc.Archive(ctx, "bob", msg.ConversationID)

	if err := h.svc.Unarchive(ctx, "bob", msg.ConversationID); err != nil {
		t.Fatalf("Unarchive: %v", err)
	}
	if got := h.conv(t, msg.ConversationID, "bob").Folder; got != domain.FolderInbox {
		t.Errorf("folder = %q, want inbox (bob has replied)", got)
	}
}

func TestMarkRead(t *testing.T) {
	h := newHarness()
	msg := h.send(t, "alice", "bob", "hi")

	if err := h.svc.MarkRead(ctx, "bob", msg.ConversationID); err != nil {
		t.Fatalf("MarkRead: %v", err)
	}
	bob := h.conv(t, msg.ConversationID, "bob")
	if bob.Unread {
		t.Error("still unseen after MarkRead")
	}
	// Reading isn't replying: it stays in Unread.
	if bob.Folder != domain.FolderUnread {
		t.Errorf("folder = %q, want unread", bob.Folder)
	}
}

func TestMembershipIsRequired(t *testing.T) {
	h := newHarness()
	msg := h.send(t, "alice", "bob", "secret")

	if _, err := h.svc.GetMessages(ctx, "carol", msg.ConversationID, 0); !errors.Is(err, domain.ErrConversationNotFound) {
		t.Errorf("carol reading: err = %v, want ErrConversationNotFound", err)
	}
	if err := h.svc.MarkRead(ctx, "carol", msg.ConversationID); !errors.Is(err, domain.ErrConversationNotFound) {
		t.Errorf("carol marking read: err = %v, want ErrConversationNotFound", err)
	}
	if err := h.svc.Archive(ctx, "carol", msg.ConversationID); !errors.Is(err, domain.ErrConversationNotFound) {
		t.Errorf("carol archiving: err = %v, want ErrConversationNotFound", err)
	}
}

func TestGetMessagesPagesNewestFirstButReturnsOldestFirst(t *testing.T) {
	h := newHarness()
	var convID string
	for i := 0; i < messagePageSize+5; i++ {
		convID = h.send(t, "alice", "bob", "m").ConversationID
	}

	latest, err := h.svc.GetMessages(ctx, "bob", convID, 0)
	if err != nil {
		t.Fatalf("GetMessages: %v", err)
	}
	if len(latest) != messagePageSize {
		t.Fatalf("got %d messages, want a page of %d", len(latest), messagePageSize)
	}
	if latest[0].ID >= latest[len(latest)-1].ID {
		t.Error("page is not oldest-first")
	}

	older, _ := h.svc.GetMessages(ctx, "bob", convID, latest[0].ID)
	if len(older) != 5 || older[len(older)-1].ID >= latest[0].ID {
		t.Errorf("older page = %d messages ending at %d, want the 5 before %d", len(older), older[len(older)-1].ID, latest[0].ID)
	}
}

// --- notifications ---

// Both members hear about every message, each with their own view.
func TestSendNotifiesBothWithTheirOwnView(t *testing.T) {
	h := newHarness()

	h.send(t, "alice", "bob", "hi")

	if len(h.notify.dms) != 2 {
		t.Fatalf("notifications = %+v, want 2", h.notify.dms)
	}
	byUser := map[string]dmNote{}
	for _, n := range h.notify.dms {
		byUser[n.to] = n
	}
	if a := byUser["alice"]; a.conv.Other.UserID != "bob" || a.conv.Folder != domain.FolderInbox || a.conv.Unread {
		t.Errorf("alice's notification = %+v", a.conv)
	}
	if b := byUser["bob"]; b.conv.Other.UserID != "alice" || b.conv.Folder != domain.FolderUnread || !b.conv.Unread || b.body != "hi" {
		t.Errorf("bob's notification = %+v", b)
	}
}

// --- send rejections ---

func TestSendDirectRejections(t *testing.T) {
	cases := []struct {
		name  string
		setup func(*harness)
		to    string
		body  string
		want  error
	}{
		{"empty", nil, "bob", "", domain.ErrInvalidMessageBody},
		{"whitespace only", nil, "bob", " \n\t ", domain.ErrInvalidMessageBody},
		{"too long", nil, "bob", strings.Repeat("a", domain.MaxMessageRunes+1), domain.ErrInvalidMessageBody},
		{"self", nil, "alice", "hi", domain.ErrCannotMessageSelf},
		{"unknown user", nil, "nobody", "hi", domain.ErrUserNotFound},
		{"recipient blocked sender", func(h *harness) { h.blocks[[2]string{"bob", "alice"}] = true }, "bob", "hi", domain.ErrUserNotFound},
		{"sender blocked recipient", func(h *harness) { h.blocks[[2]string{"alice", "bob"}] = true }, "bob", "hi", domain.ErrUserNotFound},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := newHarness()
			if c.setup != nil {
				c.setup(h)
			}
			if _, err := h.svc.SendDirect(ctx, "alice", c.to, c.body); !errors.Is(err, c.want) {
				t.Fatalf("err = %v, want %v", err, c.want)
			}
			if len(h.notify.dms) != 0 {
				t.Error("a rejected send notified someone")
			}
		})
	}
}

// The client sees err.Error(), so a block and an unknown user must not
// differ even in their message text - or the silent block isn't silent.
func TestUnknownUserAndBlockAreIndistinguishable(t *testing.T) {
	h := newHarness()
	h.blocks[[2]string{"bob", "alice"}] = true

	_, blockedErr := h.svc.SendDirect(ctx, "alice", "bob", "hi")
	h.clock = h.clock.Add(time.Minute)
	_, unknownErr := h.svc.SendDirect(ctx, "alice", "nobody", "hi")

	if blockedErr == nil || unknownErr == nil || blockedErr.Error() != unknownErr.Error() {
		t.Errorf("blocked: %v; unknown: %v - want identical", blockedErr, unknownErr)
	}
}

// The limit is 500 characters, not bytes.
func TestLengthLimitCountsCharacters(t *testing.T) {
	h := newHarness()

	if _, err := h.svc.SendDirect(ctx, "alice", "bob", strings.Repeat("é", domain.MaxMessageRunes)); err != nil {
		t.Errorf("500 two-byte characters rejected: %v", err)
	}
}

func TestSendDirectIsRateLimited(t *testing.T) {
	h := newHarness()

	for i := 0; i < rateCapacity; i++ {
		if _, err := h.svc.SendDirect(ctx, "alice", "bob", "spam"); err != nil {
			t.Fatalf("message %d within the burst: %v", i+1, err)
		}
	}
	if _, err := h.svc.SendDirect(ctx, "alice", "bob", "spam"); !errors.Is(err, domain.ErrRateLimited) {
		t.Fatalf("over the burst: err = %v, want ErrRateLimited", err)
	}
	// It's per sender: bob isn't affected by alice's burst.
	if _, err := h.svc.SendDirect(ctx, "bob", "alice", "hi"); err != nil {
		t.Errorf("bob was throttled by alice's burst: %v", err)
	}

	h.clock = h.clock.Add(time.Second)
	if _, err := h.svc.SendDirect(ctx, "alice", "bob", "later"); err != nil {
		t.Errorf("after a second's refill: %v", err)
	}
}

func TestConversationWithSomeoneIBlockedIsHidden(t *testing.T) {
	h := newHarness()
	msg := h.send(t, "alice", "bob", "hi")

	h.repo.Block("bob", "alice")

	convs, _ := h.svc.ListConversations(ctx, "bob")
	if len(convs) != 0 {
		t.Errorf("bob still lists %+v after blocking alice", convs)
	}
	if _, err := h.svc.GetMessages(ctx, "bob", msg.ConversationID, 0); !errors.Is(err, domain.ErrConversationNotFound) {
		t.Errorf("bob reading the hidden thread: err = %v, want ErrConversationNotFound", err)
	}
}

// --- game chat ---

func TestSendGameReachesBothPlayers(t *testing.T) {
	h := newHarness()

	msg, err := h.svc.SendGame(ctx, "bob", "active-game", " gg ")
	if err != nil {
		t.Fatalf("SendGame: %v", err)
	}
	if msg.Body != "gg" || msg.SenderID != "bob" {
		t.Errorf("message = %+v", msg)
	}
	want := []string{"alice:active-game:gg", "bob:active-game:gg"}
	if strings.Join(h.notify.games, ",") != strings.Join(want, ",") {
		t.Errorf("notified %v, want %v", h.notify.games, want)
	}

	log, err := h.svc.GameMessages(ctx, "alice", "active-game")
	if err != nil || len(log) != 1 {
		t.Errorf("log = %v, %v; want the one message", log, err)
	}
}

func TestSendGameRejections(t *testing.T) {
	cases := []struct {
		name, from, gameID, body string
		want                     error
	}{
		{"non-player", "carol", "active-game", "hi", domain.ErrGameNotFound},
		{"unknown game", "alice", "nope", "hi", domain.ErrGameNotFound},
		{"finished game", "alice", "finished-game", "hi", domain.ErrGameAlreadyFinished},
		{"empty body", "alice", "active-game", "  ", domain.ErrInvalidMessageBody},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := newHarness()
			if _, err := h.svc.SendGame(ctx, c.from, c.gameID, c.body); !errors.Is(err, c.want) {
				t.Errorf("err = %v, want %v", err, c.want)
			}
			if len(h.notify.games) != 0 {
				t.Error("a rejected game message was broadcast")
			}
		})
	}
}

// The log stays readable after the game ends - to its players only.
func TestGameMessagesAccess(t *testing.T) {
	h := newHarness()
	h.svc.SendGame(ctx, "alice", "active-game", "good luck")
	h.games["active-game"].Status = domain.GameStatusFinished

	log, err := h.svc.GameMessages(ctx, "bob", "active-game")
	if err != nil || len(log) != 1 {
		t.Errorf("bob reading the finished game's chat: %v, %v", log, err)
	}
	if _, err := h.svc.GameMessages(ctx, "carol", "active-game"); !errors.Is(err, domain.ErrGameNotFound) {
		t.Errorf("carol reading: err = %v, want ErrGameNotFound", err)
	}
}

// DMs and game chat share one bucket.
func TestRateLimitIsSharedAcrossDMAndGameChat(t *testing.T) {
	h := newHarness()
	for i := 0; i < rateCapacity; i++ {
		h.svc.SendDirect(ctx, "alice", "bob", "dm")
	}

	if _, err := h.svc.SendGame(ctx, "alice", "active-game", "hi"); !errors.Is(err, domain.ErrRateLimited) {
		t.Errorf("err = %v, want ErrRateLimited", err)
	}
}
