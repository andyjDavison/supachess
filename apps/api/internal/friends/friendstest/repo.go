package friendstest

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"api/internal/domain"
)

// Repo is an in-memory friends.FriendRepository, shared by the service and
// handler tests. It mirrors the Postgres constraints the service relies on:
// one row per unordered pair, and references to unknown users fail as
// ErrUserNotFound. Users are seeded by username, which doubles as their ID.
type Repo struct {
	mu          sync.Mutex
	users       map[string]*domain.User
	friendships map[pairKey]*domain.Friendship
	blocks      map[[2]string]time.Time
	now         time.Time
}

type pairKey struct{ lo, hi string }

func keyOf(a, b string) pairKey {
	if a > b {
		a, b = b, a
	}
	return pairKey{a, b}
}

func NewRepo(usernames ...string) *Repo {
	r := &Repo{
		users:       map[string]*domain.User{},
		friendships: map[pairKey]*domain.Friendship{},
		blocks:      map[[2]string]time.Time{},
		now:         time.Unix(1_700_000_000, 0).UTC(),
	}
	for i, name := range usernames {
		r.users[name] = &domain.User{UserID: name, Username: name, Rating: int32(1000 + i)}
	}
	return r
}

func (r *Repo) tick() time.Time {
	r.now = r.now.Add(time.Second)
	return r.now
}

func (r *Repo) FindFriendship(_ context.Context, a, b string) (*domain.Friendship, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	f, ok := r.friendships[keyOf(a, b)]
	if !ok {
		return nil, domain.ErrFriendRequestNotFound
	}
	copied := *f
	return &copied, nil
}

func (r *Repo) CreateRequest(_ context.Context, requesterID, addresseeID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.users[requesterID] == nil || r.users[addresseeID] == nil {
		return domain.ErrUserNotFound
	}
	k := keyOf(requesterID, addresseeID)
	if _, exists := r.friendships[k]; exists {
		return domain.ErrFriendRequestExists
	}
	r.friendships[k] = &domain.Friendship{
		RequesterID: requesterID, AddresseeID: addresseeID,
		Status: domain.FriendshipPending, CreatedAt: r.tick(),
	}
	return nil
}

func (r *Repo) AcceptRequest(_ context.Context, requesterID, addresseeID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	f, ok := r.friendships[keyOf(requesterID, addresseeID)]
	if !ok || f.RequesterID != requesterID || f.Status != domain.FriendshipPending {
		return domain.ErrFriendRequestNotFound
	}
	at := r.tick()
	f.Status = domain.FriendshipAccepted
	f.AcceptedAt = &at
	return nil
}

func (r *Repo) DeleteFriendship(_ context.Context, a, b string, status domain.FriendshipStatus) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := keyOf(a, b)
	f, ok := r.friendships[k]
	if !ok || f.Status != status {
		return false, nil
	}
	delete(r.friendships, k)
	return true, nil
}

func (r *Repo) ListFriends(_ context.Context, userID string) ([]domain.Friend, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	friends := []domain.Friend{}
	for _, f := range r.friendships {
		if f.Status != domain.FriendshipAccepted {
			continue
		}
		other := otherSide(f, userID)
		if other == "" {
			continue
		}
		u := r.users[other]
		friends = append(friends, domain.Friend{UserID: u.UserID, Username: u.Username, Rating: u.Rating, Since: *f.AcceptedAt})
	}
	sort.Slice(friends, func(i, j int) bool { return friends[i].Username < friends[j].Username })
	return friends, nil
}

func (r *Repo) listPending(userID string, incoming bool) []domain.FriendRequest {
	r.mu.Lock()
	defer r.mu.Unlock()
	requests := []domain.FriendRequest{}
	for _, f := range r.friendships {
		if f.Status != domain.FriendshipPending {
			continue
		}
		var other string
		switch {
		case incoming && f.AddresseeID == userID:
			other = f.RequesterID
		case !incoming && f.RequesterID == userID:
			other = f.AddresseeID
		default:
			continue
		}
		u := r.users[other]
		requests = append(requests, domain.FriendRequest{UserID: u.UserID, Username: u.Username, Rating: u.Rating, CreatedAt: f.CreatedAt})
	}
	sort.Slice(requests, func(i, j int) bool { return requests[i].CreatedAt.After(requests[j].CreatedAt) })
	return requests
}

func (r *Repo) ListIncoming(_ context.Context, userID string) ([]domain.FriendRequest, error) {
	return r.listPending(userID, true), nil
}

func (r *Repo) ListOutgoing(_ context.Context, userID string) ([]domain.FriendRequest, error) {
	return r.listPending(userID, false), nil
}

func (r *Repo) IsBlockedEitherWay(_ context.Context, a, b string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.blockedLocked(a, b), nil
}

func (r *Repo) blockedLocked(a, b string) bool {
	_, ab := r.blocks[[2]string{a, b}]
	_, ba := r.blocks[[2]string{b, a}]
	return ab || ba
}

func (r *Repo) Block(_ context.Context, blockerID, blockedID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.users[blockerID] == nil || r.users[blockedID] == nil {
		return domain.ErrUserNotFound
	}
	delete(r.friendships, keyOf(blockerID, blockedID))
	k := [2]string{blockerID, blockedID}
	if _, exists := r.blocks[k]; !exists {
		r.blocks[k] = r.tick()
	}
	return nil
}

func (r *Repo) Unblock(_ context.Context, blockerID, blockedID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.blocks, [2]string{blockerID, blockedID})
	return nil
}

func (r *Repo) ListBlocked(_ context.Context, userID string) ([]domain.BlockedUser, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	blocked := []domain.BlockedUser{}
	for k, at := range r.blocks {
		if k[0] != userID {
			continue
		}
		u := r.users[k[1]]
		blocked = append(blocked, domain.BlockedUser{UserID: u.UserID, Username: u.Username, BlockedAt: at})
	}
	sort.Slice(blocked, func(i, j int) bool { return blocked[i].Username < blocked[j].Username })
	return blocked, nil
}

func (r *Repo) SearchUsers(_ context.Context, userID, prefix string, limit int) ([]domain.UserSearchResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	results := []domain.UserSearchResult{}
	for _, u := range r.users {
		if u.UserID == userID || r.blockedLocked(userID, u.UserID) {
			continue
		}
		if !strings.HasPrefix(strings.ToLower(u.Username), strings.ToLower(prefix)) {
			continue
		}
		rel := domain.RelationshipNone
		if f, ok := r.friendships[keyOf(userID, u.UserID)]; ok {
			switch {
			case f.Status == domain.FriendshipAccepted:
				rel = domain.RelationshipFriend
			case f.RequesterID == userID:
				rel = domain.RelationshipOutgoing
			default:
				rel = domain.RelationshipIncoming
			}
		}
		results = append(results, domain.UserSearchResult{UserID: u.UserID, Username: u.Username, Rating: u.Rating, Relationship: rel})
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Username < results[j].Username })
	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

func otherSide(f *domain.Friendship, userID string) string {
	switch userID {
	case f.RequesterID:
		return f.AddresseeID
	case f.AddresseeID:
		return f.RequesterID
	}
	return ""
}
