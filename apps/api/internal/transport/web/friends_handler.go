package web

import (
	"context"
	"encoding/json"
	"net/http"

	"api/internal/auth"
	"api/internal/domain"
	"api/internal/friends"
)

// FriendsHandler serves /api/friends/*. Every route is behind
// auth.RequireAuth, and "me" always comes from the session - never from
// the path or body - so a caller can only act on their own relationships.
type FriendsHandler struct {
	friendService *friends.FriendService
}

func NewFriendsHandler(friendService *friends.FriendService) *FriendsHandler {
	return &FriendsHandler{friendService: friendService}
}

type FriendResponse struct {
	UserID   string `json:"userId"`
	Username string `json:"username"`
	Rating   int32  `json:"rating"`
	SinceMs  int64  `json:"sinceMs"`
}

type FriendRequestResponse struct {
	UserID      string `json:"userId"`
	Username    string `json:"username"`
	Rating      int32  `json:"rating"`
	CreatedAtMs int64  `json:"createdAtMs"`
}

type BlockedUserResponse struct {
	UserID      string `json:"userId"`
	Username    string `json:"username"`
	BlockedAtMs int64  `json:"blockedAtMs"`
}

type UserSearchResultResponse struct {
	UserID       string              `json:"userId"`
	Username     string              `json:"username"`
	Rating       int32               `json:"rating"`
	Relationship domain.Relationship `json:"relationship"`
}

type FriendsListResponse struct {
	Friends []FriendResponse `json:"friends"`
}

type FriendRequestsResponse struct {
	Incoming []FriendRequestResponse `json:"incoming"`
	Outgoing []FriendRequestResponse `json:"outgoing"`
}

type BlockedListResponse struct {
	Blocked []BlockedUserResponse `json:"blocked"`
}

type UserSearchResponse struct {
	Results []UserSearchResultResponse `json:"results"`
}

type SendFriendRequestRequest struct {
	UserID string `json:"userId"`
}

type SendFriendRequestResponse struct {
	Status domain.FriendshipStatus `json:"status"`
}

func (h *FriendsHandler) ListFriendsHandler(w http.ResponseWriter, r *http.Request) {
	meID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	found, err := h.friendService.ListFriends(r.Context(), meID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	resp := FriendsListResponse{Friends: make([]FriendResponse, 0, len(found))}
	for _, f := range found {
		resp.Friends = append(resp.Friends, FriendResponse{
			UserID: f.UserID, Username: f.Username, Rating: f.Rating, SinceMs: f.Since.UnixMilli(),
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *FriendsHandler) RemoveFriendHandler(w http.ResponseWriter, r *http.Request) {
	h.withTarget(w, r, h.friendService.RemoveFriend)
}

func (h *FriendsHandler) ListRequestsHandler(w http.ResponseWriter, r *http.Request) {
	meID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	reqs, err := h.friendService.ListRequests(r.Context(), meID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, FriendRequestsResponse{
		Incoming: toRequestResponses(reqs.Incoming),
		Outgoing: toRequestResponses(reqs.Outgoing),
	})
}

func (h *FriendsHandler) SendRequestHandler(w http.ResponseWriter, r *http.Request) {
	meID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	var req SendFriendRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UserID == "" {
		http.Error(w, "userId is required", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	status, err := h.friendService.SendRequest(r.Context(), meID, req.UserID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, SendFriendRequestResponse{Status: status})
}

func (h *FriendsHandler) AcceptRequestHandler(w http.ResponseWriter, r *http.Request) {
	h.withTarget(w, r, h.friendService.AcceptRequest)
}

func (h *FriendsHandler) DeleteRequestHandler(w http.ResponseWriter, r *http.Request) {
	h.withTarget(w, r, h.friendService.DeleteRequest)
}

func (h *FriendsHandler) ListBlockedHandler(w http.ResponseWriter, r *http.Request) {
	meID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	blocked, err := h.friendService.ListBlocked(r.Context(), meID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	resp := BlockedListResponse{Blocked: make([]BlockedUserResponse, 0, len(blocked))}
	for _, b := range blocked {
		resp.Blocked = append(resp.Blocked, BlockedUserResponse{
			UserID: b.UserID, Username: b.Username, BlockedAtMs: b.BlockedAt.UnixMilli(),
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *FriendsHandler) BlockHandler(w http.ResponseWriter, r *http.Request) {
	h.withTarget(w, r, h.friendService.Block)
}

func (h *FriendsHandler) UnblockHandler(w http.ResponseWriter, r *http.Request) {
	h.withTarget(w, r, h.friendService.Unblock)
}

func (h *FriendsHandler) SearchHandler(w http.ResponseWriter, r *http.Request) {
	meID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	results, err := h.friendService.Search(r.Context(), meID, r.URL.Query().Get("q"))
	if err != nil {
		writeServiceError(w, err)
		return
	}

	resp := UserSearchResponse{Results: make([]UserSearchResultResponse, 0, len(results))}
	for _, res := range results {
		resp.Results = append(resp.Results, UserSearchResultResponse{
			UserID: res.UserID, Username: res.Username, Rating: res.Rating, Relationship: res.Relationship,
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

// withTarget runs a (me, {userId}) action that has no response body.
func (h *FriendsHandler) withTarget(w http.ResponseWriter, r *http.Request, action func(ctx context.Context, meID, targetID string) error) {
	meID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	targetID := r.PathValue("userId")
	if targetID == "" {
		http.Error(w, "missing user id", http.StatusBadRequest)
		return
	}

	if err := action(r.Context(), meID, targetID); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func toRequestResponses(reqs []domain.FriendRequest) []FriendRequestResponse {
	out := make([]FriendRequestResponse, 0, len(reqs))
	for _, fr := range reqs {
		out = append(out, FriendRequestResponse{
			UserID: fr.UserID, Username: fr.Username, Rating: fr.Rating, CreatedAtMs: fr.CreatedAt.UnixMilli(),
		})
	}
	return out
}

// requireUserID reads the session's user. RequireAuth has already rejected
// requests without one, so the failure branch is defensive.
func requireUserID(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}
	return userID, ok
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}
