package friends

import (
	"context"
	"errors"
	"strings"
	"testing"

	"api/internal/domain"
	"api/internal/friends/friendstest"
)

var ctx = context.Background()

func newService(usernames ...string) (*FriendService, *friendstest.Repo) {
	repo := friendstest.NewRepo(usernames...)
	return NewFriendService(repo), repo
}

func mustSend(t *testing.T, s *FriendService, from, to string) domain.FriendshipStatus {
	t.Helper()
	status, err := s.SendRequest(ctx, from, to)
	if err != nil {
		t.Fatalf("SendRequest(%s -> %s): %v", from, to, err)
	}
	return status
}

func friendIDs(t *testing.T, s *FriendService, of string) []string {
	t.Helper()
	friends, err := s.ListFriends(ctx, of)
	if err != nil {
		t.Fatalf("ListFriends(%s): %v", of, err)
	}
	ids := []string{}
	for _, f := range friends {
		ids = append(ids, f.UserID)
	}
	return ids
}

func requestIDs(t *testing.T, s *FriendService, of string) (incoming, outgoing []string) {
	t.Helper()
	reqs, err := s.ListRequests(ctx, of)
	if err != nil {
		t.Fatalf("ListRequests(%s): %v", of, err)
	}
	for _, r := range reqs.Incoming {
		incoming = append(incoming, r.UserID)
	}
	for _, r := range reqs.Outgoing {
		outgoing = append(outgoing, r.UserID)
	}
	return incoming, outgoing
}

func TestSendThenAcceptMakesFriends(t *testing.T) {
	s, _ := newService("alice", "bob")

	if status := mustSend(t, s, "alice", "bob"); status != domain.FriendshipPending {
		t.Fatalf("status = %q, want pending", status)
	}
	in, out := requestIDs(t, s, "bob")
	if len(in) != 1 || in[0] != "alice" || len(out) != 0 {
		t.Fatalf("bob's requests = in %v out %v, want in [alice]", in, out)
	}
	_, out = requestIDs(t, s, "alice")
	if len(out) != 1 || out[0] != "bob" {
		t.Fatalf("alice's outgoing = %v, want [bob]", out)
	}

	if err := s.AcceptRequest(ctx, "bob", "alice"); err != nil {
		t.Fatalf("AcceptRequest: %v", err)
	}

	if got := friendIDs(t, s, "alice"); len(got) != 1 || got[0] != "bob" {
		t.Errorf("alice's friends = %v, want [bob]", got)
	}
	if got := friendIDs(t, s, "bob"); len(got) != 1 || got[0] != "alice" {
		t.Errorf("bob's friends = %v, want [alice]", got)
	}
	if in, out := requestIDs(t, s, "bob"); len(in)+len(out) != 0 {
		t.Errorf("requests left after accepting: in %v out %v", in, out)
	}
}

// If bob already asked alice, alice "sending" a request is really an
// accept - there's no sensible state with requests pending both ways.
func TestSendingToSomeoneWhoAskedYouAccepts(t *testing.T) {
	s, _ := newService("alice", "bob")
	mustSend(t, s, "bob", "alice")

	if status := mustSend(t, s, "alice", "bob"); status != domain.FriendshipAccepted {
		t.Fatalf("status = %q, want accepted", status)
	}
	if got := friendIDs(t, s, "alice"); len(got) != 1 || got[0] != "bob" {
		t.Errorf("alice's friends = %v, want [bob]", got)
	}
}

func TestSendRequestRejections(t *testing.T) {
	cases := []struct {
		name  string
		setup func(*FriendService)
		from  string
		to    string
		want  error
	}{
		{"self", nil, "alice", "alice", domain.ErrCannotFriendSelf},
		{"unknown user", nil, "alice", "nobody", domain.ErrUserNotFound},
		{"already pending", func(s *FriendService) { s.SendRequest(ctx, "alice", "bob") }, "alice", "bob", domain.ErrFriendRequestExists},
		{"already friends", func(s *FriendService) {
			s.SendRequest(ctx, "alice", "bob")
			s.AcceptRequest(ctx, "bob", "alice")
		}, "alice", "bob", domain.ErrAlreadyFriends},
		// The silent block: a blocked user sees the same error as for a
		// user that doesn't exist, whichever side did the blocking.
		{"target blocked sender", func(s *FriendService) { s.Block(ctx, "bob", "alice") }, "alice", "bob", domain.ErrUserNotFound},
		{"sender blocked target", func(s *FriendService) { s.Block(ctx, "alice", "bob") }, "alice", "bob", domain.ErrUserNotFound},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, _ := newService("alice", "bob")
			if c.setup != nil {
				c.setup(s)
			}
			if _, err := s.SendRequest(ctx, c.from, c.to); !errors.Is(err, c.want) {
				t.Errorf("err = %v, want %v", err, c.want)
			}
		})
	}
}

// Only the addressee may accept. Alice accepting her own request to bob
// must not make them friends.
func TestRequesterCannotAcceptOwnRequest(t *testing.T) {
	s, _ := newService("alice", "bob")
	mustSend(t, s, "alice", "bob")

	if err := s.AcceptRequest(ctx, "alice", "bob"); !errors.Is(err, domain.ErrFriendRequestNotFound) {
		t.Fatalf("err = %v, want ErrFriendRequestNotFound", err)
	}
	if got := friendIDs(t, s, "alice"); len(got) != 0 {
		t.Errorf("alice has friends %v after accepting her own request", got)
	}
}

func TestAcceptWithoutRequestIsNotFound(t *testing.T) {
	s, _ := newService("alice", "bob", "carol")
	mustSend(t, s, "alice", "bob")

	// carol is a bystander to the alice->bob request.
	if err := s.AcceptRequest(ctx, "carol", "alice"); !errors.Is(err, domain.ErrFriendRequestNotFound) {
		t.Errorf("err = %v, want ErrFriendRequestNotFound", err)
	}
}

func TestDeleteRequestDeclinesOrCancels(t *testing.T) {
	for _, c := range []struct{ name, deleter, other string }{
		{"addressee declines", "bob", "alice"},
		{"requester cancels", "alice", "bob"},
	} {
		t.Run(c.name, func(t *testing.T) {
			s, _ := newService("alice", "bob")
			mustSend(t, s, "alice", "bob")

			if err := s.DeleteRequest(ctx, c.deleter, c.other); err != nil {
				t.Fatalf("DeleteRequest: %v", err)
			}
			if in, out := requestIDs(t, s, "bob"); len(in)+len(out) != 0 {
				t.Errorf("request survived: in %v out %v", in, out)
			}
			// Declining isn't permanent: alice can ask again.
			mustSend(t, s, "alice", "bob")
		})
	}
}

// DeleteRequest only touches pending rows, so it can't be used to
// unfriend - that's RemoveFriend's job.
func TestDeleteRequestDoesNotRemoveFriends(t *testing.T) {
	s, _ := newService("alice", "bob")
	mustSend(t, s, "alice", "bob")
	s.AcceptRequest(ctx, "bob", "alice")

	if err := s.DeleteRequest(ctx, "alice", "bob"); !errors.Is(err, domain.ErrFriendRequestNotFound) {
		t.Errorf("err = %v, want ErrFriendRequestNotFound", err)
	}
	if got := friendIDs(t, s, "alice"); len(got) != 1 {
		t.Errorf("friendship was removed by DeleteRequest: %v", got)
	}
}

func TestRemoveFriend(t *testing.T) {
	s, _ := newService("alice", "bob")
	mustSend(t, s, "alice", "bob")
	s.AcceptRequest(ctx, "bob", "alice")

	// Either side can remove; here the addressee does.
	if err := s.RemoveFriend(ctx, "bob", "alice"); err != nil {
		t.Fatalf("RemoveFriend: %v", err)
	}
	if got := friendIDs(t, s, "alice"); len(got) != 0 {
		t.Errorf("alice's friends = %v, want none", got)
	}
	if err := s.RemoveFriend(ctx, "bob", "alice"); !errors.Is(err, domain.ErrNotFriends) {
		t.Errorf("second remove: err = %v, want ErrNotFriends", err)
	}
}

func TestRemoveFriendDoesNotCancelPendingRequest(t *testing.T) {
	s, _ := newService("alice", "bob")
	mustSend(t, s, "alice", "bob")

	if err := s.RemoveFriend(ctx, "alice", "bob"); !errors.Is(err, domain.ErrNotFriends) {
		t.Errorf("err = %v, want ErrNotFriends", err)
	}
	if _, out := requestIDs(t, s, "alice"); len(out) != 1 {
		t.Errorf("pending request was removed by RemoveFriend")
	}
}

func TestBlockClearsExistingRelationship(t *testing.T) {
	for _, c := range []struct {
		name  string
		setup func(*FriendService)
	}{
		{"friends", func(s *FriendService) {
			s.SendRequest(ctx, "alice", "bob")
			s.AcceptRequest(ctx, "bob", "alice")
		}},
		{"outgoing request", func(s *FriendService) { s.SendRequest(ctx, "alice", "bob") }},
		{"incoming request", func(s *FriendService) { s.SendRequest(ctx, "bob", "alice") }},
	} {
		t.Run(c.name, func(t *testing.T) {
			s, _ := newService("alice", "bob")
			c.setup(s)

			if err := s.Block(ctx, "alice", "bob"); err != nil {
				t.Fatalf("Block: %v", err)
			}
			for _, who := range []string{"alice", "bob"} {
				if got := friendIDs(t, s, who); len(got) != 0 {
					t.Errorf("%s still has friends %v", who, got)
				}
				if in, out := requestIDs(t, s, who); len(in)+len(out) != 0 {
					t.Errorf("%s still has requests in %v out %v", who, in, out)
				}
			}
		})
	}
}

func TestBlockAndUnblock(t *testing.T) {
	s, _ := newService("alice", "bob")

	if err := s.Block(ctx, "alice", "bob"); err != nil {
		t.Fatalf("Block: %v", err)
	}
	if err := s.Block(ctx, "alice", "bob"); err != nil {
		t.Fatalf("blocking twice should be a no-op, got %v", err)
	}
	blocked, _ := s.ListBlocked(ctx, "alice")
	if len(blocked) != 1 || blocked[0].UserID != "bob" {
		t.Fatalf("alice's blocked = %v, want [bob]", blocked)
	}
	// The block list is the blocker's; bob doesn't see who blocked him.
	if theirs, _ := s.ListBlocked(ctx, "bob"); len(theirs) != 0 {
		t.Errorf("bob's blocked list = %v, want empty", theirs)
	}

	if err := s.Unblock(ctx, "alice", "bob"); err != nil {
		t.Fatalf("Unblock: %v", err)
	}
	if err := s.Unblock(ctx, "alice", "bob"); err != nil {
		t.Fatalf("unblocking twice should be a no-op, got %v", err)
	}
	mustSend(t, s, "bob", "alice")
}

// bob can't undo alice's block of him by "unblocking" her.
func TestUnblockOnlyRemovesYourOwnBlock(t *testing.T) {
	s, _ := newService("alice", "bob")
	s.Block(ctx, "alice", "bob")

	s.Unblock(ctx, "bob", "alice")

	if _, err := s.SendRequest(ctx, "bob", "alice"); !errors.Is(err, domain.ErrUserNotFound) {
		t.Errorf("err = %v, want ErrUserNotFound - alice's block should still stand", err)
	}
}

func TestBlockSelfAndUnknown(t *testing.T) {
	s, _ := newService("alice")

	if err := s.Block(ctx, "alice", "alice"); !errors.Is(err, domain.ErrCannotFriendSelf) {
		t.Errorf("block self: err = %v, want ErrCannotFriendSelf", err)
	}
	if err := s.Unblock(ctx, "alice", "alice"); !errors.Is(err, domain.ErrCannotFriendSelf) {
		t.Errorf("unblock self: err = %v, want ErrCannotFriendSelf", err)
	}
	if err := s.Block(ctx, "alice", "nobody"); !errors.Is(err, domain.ErrUserNotFound) {
		t.Errorf("block unknown: err = %v, want ErrUserNotFound", err)
	}
}

func TestSearchAnnotatesRelationship(t *testing.T) {
	s, _ := newService("me", "mfriend", "mout", "min", "mnone", "other")
	mustSend(t, s, "me", "mfriend")
	s.AcceptRequest(ctx, "mfriend", "me")
	mustSend(t, s, "me", "mout")
	mustSend(t, s, "min", "me")

	results, err := s.Search(ctx, "me", "M")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	got := map[string]domain.Relationship{}
	for _, r := range results {
		got[r.UserID] = r.Relationship
	}
	want := map[string]domain.Relationship{
		"mfriend": domain.RelationshipFriend,
		"mout":    domain.RelationshipOutgoing,
		"min":     domain.RelationshipIncoming,
		"mnone":   domain.RelationshipNone,
	}
	if len(got) != len(want) {
		t.Fatalf("results = %v, want exactly %v (no self, no non-matches)", got, want)
	}
	for id, rel := range want {
		if got[id] != rel {
			t.Errorf("%s relationship = %q, want %q", id, got[id], rel)
		}
	}
}

func TestSearchHidesBlockedEitherWay(t *testing.T) {
	s, _ := newService("alice", "bob", "bobby")
	s.Block(ctx, "bob", "alice")

	// alice was blocked by bob: she can't find him...
	results, _ := s.Search(ctx, "alice", "bob")
	if len(results) != 1 || results[0].UserID != "bobby" {
		t.Errorf("alice's search = %v, want only bobby", results)
	}
	// ...and bob doesn't see the user he blocked either.
	results, _ = s.Search(ctx, "bob", "al")
	if len(results) != 0 {
		t.Errorf("bob's search = %v, want none", results)
	}
}

func TestSearchIgnoresEmptyAndOverlongQueries(t *testing.T) {
	s, _ := newService("alice", "bob")

	for _, q := range []string{"", "   ", strings.Repeat("a", maxSearchQueryLen+1)} {
		results, err := s.Search(ctx, "alice", q)
		if err != nil {
			t.Errorf("Search(%q): %v", q, err)
		}
		if results == nil || len(results) != 0 {
			t.Errorf("Search(%q) = %#v, want an empty non-nil slice", q, results)
		}
	}
}

func TestSearchTrimsQuery(t *testing.T) {
	s, _ := newService("alice", "bob")

	results, _ := s.Search(ctx, "alice", "  bo ")
	if len(results) != 1 || results[0].UserID != "bob" {
		t.Errorf("results = %v, want [bob]", results)
	}
}
