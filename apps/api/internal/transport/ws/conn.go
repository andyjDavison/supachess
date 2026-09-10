package ws

import (
	"api/internal/domain"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var allowedOrigins map[string]bool

// ConfigureAllowedOrigins sets the set of origins the WebSocket upgrade
// will accept connections from. Call this once, in main.go, before the
// server starts accepting connections — the same origin(s) your CORS
// middleware allows, since an unrestricted CheckOrigin is a CSRF hole
// once auth is cookie-based (a browser attaches cookies to a WS upgrade
// regardless of which site initiated it).
func ConfigureAllowedOrigins(origins ...string) {
	allowedOrigins = make(map[string]bool, len(origins))
	for _, o := range origins {
		allowedOrigins[o] = true
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// TODO: restrict this to your actual frontend origin(s) before
	// deploying anywhere but localhost.
	CheckOrigin: checkOrigin,
}

func checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		// No Origin header at all — not a browser request (a Postman/
		// websocat test, a server-to-server call). Allowed through;
		// browsers always send this header for cross-origin requests, so
		// its absence isn't itself the attack this check defends against.
		return true
	}
	return allowedOrigins[origin]
}

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingInterval   = (pongWait * 9) / 10
	sendBufferSize = 32
)

// wsConn adapts a real *websocket.Conn to the Client interface Hub
// depends on. Outbound messages go through a buffered channel rather
// than being written directly wherever Send is called from — gorilla's
// Conn isn't safe for concurrent writes, and multiple goroutines (this
// connection's own read pump triggering a reply, plus any other part of
// the app calling Hub.SendToUser for this same user) could otherwise
// race on the socket. A single writePump goroutine per connection is
// what actually owns the socket for writing.
type wsConn struct {
	id     string
	conn   *websocket.Conn
	send   chan []byte
	hub    *Hub
	onText func(clientID string, message []byte)
}

// NewConnection upgrades an HTTP request to a WebSocket, registers the
// resulting client with hub under clientID (the authenticated user's ID
// — resolved by the caller before this is invoked, never trusted from
// the request itself), and starts its read/write pumps. onText is called
// for every text message received from this client.
func NewConnection(
	w http.ResponseWriter,
	r *http.Request,
	hub *Hub,
	clientID string,
	onText func(clientID string, message []byte),
) error {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}

	c := &wsConn{
		id:     clientID,
		conn:   conn,
		send:   make(chan []byte, sendBufferSize),
		hub:    hub,
		onText: onText,
	}

	hub.Register(c)
	go c.writePump()
	go c.readPump()
	return nil
}

func (c *wsConn) ID() string { return c.id }

// Send enqueues a message for delivery without blocking the caller. If
// this client's buffer is full, it's disconnected rather than letting
// one slow reader apply backpressure to whatever's trying to send to it.
func (c *wsConn) Send(message []byte) error {
	select {
	case c.send <- message:
		return nil
	default:
		c.close()
		return domain.ErrSendBufferFull
	}
}

func (c *wsConn) readPump() {
	defer c.close()

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		if c.onText != nil {
			c.onText(c.id, message)
		}
	}
}

func (c *wsConn) writePump() {
	ticker := time.NewTicker(pingInterval)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, nil)
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *wsConn) close() {
	c.hub.Unregister(c)
	if err := c.conn.Close(); err != nil {
		slog.Debug("closing websocket connection", "client", c.id, "error", err)
	}
}