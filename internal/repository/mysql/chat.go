package mysql

import (
	"time"

	"gorm.io/gorm"

	"movara/internal/clock"
	"movara/internal/models"
	"movara/internal/repository"
)

func (s *Store) Now() time.Time { return clock.Now() }

func (s *Store) ConversationByPM(pmUserID uint) (*models.ChatConversation, error) {
	var v models.ChatConversation
	if err := s.db.Where("pm_user_id = ?", pmUserID).First(&v).Error; err != nil {
		return nil, repository.ErrNotFound
	}
	return &v, nil
}

func (s *Store) ConversationByID(id uint) (*models.ChatConversation, error) {
	var v models.ChatConversation
	if err := s.db.First(&v, id).Error; err != nil {
		return nil, repository.ErrNotFound
	}
	return &v, nil
}

func (s *Store) CreateConversation(v *models.ChatConversation) error {
	if v.CreatedAt.IsZero() {
		v.CreatedAt = clock.Now()
	}
	if v.UpdatedAt.IsZero() {
		v.UpdatedAt = v.CreatedAt
	}
	return s.db.Create(v).Error
}

func (s *Store) MessageByClientID(conversationID uint, clientMessageID string) (*models.ChatMessage, error) {
	var v models.ChatMessage
	if err := s.db.Where("conversation_id = ? AND client_message_id = ?", conversationID, clientMessageID).First(&v).Error; err != nil {
		return nil, repository.ErrNotFound
	}
	return &v, nil
}

func (s *Store) MessageByID(id uint) (*models.ChatMessage, error) {
	var v models.ChatMessage
	if err := s.db.First(&v, id).Error; err != nil {
		return nil, repository.ErrNotFound
	}
	return &v, nil
}

func (s *Store) CreateMessage(v *models.ChatMessage) error {
	if v.CreatedAt.IsZero() {
		v.CreatedAt = clock.Now()
	}
	return s.db.Create(v).Error
}

func (s *Store) ListMessages(conversationID, beforeID, afterID uint, limit int) ([]models.ChatMessage, error) {
	q := s.db.Where("conversation_id = ?", conversationID)
	if beforeID != 0 {
		q = q.Where("id < ?", beforeID).Order("id DESC")
	} else {
		q = q.Order("id ASC")
	}
	if afterID != 0 {
		q = q.Where("id > ?", afterID)
	}
	var rows []models.ChatMessage
	if err := q.Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	if beforeID != 0 {
		for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
			rows[i], rows[j] = rows[j], rows[i]
		}
	}
	return rows, nil
}

func (s *Store) ReadState(conversationID, userID uint) (*models.ChatReadState, error) {
	var v models.ChatReadState
	if err := s.db.Where("conversation_id = ? AND user_id = ?", conversationID, userID).First(&v).Error; err != nil {
		return nil, repository.ErrNotFound
	}
	return &v, nil
}

func (s *Store) SaveReadState(v *models.ChatReadState) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var current models.ChatReadState
		err := tx.Where("conversation_id = ? AND user_id = ?", v.ConversationID, v.UserID).First(&current).Error
		if err == nil {
			if current.LastReadMessageID != nil && (v.LastReadMessageID == nil || *v.LastReadMessageID < *current.LastReadMessageID) {
				return nil
			}
			v.ID = current.ID
		}
		if v.UpdatedAt.IsZero() {
			v.UpdatedAt = time.Now()
		}
		return tx.Save(v).Error
	})
}

var _ repository.ChatRepository = (*Store)(nil)
