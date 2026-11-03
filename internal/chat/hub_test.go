package chat

import (
	"errors"
	"sync"
	"testing"
)

type fakeConnection struct {
	mu       sync.Mutex
	events   []Event
	closed   bool
	failSend bool
}

func (f *fakeConnection) TrySend(event Event) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failSend || f.closed {
		return false
	}
	f.events = append(f.events, event)
	return true
}

func (f *fakeConnection) Close() error {
	f.mu.Lock()
	f.closed = true
	f.mu.Unlock()
	return nil
}

func (f *fakeConnection) snapshot() (int, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.events), f.closed
}

func newTestHub(t *testing.T) *Hub {
	t.Helper()
	hub, err := NewHub(2)
	if err != nil {
		t.Fatal(err)
	}
	return hub
}

func TestHubRegisterUnregisterAndMultipleConnections(t *testing.T) {
	hub := newTestHub(t)
	first, second := &fakeConnection{}, &fakeConnection{}
	if err := hub.Register(10, first); err != nil {
		t.Fatal(err)
	}
	if err := hub.Register(10, second); err != nil {
		t.Fatal(err)
	}
	if got := hub.ConnectionCount(10); got != 2 {
		t.Fatalf("count = %d", got)
	}
	hub.Unregister(10, first)
	if got := hub.ConnectionCount(10); got != 1 {
		t.Fatalf("count after unregister = %d", got)
	}
	_, closed := first.snapshot()
	if !closed {
		t.Fatal("unregistered connection was not closed")
	}
}

func TestHubPublishOnlyToRecipient(t *testing.T) {
	hub := newTestHub(t)
	recipient, other := &fakeConnection{}, &fakeConnection{}
	_ = hub.Register(10, recipient)
	_ = hub.Register(11, other)
	hub.Publish(10, Event{Type: "chat.message.created", ConversationID: 4, MessageID: 8})
	got, _ := recipient.snapshot()
	if got != 1 {
		t.Fatalf("recipient events = %d", got)
	}
	got, _ = other.snapshot()
	if got != 0 {
		t.Fatalf("other user events = %d", got)
	}
}

func TestHubRemovesFailedConnection(t *testing.T) {
	hub := newTestHub(t)
	conn := &fakeConnection{failSend: true}
	_ = hub.Register(10, conn)
	hub.Publish(10, Event{Type: "chat.message.created", MessageID: 1})
	if got := hub.ConnectionCount(10); got != 0 {
		t.Fatalf("failed connection count = %d", got)
	}
	_, closed := conn.snapshot()
	if !closed {
		t.Fatal("failed connection was not closed")
	}
}

func TestHubConnectionLimit(t *testing.T) {
	hub := newTestHub(t)
	_ = hub.Register(10, &fakeConnection{})
	_ = hub.Register(10, &fakeConnection{})
	if err := hub.Register(10, &fakeConnection{}); err == nil {
		t.Fatal("expected connection limit error")
	}
}

func TestHubShutdown(t *testing.T) {
	hub := newTestHub(t)
	first, second := &fakeConnection{}, &fakeConnection{}
	_ = hub.Register(10, first)
	_ = hub.Register(11, second)
	hub.Shutdown()
	if got := hub.ConnectionCount(10) + hub.ConnectionCount(11); got != 0 {
		t.Fatalf("connections after shutdown = %d", got)
	}
	if err := hub.Register(10, &fakeConnection{}); !errors.Is(err, ErrHubClosed) {
		t.Fatalf("register after shutdown: %v", err)
	}
	_, firstClosed := first.snapshot()
	_, secondClosed := second.snapshot()
	if !firstClosed || !secondClosed {
		t.Fatal("shutdown did not close all connections")
	}
	hub.Shutdown()
}

func TestHubPublishConcurrentWithUnregister(t *testing.T) {
	hub := newTestHub(t)
	conn := &fakeConnection{}
	_ = hub.Register(10, conn)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			hub.Publish(10, Event{Type: "chat.message.created", MessageID: 1})
		}()
	}
	wg.Add(1)
	go func() { defer wg.Done(); hub.Unregister(10, conn) }()
	wg.Wait()
	if got := hub.ConnectionCount(10); got != 0 {
		t.Fatalf("count after concurrent unregister = %d", got)
	}
}
