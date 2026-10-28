package ws

import (
	"context"
	"encoding/json"
	"fmt"

	"api/internal/domain"
	"api/internal/invite"
)

// InviteNotifier implements invite.Notifier - the invite equivalent of
// MatchmakingNotifier, satisfying the interface structurally so invite
// never imports this package.
type InviteNotifier struct {
	hub *Hub
}

func NewInviteNotifier(hub *Hub) *InviteNotifier {
	return &InviteNotifier{hub: hub}
}

type inviteUser struct {
	UserID   string `json:"userId"`
	Username string `json:"username"`
	Rating   int    `json:"rating"`
}

func newInviteUser(u *domain.User) inviteUser {
	return inviteUser{UserID: u.UserID, Username: u.Username, Rating: int(u.Rating)}
}

type inviteSentPayload struct {
	GameID    string     `json:"gameId"`
	To        inviteUser `json:"to"`
	Preset    string     `json:"preset"`
	ExpiresAt int64      `json:"expiresAt"` // unix millis
}

type inviteReceivedPayload struct {
	GameID    string     `json:"gameId"`
	From      inviteUser `json:"from"`
	Preset    string     `json:"preset"`
	ExpiresAt int64      `json:"expiresAt"` // unix millis
}

// inviteAcceptedPayload deliberately matches matchmaking.matched, so the
// client handles "you're in a game now" the same way for both.
type inviteAcceptedPayload struct {
	GameID   string `json:"gameId"`
	Opponent struct {
		Username string `json:"username"`
		Rating   int    `json:"rating"`
	} `json:"opponent"`
}

type inviteClosedPayload struct {
	GameID string `json:"gameId"`
	Reason string `json:"reason"`
}

func (n *InviteNotifier) NotifyInviteSent(_ context.Context, inv *invite.Invite) error {
	return n.send(inv.Inviter.UserID, "invite.sent", inviteSentPayload{
		GameID:    inv.GameID,
		To:        newInviteUser(inv.Invitee),
		Preset:    string(inv.Preset),
		ExpiresAt: inv.ExpiresAt.UnixMilli(),
	})
}

func (n *InviteNotifier) NotifyInviteReceived(_ context.Context, inv *invite.Invite) error {
	return n.send(inv.Invitee.UserID, "invite.received", inviteReceivedPayload{
		GameID:    inv.GameID,
		From:      newInviteUser(inv.Inviter),
		Preset:    string(inv.Preset),
		ExpiresAt: inv.ExpiresAt.UnixMilli(),
	})
}

func (n *InviteNotifier) NotifyInviteAccepted(_ context.Context, playerID string, g *domain.Game, opponent *domain.User) error {
	p := inviteAcceptedPayload{GameID: g.ID}
	p.Opponent.Username = opponent.Username
	p.Opponent.Rating = int(opponent.Rating)
	return n.send(playerID, "invite.accepted", p)
}

func (n *InviteNotifier) NotifyInviteClosed(_ context.Context, playerID, gameID string, reason invite.CloseReason) error {
	return n.send(playerID, "invite.closed", inviteClosedPayload{GameID: gameID, Reason: string(reason)})
}

func (n *InviteNotifier) send(playerID, msgType string, payload any) error {
	data, err := json.Marshal(envelope{Type: msgType, Payload: payload})
	if err != nil {
		return fmt.Errorf("marshaling %s envelope: %w", msgType, err)
	}
	return n.hub.SendToUser(playerID, data)
}
