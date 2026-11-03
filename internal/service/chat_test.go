package service

import (
	"errors"
	"strings"
	"testing"

	"movara/internal/models"
	"movara/internal/repository"
	"movara/internal/repository/memory"
)

type recordingNotifier struct {
	calls []struct{ conversationID, pmUserID, messageID uint }
}

func (n *recordingNotifier) PublishMessageCreated(conversationID, pmUserID, messageID uint) {
	n.calls = append(n.calls, struct{ conversationID, pmUserID, messageID uint }{conversationID, pmUserID, messageID})
}

type failingChatRepo struct{ repository.ChatRepository }

func (f failingChatRepo) CreateMessage(*models.ChatMessage) error {
	return errors.New("storage failure")
}

const testUUID = "550e8400-e29b-41d4-a716-446655440000"

func newChatTestService(t *testing.T) *ChatService {
	t.Helper()
	repos, err := memory.New()
	if err != nil {
		t.Fatal(err)
	}
	return NewChatService(repos.Chat)
}

func TestValidateChatClientMessageID(t *testing.T) {
	if err := ValidateChatClientMessageID(testUUID); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"", "not-a-uuid", strings.Repeat("a", 101), "550e8400-e29b-41d4-a716-44665544000z"} {
		if err := ValidateChatClientMessageID(value); err == nil {
			t.Fatalf("expected invalid UUID %q", value)
		}
	}
}

func TestChatMessageLengthAndIdempotency(t *testing.T) {
	svc := newChatTestService(t)
	conversation, err := svc.EnsureConversation(4)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.SendMessage(conversation.ID, 1, testUUID, strings.Repeat("x", ChatMaxMessageLength+1)); err != ErrChatMessageTooLong {
		t.Fatalf("expected length error, got %v", err)
	}
	message, existing, err := svc.SendMessage(conversation.ID, 1, testUUID, "halo")
	if err != nil || existing || message.ID == 0 {
		t.Fatalf("first message: %#v, %v, %v", message, existing, err)
	}
	retry, existing, err := svc.SendMessage(conversation.ID, 1, testUUID, "halo")
	if err != nil || !existing || retry.ID != message.ID {
		t.Fatalf("retry: %#v, %v, %v", retry, existing, err)
	}
	if _, _, err := svc.SendMessage(conversation.ID, 1, testUUID, "isi berbeda"); err == nil {
		t.Fatal("expected idempotency conflict")
	}
}

func TestAuthorizedChatPublishesOnlyForNewMessage(t *testing.T) {
	repos, err := memory.New()
	if err != nil {
		t.Fatal(err)
	}
	notifier := &recordingNotifier{}
	svc := NewAuthorizedChatService(repos.Chat, repos.Users, notifier)
	conversation, err := NewChatService(repos.Chat).EnsureConversation(4)
	if err != nil {
		t.Fatal(err)
	}
	admin, _ := repos.Users.ByID(1)
	if _, existing, err := svc.SendMessage(admin, conversation.ID, testUUID, "halo"); err != nil || existing {
		t.Fatalf("new message: existing=%v err=%v", existing, err)
	}
	if len(notifier.calls) != 1 || notifier.calls[0].conversationID != conversation.ID || notifier.calls[0].pmUserID != 4 {
		t.Fatalf("unexpected publish calls: %#v", notifier.calls)
	}
	if _, existing, err := svc.SendMessage(admin, conversation.ID, testUUID, "halo"); err != nil || !existing {
		t.Fatalf("retry: existing=%v err=%v", existing, err)
	}
	if len(notifier.calls) != 1 {
		t.Fatalf("retry published duplicate event: %#v", notifier.calls)
	}
}

func TestAuthorizedChatDoesNotPublishWhenStorageFails(t *testing.T) {
	repos, err := memory.New()
	if err != nil {
		t.Fatal(err)
	}
	notifier := &recordingNotifier{}
	conversation, err := NewChatService(repos.Chat).EnsureConversation(4)
	if err != nil {
		t.Fatal(err)
	}
	admin, _ := repos.Users.ByID(1)
	svc := NewAuthorizedChatService(failingChatRepo{ChatRepository: repos.Chat}, repos.Users, notifier)
	if _, _, err := svc.SendMessage(admin, conversation.ID, testUUID, "halo"); err == nil {
		t.Fatal("expected storage error")
	}
	if len(notifier.calls) != 0 {
		t.Fatalf("storage failure published event: %#v", notifier.calls)
	}
}

func TestChatConversationUniquePerPM(t *testing.T) {
	svc := newChatTestService(t)
	a, err := svc.EnsureConversation(4)
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.EnsureConversation(4)
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != b.ID {
		t.Fatalf("expected same conversation, got %d and %d", a.ID, b.ID)
	}
}

func TestChatPaginationAndReadState(t *testing.T) {
	svc := newChatTestService(t)
	conversation, err := svc.EnsureConversation(4)
	if err != nil {
		t.Fatal(err)
	}
	ids := []string{
		"550e8400-e29b-41d4-a716-446655440001",
		"550e8400-e29b-41d4-a716-446655440002",
		"550e8400-e29b-41d4-a716-446655440003",
		"550e8400-e29b-41d4-a716-446655440004",
		"550e8400-e29b-41d4-a716-446655440005",
	}
	for i := 0; i < 5; i++ {
		if _, _, err := svc.SendMessage(conversation.ID, 1, ids[i], "pesan"); err != nil {
			t.Fatal(err)
		}
	}
	page, err := svc.ListMessages(conversation.ID, 0, 0, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(page) != 2 || page[0].ID != 4 || page[1].ID != 5 {
		t.Fatalf("latest page: %#v", page)
	}
	older, err := svc.ListMessages(conversation.ID, page[0].ID, 0, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(older) != 2 || older[0].ID != 2 || older[1].ID != 3 {
		t.Fatalf("older page: %#v", older)
	}
	state, err := svc.MarkRead(conversation.ID, 2, 3)
	if err != nil {
		t.Fatal(err)
	}
	if state.LastReadMessageID == nil || *state.LastReadMessageID != 3 {
		t.Fatalf("unexpected state: %#v", state)
	}
	if _, err := svc.MarkRead(conversation.ID, 2, 2); err != ErrChatReadStateBackwards {
		t.Fatalf("expected backwards error, got %v", err)
	}
	other, err := svc.EnsureConversation(5)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.MarkRead(other.ID, 2, 1); err != ErrChatReadMessageMismatch {
		t.Fatalf("expected conversation mismatch, got %v", err)
	}
}
