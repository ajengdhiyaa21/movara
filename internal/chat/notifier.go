package chat

import (
	"movara/internal/models"
	"movara/internal/repository"
)

// HubNotifier resolves recipients on the server and publishes only identifier
// events. It never sends message contents through the Hub.
type HubNotifier struct {
	hub   *Hub
	users repository.Users
}

func NewHubNotifier(hub *Hub, users repository.Users) *HubNotifier {
	return &HubNotifier{hub: hub, users: users}
}

func (n *HubNotifier) PublishMessageCreated(conversationID, pmUserID, messageID uint) {
	if n == nil || n.hub == nil || n.users == nil {
		return
	}
	admins, err := n.users.ListByRole(models.RoleAdmin)
	if err == nil {
		for _, admin := range admins {
			if admin.IsActive {
				n.hub.Publish(admin.ID, Event{Type: "chat.message.created", ConversationID: conversationID, MessageID: messageID})
			}
		}
	}
	pm, err := n.users.ByID(pmUserID)
	if err == nil && pm.IsActive && pm.Role == models.RolePM {
		n.hub.Publish(pm.ID, Event{Type: "chat.message.created", ConversationID: conversationID, MessageID: messageID})
	}
}
