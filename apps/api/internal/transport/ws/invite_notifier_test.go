package ws

import (
	"context"
	"testing"
	"time"

	"api/internal/domain"
	"api/internal/invite"
	"api/internal/matchmaking"
)

// Pin the invite.* envelope shapes from websocket-protocol.md, same reason
// as publishers_test.go.

func sampleInvite() *invite.Invite {
	return &invite.Invite{
		GameID:    "game-7",
		Inviter:   &domain.User{UserID: "alice-id", Username: "alice", Rating: 1500},
		Invitee:   &domain.User{UserID: "bob-id", Username: "bob", Rating: 1300},
		Preset:    matchmaking.Preset3_2,
		ExpiresAt: time.UnixMilli(1_700_000_060_000).UTC(),
	}
}

func onlyMessage(t *testing.T, c *fakeClient) (string, map[string]any) {
	t.Helper()
	msgs := c.messages()
	if len(msgs) != 1 {
		t.Fatalf("%s received %d messages, want 1", c.id, len(msgs))
	}
	return decodeEnvelope(t, msgs[0])
}

func TestInviteSentAndReceivedEnvelopes(t *testing.T) {
	h := NewHub()
	alice := &fakeClient{id: "alice-id"}
	bob := &fakeClient{id: "bob-id"}
	h.Register(alice)
	h.Register(bob)
	n := NewInviteNotifier(h)

	if err := n.NotifyInviteSent(context.Background(), sampleInvite()); err != nil {
		t.Fatalf("NotifyInviteSent: %v", err)
	}
	if err := n.NotifyInviteReceived(context.Background(), sampleInvite()); err != nil {
		t.Fatalf("NotifyInviteReceived: %v", err)
	}

	typ, p := onlyMessage(t, alice)
	if typ != "invite.sent" || p["gameId"] != "game-7" || p["preset"] != "3+2" || p["expiresAt"] != float64(1_700_000_060_000) {
		t.Errorf("inviter got %s %v", typ, p)
	}
	to, _ := p["to"].(map[string]any)
	if to["userId"] != "bob-id" || to["username"] != "bob" || to["rating"] != float64(1300) {
		t.Errorf("sent.to = %v, want bob", to)
	}

	typ, p = onlyMessage(t, bob)
	if typ != "invite.received" || p["gameId"] != "game-7" || p["preset"] != "3+2" || p["expiresAt"] != float64(1_700_000_060_000) {
		t.Errorf("invitee got %s %v", typ, p)
	}
	from, _ := p["from"].(map[string]any)
	if from["userId"] != "alice-id" || from["username"] != "alice" || from["rating"] != float64(1500) {
		t.Errorf("received.from = %v, want alice", from)
	}
}

func TestInviteAcceptedEnvelopeMatchesMatchmakingMatched(t *testing.T) {
	h := NewHub()
	alice := &fakeClient{id: "alice-id"}
	h.Register(alice)

	opponent := &domain.User{UserID: "bob-id", Username: "bob", Rating: 1300}
	if err := NewInviteNotifier(h).NotifyInviteAccepted(context.Background(), "alice-id", sampleGame(), opponent); err != nil {
		t.Fatalf("NotifyInviteAccepted: %v", err)
	}

	typ, p := onlyMessage(t, alice)
	if typ != "invite.accepted" || p["gameId"] != "game-1" {
		t.Errorf("got %s %v", typ, p)
	}
	opp, _ := p["opponent"].(map[string]any)
	if opp["username"] != "bob" || opp["rating"] != float64(1300) || len(opp) != 2 {
		t.Errorf("opponent = %v, want exactly {username, rating}", opp)
	}
}

func TestInviteClosedEnvelope(t *testing.T) {
	h := NewHub()
	bob := &fakeClient{id: "bob-id"}
	h.Register(bob)

	if err := NewInviteNotifier(h).NotifyInviteClosed(context.Background(), "bob-id", "game-7", invite.ReasonExpired); err != nil {
		t.Fatalf("NotifyInviteClosed: %v", err)
	}

	typ, p := onlyMessage(t, bob)
	if typ != "invite.closed" || p["gameId"] != "game-7" || p["reason"] != "expired" {
		t.Errorf("got %s %v", typ, p)
	}
}

// The invite service relies on this to tell "they left" from a real failure.
func TestInviteNotifierReportsOfflineRecipient(t *testing.T) {
	err := NewInviteNotifier(NewHub()).NotifyInviteClosed(context.Background(), "nobody", "game-7", invite.ReasonExpired)
	if err != domain.ErrUserNotConnected {
		t.Errorf("err = %v, want ErrUserNotConnected", err)
	}
}
