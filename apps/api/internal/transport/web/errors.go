package web

import (
	"errors"
	"log/slog"
	"net/http"

	"api/internal/domain"
)

// writeServiceError maps a domain.Err* sentinel onto an HTTP status - the
// HTTP counterpart of ws.sendServiceError, which the WebSocket handler's
// comments have referred to as "writeServiceError in your HTTP handlers"
// since before it existed.
//
// Its absence was a live bug: LoginHandler collapsed every error from
// VerifyCredentials into a 500, so a wrong password answered "internal
// error" instead of 401. Routing service errors through one mapper keeps
// each handler from having to get that right on its own.
//
// Unrecognised errors still become a 500, but are logged rather than
// discarded - an unmapped sentinel should be findable, not silent.
func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidCredentials):
		// Deliberately identical for an unknown email and a bad password:
		// distinguishing them tells a caller which accounts exist.
		http.Error(w, "invalid email or password", http.StatusUnauthorized)
	case errors.Is(err, domain.ErrUserNotFound), errors.Is(err, domain.ErrGameNotFound):
		http.Error(w, "not found", http.StatusNotFound)
	case errors.Is(err, domain.ErrFriendRequestNotFound), errors.Is(err, domain.ErrNotFriends),
		errors.Is(err, domain.ErrConversationNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, domain.ErrInvalidMessageBody):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, domain.ErrRateLimited):
		http.Error(w, err.Error(), http.StatusTooManyRequests)
	case errors.Is(err, domain.ErrUsernameTaken), errors.Is(err, domain.ErrEmailTaken),
		errors.Is(err, domain.ErrAlreadyFriends), errors.Is(err, domain.ErrFriendRequestExists):
		http.Error(w, err.Error(), http.StatusConflict)
	case errors.Is(err, domain.ErrCannotFriendSelf):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		slog.Error("unhandled service error", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}
