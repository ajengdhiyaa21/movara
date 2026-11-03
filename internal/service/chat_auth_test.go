package service

import (
	"testing"

	"movara/internal/models"
	"movara/internal/repository"
	"movara/internal/repository/memory"
)

type overriddenUsers struct {
	repository.Users
	override *models.User
}

func (u overriddenUsers) ByID(id uint) (*models.User, error) {
	if u.override != nil && u.override.ID == id {
		v := *u.override
		return &v, nil
	}
	return u.Users.ByID(id)
}

func TestAuthorizedChatServiceRoleAndActivityRules(t *testing.T) {
	repos, err := memory.New()
	if err != nil {
		t.Fatal(err)
	}
	auth := NewAuthorizedChatService(repos.Chat, repos.Users)
	conversation, err := NewChatService(repos.Chat).EnsureConversation(4)
	if err != nil {
		t.Fatal(err)
	}
	admin, _ := repos.Users.ByID(1)
	manager, _ := repos.Users.ByID(3)
	pm, _ := repos.Users.ByID(4)
	otherPM, _ := repos.Users.ByID(5)
	if _, err := auth.ListMessages(manager, conversation.ID, 0, 0, 10); err != ErrChatUnauthorized {
		t.Fatalf("manager access: %v", err)
	}
	if _, err := auth.ListMessages(otherPM, conversation.ID, 0, 0, 10); err != ErrChatUnauthorized {
		t.Fatalf("cross-PM access: %v", err)
	}
	if _, err := auth.ListMessages(admin, conversation.ID, 0, 0, 10); err != nil {
		t.Fatalf("admin read: %v", err)
	}
	if _, _, err := auth.SendMessage(pm, conversation.ID, testUUID, "halo"); err != nil {
		t.Fatalf("PM send: %v", err)
	}
	if _, _, err := auth.SendMessage(admin, conversation.ID, "550e8400-e29b-41d4-a716-446655440001", "balasan"); err != nil {
		t.Fatalf("admin send: %v", err)
	}

	// The memory repository has no public mutation helper for users; use the
	// returned user object only to verify the service does not trust role input.
	inactive := *pm
	inactive.IsActive = false
	if _, _, err := auth.SendMessage(&inactive, conversation.ID, "550e8400-e29b-41d4-a716-446655440002", "blocked"); err != ErrChatUnauthorized {
		t.Fatalf("inactive PM send: %v", err)
	}
	badRole := &models.User{ID: 1, Role: models.RoleManager, IsActive: true}
	if _, err := auth.ListMessages(badRole, conversation.ID, 0, 0, 10); err != ErrChatUnauthorized {
		t.Fatalf("manager-shaped user: %v", err)
	}
	inactiveRecipient := *pm
	inactiveRecipient.IsActive = false
	authInactiveRecipient := NewAuthorizedChatService(repos.Chat, overriddenUsers{Users: repos.Users, override: &inactiveRecipient})
	if _, err := authInactiveRecipient.ListMessages(admin, conversation.ID, 0, 0, 10); err != nil {
		t.Fatalf("admin read inactive PM: %v", err)
	}
	if _, _, err := authInactiveRecipient.SendMessage(admin, conversation.ID, "550e8400-e29b-41d4-a716-446655440003", "blocked"); err != ErrChatInactiveRecipient {
		t.Fatalf("admin send inactive PM: %v", err)
	}
}
