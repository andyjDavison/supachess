package web

import (
	"context"
	"net/http"
	"strconv"

	"api/internal/chat"
	"api/internal/transport/dto"
)

// ChatHandler serves the read side of chat: conversation lists, history,
// read/archive state, and game chat logs. Sending goes over the WebSocket.
// Every route is behind auth.RequireAuth and acts as the session's user.
type ChatHandler struct {
	chatService *chat.ChatService
}

func NewChatHandler(chatService *chat.ChatService) *ChatHandler {
	return &ChatHandler{chatService: chatService}
}

type ConversationsResponse struct {
	Conversations []dto.ConversationDTO `json:"conversations"`
}

type DirectMessagesResponse struct {
	Messages []dto.DirectMessageDTO `json:"messages"`
}

type GameMessagesResponse struct {
	Messages []dto.GameMessageDTO `json:"messages"`
}

func (h *ChatHandler) ListConversationsHandler(w http.ResponseWriter, r *http.Request) {
	meID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	convs, err := h.chatService.ListConversations(r.Context(), meID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	resp := ConversationsResponse{Conversations: make([]dto.ConversationDTO, 0, len(convs))}
	for i := range convs {
		resp.Conversations = append(resp.Conversations, dto.NewConversationDTO(&convs[i]))
	}
	writeJSON(w, http.StatusOK, resp)
}

// ListMessagesHandler pages backwards: ?before=<message id> returns the
// page older than it, and no cursor returns the latest page.
func (h *ChatHandler) ListMessagesHandler(w http.ResponseWriter, r *http.Request) {
	meID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	var before int64
	if raw := r.URL.Query().Get("before"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed <= 0 {
			http.Error(w, "before must be a positive message id", http.StatusBadRequest)
			return
		}
		before = parsed
	}

	msgs, err := h.chatService.GetMessages(r.Context(), meID, r.PathValue("id"), before)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	resp := DirectMessagesResponse{Messages: make([]dto.DirectMessageDTO, 0, len(msgs))}
	for i := range msgs {
		resp.Messages = append(resp.Messages, dto.NewDirectMessageDTO(&msgs[i]))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *ChatHandler) MarkReadHandler(w http.ResponseWriter, r *http.Request) {
	h.conversationAction(w, r, h.chatService.MarkRead)
}

func (h *ChatHandler) ArchiveHandler(w http.ResponseWriter, r *http.Request) {
	h.conversationAction(w, r, h.chatService.Archive)
}

func (h *ChatHandler) UnarchiveHandler(w http.ResponseWriter, r *http.Request) {
	h.conversationAction(w, r, h.chatService.Unarchive)
}

func (h *ChatHandler) GameMessagesHandler(w http.ResponseWriter, r *http.Request) {
	meID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	msgs, err := h.chatService.GameMessages(r.Context(), meID, r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}

	resp := GameMessagesResponse{Messages: make([]dto.GameMessageDTO, 0, len(msgs))}
	for i := range msgs {
		resp.Messages = append(resp.Messages, dto.NewGameMessageDTO(&msgs[i]))
	}
	writeJSON(w, http.StatusOK, resp)
}

// conversationAction runs a (me, {id}) action that has no response body.
func (h *ChatHandler) conversationAction(w http.ResponseWriter, r *http.Request, action func(ctx context.Context, meID, convID string) error) {
	meID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	if err := action(r.Context(), meID, r.PathValue("id")); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
