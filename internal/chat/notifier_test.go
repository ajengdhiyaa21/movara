package chat

import (
	"testing"

	"movara/internal/repository/memory"
)

func TestHubNotifierRecipients(t *testing.T) {
	repos, err := memory.New()
	if err != nil {
		t.Fatal(err)
	}
	hub := NewDefaultHub()
	adminOne, adminTwo := &fakeConnection{}, &fakeConnection{}
	manager, pm := &fakeConnection{}, &fakeConnection{}
	for id, conn := range map[uint]*fakeConnection{1: adminOne, 2: adminTwo, 3: manager, 4: pm} {
		if err := hub.Register(id, conn); err != nil {
			t.Fatal(err)
		}
	}
	NewHubNotifier(hub, repos.Users).PublishMessageCreated(9, 4, 12)
	for name, conn := range map[string]*fakeConnection{"admin 1": adminOne, "admin 2": adminTwo, "pm": pm} {
		if count, _ := conn.snapshot(); count != 1 {
			t.Fatalf("%s received %d events", name, count)
		}
	}
	if count, _ := manager.snapshot(); count != 0 {
		t.Fatalf("manager received %d events", count)
	}
}
