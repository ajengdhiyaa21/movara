package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"movara/internal/middleware"
	"movara/internal/models"
	"movara/internal/service"
)


type ChatHandler struct {
	chat *service.AuthorizedChatService
}

func NewChatHandler(chat *service.AuthorizedChatService) *ChatHandler {
	return &ChatHandler{chat: chat}
}

// Index returns the current conversation and its messages. Admins select a PM
// with pm_id; PMs always use their own authenticated user ID.
func (h *ChatHandler) Index(c *gin.Context) {
	u := middleware.CurrentUser(c)
	conversationID, err := h.conversationID(c, u)
	if err != nil {
		h.fail(c, err)
		return
	}
	beforeID, err := queryUint(c, "before_id")
	if err != nil {
		h.fail(c, err)
		return
	}
	afterID, err := queryUint(c, "after_id")
	if err != nil {
		h.fail(c, err)
		return
	}
	limit, err := queryUint(c, "limit")
	if err != nil {
		h.fail(c, err)
		return
	}
	messages, err := h.chat.ListMessages(u, conversationID, beforeID, afterID, limit)
	if err != nil {
		h.fail(c, err)
		return
	}
	csrfToken, err := middleware.CSRFToken(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal menyiapkan keamanan request"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"conversation_id": conversationID, "messages": messages, "csrf_token": csrfToken})
}

type sendMessageRequest struct {
	ConversationID  uint   `json:"conversation_id" form:"conversation_id" binding:"required"`
	ClientMessageID string `json:"client_message_id" form:"client_message_id" binding:"required"`
	Body            string `json:"body" form:"body" binding:"required"`
}

func (h *ChatHandler) SendMessage(c *gin.Context) {
	u := middleware.CurrentUser(c)
	var req sendMessageRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "format pesan tidak valid"})
		return
	}
	message, existing, err := h.chat.SendMessage(u, req.ConversationID, req.ClientMessageID, req.Body)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": message, "existing": existing})
}

type readStateRequest struct {
	ConversationID uint `json:"conversation_id" form:"conversation_id" binding:"required"`
	MessageID      uint `json:"message_id" form:"message_id" binding:"required"`
}

func (h *ChatHandler) MarkRead(c *gin.Context) {
	u := middleware.CurrentUser(c)
	var req readStateRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "format status baca tidak valid"})
		return
	}
	state, err := h.chat.MarkRead(u, req.ConversationID, req.MessageID)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"read_state": state})
}

func (h *ChatHandler) conversationID(c *gin.Context, u *models.User) (uint, error) {
	if u == nil {
		return 0, service.ErrChatUnauthorized
	}
	if id, err := queryUint(c, "conversation_id"); err != nil {
		return 0, err
	} else if id != 0 {
		return id, nil
	}
	pmID, err := queryUint(c, "pm_id")
	if err != nil {
		return 0, err
	}
	if u.Role == models.RolePM {
		pmID = u.ID
	}
	if pmID == 0 {
		return 0, errors.New("pm_id atau conversation_id wajib diisi")
	}
	conversation, err := h.chat.EnsureConversationForUser(u, pmID)
	if err != nil {
		return 0, err
	}
	return conversation.ID, nil
}

func queryUint(c *gin.Context, name string) (uint, error) {
	value := c.Query(name)
	if value == "" {
		return 0, nil
	}
	n, err := strconv.ParseUint(value, 10, 64)
	if err != nil || n > uint64(^uint(0)) {
		return 0, errors.New("parameter " + name + " tidak valid")
	}
	return uint(n), nil
}

func (h *ChatHandler) fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrChatUnauthorized):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrChatInactiveRecipient):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrChatReadMessageMismatch), errors.Is(err, service.ErrChatReadStateBackwards), errors.Is(err, service.ErrChatMessageEmpty), errors.Is(err, service.ErrChatMessageTooLong), errors.Is(err, service.ErrChatInvalidClientMessageID):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal memproses chat"})
	}
}
