package service

import (
	"errors"
	"strings"

	"movara/internal/models"
	"movara/internal/repository"
)

const (
	ChatMaxMessageLength  = 3000
	ChatMaxClientIDLength = 100
	ChatDefaultPageSize   = 30
	ChatMaxPageSize       = 100
)

var (
	ErrChatInvalidClientMessageID = errors.New("client message ID tidak valid")
	ErrChatMessageTooLong         = errors.New("pesan melebihi 3000 karakter")
	ErrChatMessageEmpty           = errors.New("pesan tidak boleh kosong")
	ErrChatReadMessageMismatch    = errors.New("pesan bukan bagian dari conversation")
	ErrChatReadStateBackwards     = errors.New("status baca tidak boleh mundur")
)

// ChatService contains database-independent chat rules for the foundation.
type ChatService struct {
	repo repository.ChatRepository
}

func NewChatService(repo repository.ChatRepository) *ChatService { return &ChatService{repo: repo} }

// AuthorizedChatService applies role and account-state rules before data access.
type AuthorizedChatService struct {
	chat     repository.ChatRepository
	users    repository.Users
	notifier ChatNotifier
}

type ChatNotifier interface {
	PublishMessageCreated(conversationID, pmUserID, messageID uint)
}

func NewAuthorizedChatService(chat repository.ChatRepository, users repository.Users, notifiers ...ChatNotifier) *AuthorizedChatService {
	var notifier ChatNotifier
	if len(notifiers) > 0 {
		notifier = notifiers[0]
	}
	return &AuthorizedChatService{chat: chat, users: users, notifier: notifier}
}

var (
	ErrChatUnauthorized      = errors.New("akun tidak memiliki akses chat")
	ErrChatInactiveRecipient = errors.New("penerima chat tidak aktif")
)

// EnsureConversationForUser opens the PM conversation requested by an
// authenticated user. The PM target is resolved server-side; PM callers can
// only open their own conversation.
func (s *AuthorizedChatService) EnsureConversationForUser(user *models.User, pmUserID uint) (*models.ChatConversation, error) {
	if user == nil || !user.IsActive || (user.Role != models.RoleAdmin && user.Role != models.RolePM) {
		return nil, ErrChatUnauthorized
	}
	target, err := s.users.ByID(pmUserID)
	if err != nil || target.Role != models.RolePM {
		return nil, ErrChatUnauthorized
	}
	if user.Role == models.RolePM && target.ID != user.ID {
		return nil, ErrChatUnauthorized
	}
	return NewChatService(s.chat).EnsureConversation(target.ID)
}

func (s *AuthorizedChatService) conversationAccess(user *models.User, conversationID uint) (*models.ChatConversation, error) {
	if user == nil || !user.IsActive || (user.Role != models.RoleAdmin && user.Role != models.RolePM) {
		return nil, ErrChatUnauthorized
	}
	conversation, err := s.chat.ConversationByID(conversationID)
	if err != nil {
		return nil, ErrChatUnauthorized
	}
	if user.Role == models.RolePM && conversation.PMUserID != user.ID {
		return nil, ErrChatUnauthorized
	}
	return conversation, nil
}

func (s *AuthorizedChatService) ListMessages(user *models.User, conversationID, beforeID, afterID, limit uint) ([]models.ChatMessage, error) {
	if _, err := s.conversationAccess(user, conversationID); err != nil {
		return nil, err
	}
	return NewChatService(s.chat).ListMessages(conversationID, beforeID, afterID, int(limit))
}

func (s *AuthorizedChatService) SendMessage(user *models.User, conversationID uint, clientMessageID, body string) (*models.ChatMessage, bool, error) {
	conversation, err := s.conversationAccess(user, conversationID)
	if err != nil {
		return nil, false, err
	}
	recipient, err := s.users.ByID(conversation.PMUserID)
	if err != nil || recipient.Role != models.RolePM {
		return nil, false, ErrChatUnauthorized
	}
	if user.Role == models.RoleAdmin && !recipient.IsActive {
		return nil, false, ErrChatInactiveRecipient
	}
	message, existing, err := NewChatService(s.chat).SendMessage(conversationID, user.ID, clientMessageID, body)
	if err == nil && !existing && s.notifier != nil {
		s.notifier.PublishMessageCreated(conversationID, recipient.ID, message.ID)
	}
	return message, existing, err
}

func (s *AuthorizedChatService) MarkRead(user *models.User, conversationID, messageID uint) (*models.ChatReadState, error) {
	if _, err := s.conversationAccess(user, conversationID); err != nil {
		return nil, err
	}
	return NewChatService(s.chat).MarkRead(conversationID, user.ID, messageID)
}

func ValidateChatClientMessageID(value string) error {
	if len(value) == 0 || len(value) > ChatMaxClientIDLength || strings.TrimSpace(value) != value {
		return ErrChatInvalidClientMessageID
	}
	// UUID format is intentionally validated without a new dependency.
	if len(value) != 36 {
		return ErrChatInvalidClientMessageID
	}
	for i, r := range value {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if r != '-' {
				return ErrChatInvalidClientMessageID
			}
			continue
		}
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return ErrChatInvalidClientMessageID
		}
	}
	return nil
}

func ValidateChatBody(body string) error {
	if strings.TrimSpace(body) == "" {
		return ErrChatMessageEmpty
	}
	if len([]rune(body)) > ChatMaxMessageLength {
		return ErrChatMessageTooLong
	}
	return nil
}

func (s *ChatService) EnsureConversation(pmUserID uint) (*models.ChatConversation, error) {
	if conversation, err := s.repo.ConversationByPM(pmUserID); err == nil {
		return conversation, nil
	}
	conversation := &models.ChatConversation{PMUserID: pmUserID, CreatedAt: s.repo.Now(), UpdatedAt: s.repo.Now()}
	if err := s.repo.CreateConversation(conversation); err != nil {
		// A database implementation should return/retry lookup after a unique conflict.
		if existing, lookupErr := s.repo.ConversationByPM(pmUserID); lookupErr == nil {
			return existing, nil
		}
		return nil, err
	}
	return conversation, nil
}

// SendMessage returns (message, existing, error). Existing is true for an idempotent retry.
func (s *ChatService) SendMessage(conversationID, senderUserID uint, clientMessageID, body string) (*models.ChatMessage, bool, error) {
	if err := ValidateChatClientMessageID(clientMessageID); err != nil {
		return nil, false, err
	}
	if err := ValidateChatBody(body); err != nil {
		return nil, false, err
	}
	if existing, err := s.repo.MessageByClientID(conversationID, clientMessageID); err == nil {
		if existing.Body != body || existing.SenderUserID != senderUserID {
			return nil, false, errors.New("client message ID sudah digunakan dengan payload berbeda")
		}
		return existing, true, nil
	}
	message := &models.ChatMessage{ConversationID: conversationID, SenderUserID: senderUserID, ClientMessageID: clientMessageID, Body: body, CreatedAt: s.repo.Now()}
	if err := s.repo.CreateMessage(message); err != nil {
		if existing, lookupErr := s.repo.MessageByClientID(conversationID, clientMessageID); lookupErr == nil {
			if existing.Body != body || existing.SenderUserID != senderUserID {
				return nil, false, errors.New("client message ID sudah digunakan dengan payload berbeda")
			}
			return existing, true, nil
		}
		return nil, false, err
	}
	return message, false, nil
}

func (s *ChatService) ListMessages(conversationID, beforeID, afterID uint, limit int) ([]models.ChatMessage, error) {
	if beforeID != 0 && afterID != 0 {
		return nil, errors.New("before_id dan after_id tidak dapat dipakai bersamaan")
	}
	if limit <= 0 {
		limit = ChatDefaultPageSize
	}
	if limit > ChatMaxPageSize {
		limit = ChatMaxPageSize
	}
	return s.repo.ListMessages(conversationID, beforeID, afterID, limit)
}

func (s *ChatService) MarkRead(conversationID, userID, messageID uint) (*models.ChatReadState, error) {
	message, err := s.repo.MessageByID(messageID)
	if err != nil || message.ConversationID != conversationID {
		return nil, ErrChatReadMessageMismatch
	}
	now := s.repo.Now()
	state, err := s.repo.ReadState(conversationID, userID)
	if err != nil {
		state = &models.ChatReadState{ConversationID: conversationID, UserID: userID, CreatedAt: now, UpdatedAt: now}
	} else if state.LastReadMessageID != nil && messageID < *state.LastReadMessageID {
		return nil, ErrChatReadStateBackwards
	}
	state.LastReadMessageID = &messageID
	state.LastReadAt = &now
	state.UpdatedAt = now
	if err := s.repo.SaveReadState(state); err != nil {
		return nil, err
	}
	return state, nil
}
