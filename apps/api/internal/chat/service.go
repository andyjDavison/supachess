package chat

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"api/internal/domain"
)

const (
	conversationListLimit = 100
	messagePageSize       = 50
	gameMessageLimit      = 500

	// 5 messages in a burst, then one a second - shared across DMs and
	// game chat.
	rateCapacity     = 5
	rateRefillPerSec = 1
)

// blockChecker and gameLookup are the narrow slices of other
// packages this service calls - same pattern as matchmaking and invite.
type blockChecker interface {
	IsBlocked(ctx context.Context, a, b string) (bool, error)
}

type gameLookup interface {
	GetGame(ctx context.Context, gameID string) (*domain.Game, error)
}

// Notifier is the seam to the WebSocket layer, implemented by
// ws.ChatNotifier.
type Notifier interface {
	// NotifyDirectMessage tells userID about msg, with conv as userID's
	// own view of the conversation (folder, unread, other member).
	NotifyDirectMessage(ctx context.Context, userID string, conv *domain.ConversationSummary, msg *domain.DirectMessage) error
	NotifyGameMessage(ctx context.Context, userID string, msg *domain.GameMessage) error
}

type ChatService struct {
	repo    ChatRepository
	blocks  blockChecker
	games   gameLookup
	notify  Notifier
	limiter *RateLimiter
	now     func() time.Time
}

func NewChatService(repo ChatRepository, blocks blockChecker, games gameLookup, notify Notifier) *ChatService {
	return &ChatService{
		repo:    repo,
		blocks:  blocks,
		games:   games,
		notify:  notify,
		limiter: NewRateLimiter(rateCapacity, rateRefillPerSec),
		now:     func() time.Time { return time.Now().UTC() },
	}
}

// normalizeBody trims body and enforces the length limit, counted in
// characters rather than bytes so non-Latin text gets the same 500.
func normalizeBody(body string) (string, error) {
	body = strings.TrimSpace(body)
	if body == "" || utf8.RuneCountInString(body) > domain.MaxMessageRunes {
		return "", domain.ErrInvalidMessageBody
	}
	return body, nil
}

// SendDirect sends a DM, creating the conversation on first contact, and
// pushes it to both members - the sender too, so their other views (the
// messages panel, the full page) stay in step.
//
// A block in either direction answers ErrUserNotFound, the same silent
// block friends and invites use.
func (s *ChatService) SendDirect(ctx context.Context, meID, toUserID, body string) (*domain.DirectMessage, error) {
	body, err := normalizeBody(body)
	if err != nil {
		return nil, err
	}
	if meID == toUserID {
		return nil, domain.ErrCannotMessageSelf
	}
	if !s.limiter.Allow(meID, s.now()) {
		return nil, domain.ErrRateLimited
	}

	blocked, err := s.blocks.IsBlocked(ctx, meID, toUserID)
	if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
		return nil, fmt.Errorf("checking blocks: %w", err)
	}
	if blocked || err != nil {
		return nil, domain.ErrUserNotFound
	}

	msg, err := s.repo.AppendDirectMessage(ctx, meID, toUserID, body)
	if errors.Is(err, domain.ErrUserNotFound) {
		// Returned bare, not wrapped: the client sees err.Error(), and an
		// unknown user must read exactly like a block does.
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("storing message: %w", err)
	}

	for _, userID := range []string{meID, toUserID} {
		conv, err := s.repo.GetConversation(ctx, msg.ConversationID, userID)
		if err != nil {
			slog.Error("loading conversation for notify", "conversation", msg.ConversationID, "user", userID, "error", err)
			continue
		}
		if err := s.notify.NotifyDirectMessage(ctx, userID, conv, msg); err != nil && !errors.Is(err, domain.ErrUserNotConnected) {
			slog.Error("notifying direct message", "conversation", msg.ConversationID, "user", userID, "error", err)
		}
	}
	return msg, nil
}

func (s *ChatService) ListConversations(ctx context.Context, meID string) ([]domain.ConversationSummary, error) {
	convs, err := s.repo.ListConversations(ctx, meID, conversationListLimit)
	if err != nil {
		return nil, fmt.Errorf("listing conversations: %w", err)
	}
	return convs, nil
}

// GetMessages returns a page of convID's history, older than beforeID
// (0 for the latest). Only members can read it.
func (s *ChatService) GetMessages(ctx context.Context, meID, convID string, beforeID int64) ([]domain.DirectMessage, error) {
	if _, err := s.repo.GetConversation(ctx, convID, meID); err != nil {
		return nil, err
	}
	msgs, err := s.repo.ListDirectMessages(ctx, convID, beforeID, messagePageSize)
	if err != nil {
		return nil, fmt.Errorf("listing messages: %w", err)
	}
	return msgs, nil
}

func (s *ChatService) MarkRead(ctx context.Context, meID, convID string) error {
	return s.repo.MarkRead(ctx, convID, meID)
}

func (s *ChatService) Archive(ctx context.Context, meID, convID string) error {
	return s.repo.SetArchived(ctx, convID, meID, true)
}

func (s *ChatService) Unarchive(ctx context.Context, meID, convID string) error {
	return s.repo.SetArchived(ctx, convID, meID, false)
}

// SendGame posts to a game's chat. Only its two players may, and only
// while it's being played.
func (s *ChatService) SendGame(ctx context.Context, meID, gameID, body string) (*domain.GameMessage, error) {
	body, err := normalizeBody(body)
	if err != nil {
		return nil, err
	}
	g, err := s.playerGame(ctx, meID, gameID)
	if err != nil {
		return nil, err
	}
	if g.Status != domain.GameStatusActive {
		return nil, domain.ErrGameAlreadyFinished
	}
	if !s.limiter.Allow(meID, s.now()) {
		return nil, domain.ErrRateLimited
	}

	msg, err := s.repo.AppendGameMessage(ctx, gameID, meID, body)
	if err != nil {
		return nil, fmt.Errorf("storing game message: %w", err)
	}
	for _, userID := range []string{g.WhiteID, g.BlackID} {
		if err := s.notify.NotifyGameMessage(ctx, userID, msg); err != nil && !errors.Is(err, domain.ErrUserNotConnected) {
			slog.Error("notifying game message", "game", gameID, "user", userID, "error", err)
		}
	}
	return msg, nil
}

// GameMessages returns a game's chat log - finished games included - to
// its players only.
func (s *ChatService) GameMessages(ctx context.Context, meID, gameID string) ([]domain.GameMessage, error) {
	if _, err := s.playerGame(ctx, meID, gameID); err != nil {
		return nil, err
	}
	msgs, err := s.repo.ListGameMessages(ctx, gameID, gameMessageLimit)
	if err != nil {
		return nil, fmt.Errorf("listing game messages: %w", err)
	}
	return msgs, nil
}

// playerGame loads gameID if meID is one of its players. A non-player gets
// ErrGameNotFound, the same as for a game that doesn't exist, so the chat
// of someone else's game can't even be confirmed to exist.
func (s *ChatService) playerGame(ctx context.Context, meID, gameID string) (*domain.Game, error) {
	g, err := s.games.GetGame(ctx, gameID)
	if err != nil {
		return nil, err
	}
	if meID != g.WhiteID && meID != g.BlackID {
		return nil, domain.ErrGameNotFound
	}
	return g, nil
}
