package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"api/internal/auth"
	"api/internal/friends"
	"api/internal/friends/friendstest"
)

var friendsTestSecret = []byte("friends-handler-test-secret")

func newFriendsHandler() *FriendsHandler {
	return NewFriendsHandler(friends.NewFriendService(friendstest.NewRepo("alice", "bob", "carol")))
}

// callAs runs handler as userID through the real RequireAuth middleware,
// so identity reaches the handler the same way it does in production.
func callAs(t *testing.T, userID string, handler http.HandlerFunc, method, target, body string, pathUserID string) *httptest.ResponseRecorder {
	t.Helper()
	token, err := auth.GenerateToken(userID, friendsTestSecret, time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: token})
	if pathUserID != "" {
		req.SetPathValue("userId", pathUserID)
	}
	rec := httptest.NewRecorder()
	auth.RequireAuth(friendsTestSecret)(handler).ServeHTTP(rec, req)
	return rec
}

func sendRequest(t *testing.T, h *FriendsHandler, from, to string) *httptest.ResponseRecorder {
	t.Helper()
	return callAs(t, from, h.SendRequestHandler, http.MethodPost, "/api/friends/requests", `{"userId":"`+to+`"}`, "")
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.NewDecoder(rec.Body).Decode(&v); err != nil {
		t.Fatalf("decoding %q: %v", rec.Body.String(), err)
	}
	return v
}

func TestFriendRequestLifecycle(t *testing.T) {
	h := newFriendsHandler()

	rec := sendRequest(t, h, "alice", "bob")
	if rec.Code != http.StatusCreated {
		t.Fatalf("send: status = %d: %s", rec.Code, rec.Body.String())
	}
	if got := decode[SendFriendRequestResponse](t, rec); got.Status != "pending" {
		t.Errorf("send: status field = %q, want pending", got.Status)
	}

	rec = callAs(t, "bob", h.ListRequestsHandler, http.MethodGet, "/api/friends/requests", "", "")
	reqs := decode[FriendRequestsResponse](t, rec)
	if len(reqs.Incoming) != 1 || reqs.Incoming[0].UserID != "alice" || reqs.Incoming[0].Username != "alice" {
		t.Fatalf("bob's incoming = %+v, want alice", reqs.Incoming)
	}
	if reqs.Incoming[0].CreatedAtMs == 0 {
		t.Error("createdAtMs is zero; timestamps should cross the wire as unix ms")
	}
	if reqs.Outgoing == nil {
		t.Error("outgoing encoded as null; the client expects an array")
	}

	rec = callAs(t, "bob", h.AcceptRequestHandler, http.MethodPost, "/api/friends/requests/alice/accept", "", "alice")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("accept: status = %d: %s", rec.Code, rec.Body.String())
	}

	rec = callAs(t, "alice", h.ListFriendsHandler, http.MethodGet, "/api/friends", "", "")
	list := decode[FriendsListResponse](t, rec)
	if len(list.Friends) != 1 || list.Friends[0].UserID != "bob" || list.Friends[0].SinceMs == 0 {
		t.Fatalf("alice's friends = %+v, want bob with sinceMs", list.Friends)
	}

	rec = callAs(t, "alice", h.RemoveFriendHandler, http.MethodDelete, "/api/friends/bob", "", "bob")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("remove: status = %d: %s", rec.Code, rec.Body.String())
	}
}

func TestFriendsErrorStatuses(t *testing.T) {
	quietLogs(t)

	cases := []struct {
		name  string
		setup func(*testing.T, *FriendsHandler)
		call  func(*testing.T, *FriendsHandler) *httptest.ResponseRecorder
		want  int
	}{
		{"send to self", nil, func(t *testing.T, h *FriendsHandler) *httptest.ResponseRecorder {
			return sendRequest(t, h, "alice", "alice")
		}, http.StatusBadRequest},
		{"send to unknown", nil, func(t *testing.T, h *FriendsHandler) *httptest.ResponseRecorder {
			return sendRequest(t, h, "alice", "nobody")
		}, http.StatusNotFound},
		{"send without userId", nil, func(t *testing.T, h *FriendsHandler) *httptest.ResponseRecorder {
			return callAs(t, "alice", h.SendRequestHandler, http.MethodPost, "/api/friends/requests", `{}`, "")
		}, http.StatusBadRequest},
		{"send malformed json", nil, func(t *testing.T, h *FriendsHandler) *httptest.ResponseRecorder {
			return callAs(t, "alice", h.SendRequestHandler, http.MethodPost, "/api/friends/requests", `{`, "")
		}, http.StatusBadRequest},
		{"duplicate request", func(t *testing.T, h *FriendsHandler) { sendRequest(t, h, "alice", "bob") },
			func(t *testing.T, h *FriendsHandler) *httptest.ResponseRecorder {
				return sendRequest(t, h, "alice", "bob")
			}, http.StatusConflict},
		{"accept nonexistent", nil, func(t *testing.T, h *FriendsHandler) *httptest.ResponseRecorder {
			return callAs(t, "bob", h.AcceptRequestHandler, http.MethodPost, "/", "", "alice")
		}, http.StatusNotFound},
		{"remove non-friend", nil, func(t *testing.T, h *FriendsHandler) *httptest.ResponseRecorder {
			return callAs(t, "alice", h.RemoveFriendHandler, http.MethodDelete, "/", "", "bob")
		}, http.StatusNotFound},
		// The silent block: indistinguishable from "send to unknown".
		{"send to someone who blocked you", func(t *testing.T, h *FriendsHandler) {
			callAs(t, "bob", h.BlockHandler, http.MethodPut, "/", "", "alice")
		}, func(t *testing.T, h *FriendsHandler) *httptest.ResponseRecorder {
			return sendRequest(t, h, "alice", "bob")
		}, http.StatusNotFound},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := newFriendsHandler()
			if c.setup != nil {
				c.setup(t, h)
			}
			if rec := c.call(t, h); rec.Code != c.want {
				t.Errorf("status = %d, want %d: %s", rec.Code, c.want, rec.Body.String())
			}
		})
	}
}

func TestBlockListAndUnblock(t *testing.T) {
	h := newFriendsHandler()

	if rec := callAs(t, "alice", h.BlockHandler, http.MethodPut, "/", "", "bob"); rec.Code != http.StatusNoContent {
		t.Fatalf("block: status = %d: %s", rec.Code, rec.Body.String())
	}

	rec := callAs(t, "alice", h.ListBlockedHandler, http.MethodGet, "/api/friends/blocks", "", "")
	blocked := decode[BlockedListResponse](t, rec)
	if len(blocked.Blocked) != 1 || blocked.Blocked[0].UserID != "bob" {
		t.Fatalf("blocked = %+v, want bob", blocked.Blocked)
	}

	if rec := callAs(t, "alice", h.UnblockHandler, http.MethodDelete, "/", "", "bob"); rec.Code != http.StatusNoContent {
		t.Fatalf("unblock: status = %d: %s", rec.Code, rec.Body.String())
	}
	rec = callAs(t, "alice", h.ListBlockedHandler, http.MethodGet, "/api/friends/blocks", "", "")
	if got := decode[BlockedListResponse](t, rec); got.Blocked == nil || len(got.Blocked) != 0 {
		t.Errorf("blocked after unblock = %#v, want []", got.Blocked)
	}
}

func TestSearchHandler(t *testing.T) {
	h := newFriendsHandler()
	sendRequest(t, h, "alice", "bob")

	rec := callAs(t, "alice", h.SearchHandler, http.MethodGet, "/api/friends/search?q=B", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}
	got := decode[UserSearchResponse](t, rec)
	if len(got.Results) != 1 || got.Results[0].UserID != "bob" || got.Results[0].Relationship != "outgoing" {
		t.Errorf("results = %+v, want bob/outgoing", got.Results)
	}

	// No query is an empty list, not an error.
	rec = callAs(t, "alice", h.SearchHandler, http.MethodGet, "/api/friends/search", "", "")
	if got := decode[UserSearchResponse](t, rec); rec.Code != http.StatusOK || got.Results == nil || len(got.Results) != 0 {
		t.Errorf("empty query: status %d, results %#v; want 200 and []", rec.Code, got.Results)
	}
}
