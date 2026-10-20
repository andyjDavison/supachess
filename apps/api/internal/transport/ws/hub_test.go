package ws

import (
	"errors"
	"sync"
	"testing"

	"api/internal/domain"
)

// fakeClient records what the hub delivers to it.
type fakeClient struct {
	mu       sync.Mutex
	id       string
	received [][]byte
	sendErr  error
}

func (c *fakeClient) ID() string { return c.id }

func (c *fakeClient) Send(message []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.sendErr != nil {
		return c.sendErr
	}
	c.received = append(c.received, message)
	return nil
}

func (c *fakeClient) messages() [][]byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([][]byte(nil), c.received...)
}

func TestHubSendToRegisteredUser(t *testing.T) {
	h := NewHub()
	client := &fakeClient{id: "user-1"}
	h.Register(client)

	if err := h.SendToUser("user-1", []byte("hello")); err != nil {
		t.Fatalf("SendToUser: %v", err)
	}

	got := client.messages()
	if len(got) != 1 || string(got[0]) != "hello" {
		t.Errorf("client received %q, want one \"hello\"", got)
	}
}

// Not being connected is an ordinary state, not a failure - callers treat
// this error as "skip them", so it has to be this specific sentinel.
func TestHubSendToUnknownUser(t *testing.T) {
	h := NewHub()

	err := h.SendToUser("nobody", []byte("hello"))

	if !errors.Is(err, domain.ErrUserNotConnected) {
		t.Errorf("got %v, want ErrUserNotConnected", err)
	}
}

func TestHubIsConnected(t *testing.T) {
	h := NewHub()
	client := &fakeClient{id: "user-1"}

	if h.IsConnected("user-1") {
		t.Error("reported connected before registering")
	}

	h.Register(client)
	if !h.IsConnected("user-1") {
		t.Error("reported not connected after registering")
	}

	h.Unregister(client)
	if h.IsConnected("user-1") {
		t.Error("reported connected after unregistering")
	}
}

func TestHubUnregisterFiresOnDisconnect(t *testing.T) {
	h := NewHub()
	var got []string
	h.OnDisconnect = func(userID string) { got = append(got, userID) }

	client := &fakeClient{id: "user-1"}
	h.Register(client)
	h.Unregister(client)

	if len(got) != 1 || got[0] != "user-1" {
		t.Errorf("OnDisconnect saw %v, want [user-1]", got)
	}
}

func TestHubUnregisterUnknownClientDoesNotFireOnDisconnect(t *testing.T) {
	h := NewHub()
	fired := false
	h.OnDisconnect = func(string) { fired = true }

	h.Unregister(&fakeClient{id: "never-registered"})

	if fired {
		t.Error("OnDisconnect fired for a client that was never registered")
	}
}

// One entry per user id means a second connection replaces the first. When
// the stale connection later closes it must not evict the live one, nor
// report the user as disconnected - that is what the `existing == c` guard
// in Unregister is for.
func TestHubUnregisterOfAReplacedClientIsIgnored(t *testing.T) {
	h := NewHub()
	var disconnects []string
	h.OnDisconnect = func(userID string) { disconnects = append(disconnects, userID) }

	first := &fakeClient{id: "user-1"}
	second := &fakeClient{id: "user-1"}
	h.Register(first)
	h.Register(second) // second tab replaces the first

	h.Unregister(first) // the stale connection closes

	if !h.IsConnected("user-1") {
		t.Error("the live connection was evicted by the stale one closing")
	}
	if len(disconnects) != 0 {
		t.Errorf("OnDisconnect fired %v, want nothing - the user is still connected", disconnects)
	}

	if err := h.SendToUser("user-1", []byte("still here")); err != nil {
		t.Errorf("SendToUser after the stale close: %v", err)
	}
	if len(second.messages()) != 1 {
		t.Error("the live client did not receive the message")
	}
}

func TestHubSendPropagatesClientError(t *testing.T) {
	h := NewHub()
	boom := errors.New("send buffer full")
	h.Register(&fakeClient{id: "user-1", sendErr: boom})

	if err := h.SendToUser("user-1", []byte("x")); !errors.Is(err, boom) {
		t.Errorf("got %v, want the client's own error", err)
	}
}

func TestHubConcurrentAccess(t *testing.T) {
	h := NewHub()
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c := &fakeClient{id: string(rune('a' + i%26))}
			h.Register(c)
			_ = h.SendToUser(c.ID(), []byte("ping"))
			h.IsConnected(c.ID())
			h.Unregister(c)
		}(i)
	}

	wg.Wait() // -race turns any lock mistake here into a failure
}
