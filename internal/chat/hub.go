// Package chat contains realtime chat infrastructure independent of transport.
package chat

import (
	"errors"
	"sync"
)

const (
	DefaultMaxConnectionsPerUser = 3
)

var ErrHubClosed = errors.New("chat hub sudah ditutup")

// Event is deliberately limited to identifiers. Message contents and sender
// identity are retrieved through the authorized HTTP API.
type Event struct {
	Type           string `json:"type"`
	ConversationID uint   `json:"conversation_id"`
	MessageID      uint   `json:"message_id"`
}

// Connection is the transport-neutral contract used by the hub. A WebSocket
// handler can adapt its connection to this interface in a later stage.
type Connection interface {
	// TrySend must not block. A transport adapter owns its buffered outbound
	// queue and returns false when the queue is closed or full.
	TrySend(Event) bool
	Close() error
}

type Hub struct {
	mu          sync.Mutex
	connections map[uint]map[Connection]struct{}
	maxPerUser  int
	closed      bool
}

func NewHub(maxConnectionsPerUser int) (*Hub, error) {
	if maxConnectionsPerUser <= 0 {
		return nil, errors.New("konfigurasi chat hub tidak valid")
	}
	return &Hub{
		connections: make(map[uint]map[Connection]struct{}),
		maxPerUser:  maxConnectionsPerUser,
	}, nil
}

func NewDefaultHub() *Hub {
	hub, _ := NewHub(DefaultMaxConnectionsPerUser)
	return hub
}

// Register adds a connection for an already authenticated user. The hub does
// not inspect or accept identity data from the connection itself.
func (h *Hub) Register(userID uint, conn Connection) error {
	if userID == 0 || conn == nil {
		return errors.New("koneksi chat tidak valid")
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return ErrHubClosed
	}
	connections := h.connections[userID]
	if len(connections) >= h.maxPerUser {
		return errors.New("batas koneksi chat tercapai")
	}
	if connections == nil {
		connections = make(map[Connection]struct{})
		h.connections[userID] = connections
	}
	connections[conn] = struct{}{}
	return nil
}

// Unregister removes a connection and closes it. Calling it repeatedly is safe.
func (h *Hub) Unregister(userID uint, conn Connection) {
	if conn == nil {
		return
	}
	h.mu.Lock()
	if connections := h.connections[userID]; connections != nil {
		if _, ok := connections[conn]; ok {
			delete(connections, conn)
			if len(connections) == 0 {
				delete(h.connections, userID)
			}
			_ = conn.Close()
		}
	}
	h.mu.Unlock()
}

// Publish sends only to the selected recipient. A failed send removes that
// connection; publication never returns the error to the message writer.
func (h *Hub) Publish(userID uint, event Event) {
	h.mu.Lock()
	connections := h.connections[userID]
	for conn := range connections {
		if !conn.TrySend(event) {
			delete(connections, conn)
			_ = conn.Close()
		}
	}
	if len(connections) == 0 {
		delete(h.connections, userID)
	}
	h.mu.Unlock()
}

// ConnectionCount is intended for operational checks and tests.
func (h *Hub) ConnectionCount(userID uint) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.connections[userID])
}

// Shutdown closes every registered connection and makes future registration
// fail. It is safe to call more than once.
func (h *Hub) Shutdown() {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return
	}
	h.closed = true
	connections := h.connections
	h.connections = make(map[uint]map[Connection]struct{})
	for _, userConnections := range connections {
		for conn := range userConnections {
			_ = conn.Close()
		}
	}
	h.mu.Unlock()
}
