package chat

import (
	"errors"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	wsWriteWait = 10 * time.Second
	wsPongWait  = 60 * time.Second
	wsPingWait  = (wsPongWait * 9) / 10
	wsMaxSize   = 4096
)

// WebSocketConnection adapts a gorilla connection to the hub contract.
// Only the writer loop writes to the underlying WebSocket.
type WebSocketConnection struct {
	ws        *websocket.Conn
	outbound  chan Event
	closeOnce sync.Once
	closeErr  error
}

func NewWebSocketConnection(ws *websocket.Conn, buffer int) (*WebSocketConnection, error) {
	if ws == nil || buffer <= 0 {
		return nil, errors.New("koneksi websocket tidak valid")
	}
	return &WebSocketConnection{ws: ws, outbound: make(chan Event, buffer)}, nil
}

func (c *WebSocketConnection) TrySend(event Event) bool {
	select {
	case c.outbound <- event:
		return true
	default:
		return false
	}
}

func (c *WebSocketConnection) Close() error {
	c.closeOnce.Do(func() {
		close(c.outbound)
		c.closeErr = c.ws.Close()
	})
	return c.closeErr
}

func (c *WebSocketConnection) WriteLoop(done <-chan struct{}) {
	ticker := time.NewTicker(wsPingWait)
	defer ticker.Stop()
	for {
		select {
		case event, ok := <-c.outbound:
			if !ok {
				return
			}
			if err := c.ws.SetWriteDeadline(time.Now().Add(wsWriteWait)); err != nil {
				return
			}
			if err := c.ws.WriteJSON(event); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.ws.SetWriteDeadline(time.Now().Add(wsWriteWait)); err != nil {
				return
			}
			if err := c.ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-done:
			return
		}
	}
}

func (c *WebSocketConnection) ReadLoop() {
	c.ws.SetReadLimit(wsMaxSize)
	_ = c.ws.SetReadDeadline(time.Now().Add(wsPongWait))
	c.ws.SetPongHandler(func(string) error { return c.ws.SetReadDeadline(time.Now().Add(wsPongWait)) })
	for {
		if _, _, err := c.ws.ReadMessage(); err != nil {
			return
		}
	}
}
