package repository

import (
	"time"

	"movara/internal/models"
)

// ChatRepository is the persistence contract for the chat foundation.
// Implementations must enforce the database unique constraints as well.
type ChatRepository interface {
	ConversationByPM(pmUserID uint) (*models.ChatConversation, error)
	ConversationByID(id uint) (*models.ChatConversation, error)
	CreateConversation(conversation *models.ChatConversation) error
	MessageByClientID(conversationID uint, clientMessageID string) (*models.ChatMessage, error)
	MessageByID(id uint) (*models.ChatMessage, error)
	CreateMessage(message *models.ChatMessage) error
	ListMessages(conversationID, beforeID, afterID uint, limit int) ([]models.ChatMessage, error)
	ReadState(conversationID, userID uint) (*models.ChatReadState, error)
	SaveReadState(state *models.ChatReadState) error
	Now() time.Time
}
