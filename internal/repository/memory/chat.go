package memory

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"movara/internal/clock"
	"movara/internal/models"
	"movara/internal/repository"
)

type chatStore struct {
	mu                 sync.RWMutex
	conversations      map[uint]models.ChatConversation
	conversationByPM   map[uint]uint
	messages           map[uint]models.ChatMessage
	messageByClient    map[string]uint
	readStates         map[string]models.ChatReadState
	nextConversationID uint
	nextMessageID      uint
	nextReadStateID    uint
}

func newChatStore() *chatStore {
	return &chatStore{conversations: map[uint]models.ChatConversation{}, conversationByPM: map[uint]uint{}, messages: map[uint]models.ChatMessage{}, messageByClient: map[string]uint{}, readStates: map[string]models.ChatReadState{}}
}

func chatKey(conversationID, userID uint) string { return fmt.Sprintf("%d:%d", conversationID, userID) }
func clientKey(conversationID uint, clientID string) string {
	return fmt.Sprintf("%d:%s", conversationID, clientID)
}

func (s *chatStore) Now() time.Time { return clock.Now() }

func (s *chatStore) ConversationByPM(pmUserID uint) (*models.ChatConversation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.conversationByPM[pmUserID]
	if !ok {
		return nil, repository.ErrNotFound
	}
	v := s.conversations[id]
	return &v, nil
}
func (s *chatStore) ConversationByID(id uint) (*models.ChatConversation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.conversations[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return &v, nil
}
func (s *chatStore) CreateConversation(v *models.ChatConversation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.conversationByPM[v.PMUserID]; ok {
		return errors.New("duplicate conversation")
	}
	s.nextConversationID++
	v.ID = s.nextConversationID
	if v.CreatedAt.IsZero() {
		v.CreatedAt = s.Now()
	}
	if v.UpdatedAt.IsZero() {
		v.UpdatedAt = v.CreatedAt
	}
	s.conversations[v.ID] = *v
	s.conversationByPM[v.PMUserID] = v.ID
	return nil
}
func (s *chatStore) MessageByClientID(conversationID uint, clientID string) (*models.ChatMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.messageByClient[clientKey(conversationID, clientID)]
	if !ok {
		return nil, repository.ErrNotFound
	}
	v := s.messages[id]
	return &v, nil
}
func (s *chatStore) MessageByID(id uint) (*models.ChatMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.messages[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return &v, nil
}
func (s *chatStore) CreateMessage(v *models.ChatMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := clientKey(v.ConversationID, v.ClientMessageID)
	if _, ok := s.messageByClient[key]; ok {
		return errors.New("duplicate message")
	}
	s.nextMessageID++
	v.ID = s.nextMessageID
	if v.CreatedAt.IsZero() {
		v.CreatedAt = s.Now()
	}
	s.messages[v.ID] = *v
	s.messageByClient[key] = v.ID
	return nil
}
func (s *chatStore) ListMessages(conversationID, beforeID, afterID uint, limit int) ([]models.ChatMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	all := make([]models.ChatMessage, 0)
	for _, v := range s.messages {
		if v.ConversationID == conversationID && (beforeID == 0 || v.ID < beforeID) && (afterID == 0 || v.ID > afterID) {
			all = append(all, v)
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].ID < all[j].ID })
	if beforeID != 0 && len(all) > limit {
		all = all[len(all)-limit:]
	} else if afterID != 0 && len(all) > limit {
		all = all[:limit]
	} else if len(all) > limit {
		all = all[len(all)-limit:]
	}
	return all, nil
}
func (s *chatStore) ReadState(conversationID, userID uint) (*models.ChatReadState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.readStates[chatKey(conversationID, userID)]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return &v, nil
}
func (s *chatStore) SaveReadState(v *models.ChatReadState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := chatKey(v.ConversationID, v.UserID)
	if v.ID == 0 {
		s.nextReadStateID++
		v.ID = s.nextReadStateID
	}
	s.readStates[key] = *v
	return nil
}
