package handler

import (
	"net/http"
	"net/url"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"movara/internal/chat"
	"movara/internal/middleware"
)

type ChatWebSocketHandler struct {
	hub     *chat.Hub
	port    string
	buffer  int
	mu      sync.Mutex
	clients map[*chat.WebSocketConnection]uint
}

func NewChatWebSocketHandler(hub *chat.Hub, port string) *ChatWebSocketHandler {
	return &ChatWebSocketHandler{hub: hub, port: port, buffer: 16, clients: make(map[*chat.WebSocketConnection]uint)}
}

func (h *ChatWebSocketHandler) Serve(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil || !user.IsActive {
		c.Status(http.StatusUnauthorized)
		return
	}
	if !originAllowed(c.GetHeader("Origin"), h.port) {
		c.Status(http.StatusForbidden)
		return
	}
	upgrader := websocket.Upgrader{ReadBufferSize: 1024, WriteBufferSize: 1024, CheckOrigin: func(r *http.Request) bool { return originAllowed(r.Header.Get("Origin"), h.port) }}
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	conn, err := chat.NewWebSocketConnection(ws, h.buffer)
	if err != nil {
		_ = ws.Close()
		return
	}
	if err := h.hub.Register(user.ID, conn); err != nil {
		_ = conn.Close()
		c.Status(http.StatusServiceUnavailable)
		return
	}

	done := make(chan struct{})
	h.mu.Lock()
	h.clients[conn] = user.ID
	h.mu.Unlock()
	go func() { conn.WriteLoop(done) }()
	conn.ReadLoop()
	close(done)
	h.hub.Unregister(user.ID, conn)
	h.mu.Lock()
	delete(h.clients, conn)
	h.mu.Unlock()
}

func originAllowed(raw, port string) bool {
	if raw == "" || port == "" {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "http" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return false
	}
	wantPort, err := strconv.Atoi(port)
	if err != nil || wantPort < 1 || wantPort > 65535 {
		return false
	}
	return (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1") && u.Port() == port
}
