package ws

import (
	"api/internal/domain"
	"sync"
)

// Hub tracks connected clients by user ID and delivers messages to a
// specific user's connection. Register/Unregister/SendToUser are called
// concurrently from many goroutines, so state lives behind a single
// RWMutex, same reasoning as before.
//
// One entry per user ID is a known simplification: a second connection
// from the same user (a second tab, another device) currently overwrites
// the first in this map rather than both receiving messages. Worth
// revisiting if/when multi-device support matters — flagging it here
// rather than solving it now.
type Hub struct {
	mu      sync.RWMutex
	clients map[string]Client // userID -> Client
}

func NewHub() *Hub {
	return &Hub{clients: make(map[string]Client)}
}

func (h *Hub) Register(c Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c.ID()] = c
}

func (h *Hub) Unregister(c Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	// Only remove if it's still the same connection — guards against a
	// stale Unregister call (from a since-replaced connection) evicting
	// a newer, legitimate one for the same user.
	if existing, ok := h.clients[c.ID()]; ok && existing == c {
		delete(h.clients, c.ID())
	}
}

// SendToUser delivers message to the given user's connection, if they're
// currently connected. Returns errUserNotConnected if not — not treated
// as a failure by callers; players are frequently not connected to
// something at any given moment (browser tab closed, hasn't opened the
// app), and that's an expected, non-exceptional case.
func (h *Hub) SendToUser(userID string, message []byte) error {
	h.mu.RLock()
	c, ok := h.clients[userID]
	h.mu.RUnlock()

	if !ok {
		return domain.ErrUserNotConnected
	}
	return c.Send(message)
}

func (h *Hub) IsConnected(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.clients[userID]
	return ok
}