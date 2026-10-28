package friends

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"api/internal/domain"
)

const (
	searchLimit       = 20
	maxSearchQueryLen = 32
)

type FriendService struct {
	repo FriendRepository
}

func NewFriendService(repo FriendRepository) *FriendService {
	return &FriendService{repo: repo}
}

type Requests struct {
	Incoming []domain.FriendRequest
	Outgoing []domain.FriendRequest
}

// SendRequest asks targetID to be meID's friend, returning the status the
// pair ends up in. If targetID already asked meID, this accepts that
// request instead of creating a second one.
//
// A block in either direction answers ErrUserNotFound, exactly like a user
// that doesn't exist, so a blocked user can't learn they've been blocked.
func (s *FriendService) SendRequest(ctx context.Context, meID, targetID string) (domain.FriendshipStatus, error) {
	if meID == targetID {
		return "", domain.ErrCannotFriendSelf
	}

	blocked, err := s.repo.IsBlockedEitherWay(ctx, meID, targetID)
	if err != nil {
		return "", fmt.Errorf("checking blocks: %w", err)
	}
	if blocked {
		return "", domain.ErrUserNotFound
	}

	existing, err := s.repo.FindFriendship(ctx, meID, targetID)
	switch {
	case errors.Is(err, domain.ErrFriendRequestNotFound):
		if err := s.repo.CreateRequest(ctx, meID, targetID); err != nil {
			return "", fmt.Errorf("creating friend request: %w", err)
		}
		return domain.FriendshipPending, nil
	case err != nil:
		return "", fmt.Errorf("finding friendship: %w", err)
	}

	switch {
	case existing.Status == domain.FriendshipAccepted:
		return "", domain.ErrAlreadyFriends
	case existing.RequesterID == meID:
		return "", domain.ErrFriendRequestExists
	default:
		if err := s.repo.AcceptRequest(ctx, targetID, meID); err != nil {
			return "", fmt.Errorf("accepting reverse request: %w", err)
		}
		return domain.FriendshipAccepted, nil
	}
}

// AcceptRequest accepts requesterID's pending request to meID. Only the
// addressee can accept - the requester accepting their own request is
// ErrFriendRequestNotFound, since from their side no such request exists.
func (s *FriendService) AcceptRequest(ctx context.Context, meID, requesterID string) error {
	if err := s.repo.AcceptRequest(ctx, requesterID, meID); err != nil {
		return fmt.Errorf("accepting friend request: %w", err)
	}
	return nil
}

// DeleteRequest removes a pending request between the two users in either
// direction: declining one received, or cancelling one sent.
func (s *FriendService) DeleteRequest(ctx context.Context, meID, otherID string) error {
	deleted, err := s.repo.DeleteFriendship(ctx, meID, otherID, domain.FriendshipPending)
	if err != nil {
		return fmt.Errorf("deleting friend request: %w", err)
	}
	if !deleted {
		return domain.ErrFriendRequestNotFound
	}
	return nil
}

func (s *FriendService) RemoveFriend(ctx context.Context, meID, friendID string) error {
	deleted, err := s.repo.DeleteFriendship(ctx, meID, friendID, domain.FriendshipAccepted)
	if err != nil {
		return fmt.Errorf("removing friend: %w", err)
	}
	if !deleted {
		return domain.ErrNotFriends
	}
	return nil
}

func (s *FriendService) ListFriends(ctx context.Context, meID string) ([]domain.Friend, error) {
	friends, err := s.repo.ListFriends(ctx, meID)
	if err != nil {
		return nil, fmt.Errorf("listing friends: %w", err)
	}
	return friends, nil
}

func (s *FriendService) ListRequests(ctx context.Context, meID string) (Requests, error) {
	incoming, err := s.repo.ListIncoming(ctx, meID)
	if err != nil {
		return Requests{}, fmt.Errorf("listing incoming requests: %w", err)
	}
	outgoing, err := s.repo.ListOutgoing(ctx, meID)
	if err != nil {
		return Requests{}, fmt.Errorf("listing outgoing requests: %w", err)
	}
	return Requests{Incoming: incoming, Outgoing: outgoing}, nil
}

// Block removes any friendship or pending request between the pair and
// hides each from the other. Blocking someone already blocked is a no-op.
func (s *FriendService) Block(ctx context.Context, meID, targetID string) error {
	if meID == targetID {
		return domain.ErrCannotFriendSelf
	}
	if err := s.repo.Block(ctx, meID, targetID); err != nil {
		return fmt.Errorf("blocking user: %w", err)
	}
	return nil
}

// Unblock is idempotent: unblocking someone who isn't blocked succeeds.
func (s *FriendService) Unblock(ctx context.Context, meID, targetID string) error {
	if meID == targetID {
		return domain.ErrCannotFriendSelf
	}
	if err := s.repo.Unblock(ctx, meID, targetID); err != nil {
		return fmt.Errorf("unblocking user: %w", err)
	}
	return nil
}

func (s *FriendService) ListBlocked(ctx context.Context, meID string) ([]domain.BlockedUser, error) {
	blocked, err := s.repo.ListBlocked(ctx, meID)
	if err != nil {
		return nil, fmt.Errorf("listing blocked users: %w", err)
	}
	return blocked, nil
}

// Search finds users whose username starts with q. An empty or overlong
// query returns no results rather than an error, so a search box can call
// this on every keystroke.
func (s *FriendService) Search(ctx context.Context, meID, q string) ([]domain.UserSearchResult, error) {
	q = strings.TrimSpace(q)
	if q == "" || len(q) > maxSearchQueryLen {
		return []domain.UserSearchResult{}, nil
	}
	results, err := s.repo.SearchUsers(ctx, meID, q, searchLimit)
	if err != nil {
		return nil, fmt.Errorf("searching users: %w", err)
	}
	return results, nil
}
