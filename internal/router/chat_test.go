package router

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"movara/internal/chat"
	"movara/internal/config"
	"movara/internal/handler"
	"movara/internal/repository"
	"movara/internal/repository/memory"
)

func newChatTestEngine(t *testing.T) *gin.Engine {
	t.Helper()
	repos, err := memory.New()
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{AppEnv: "development", SessionSecret: "test-session-secret-with-at-least-32-chars"}
	engine, err := NewWithRenderer(cfg, repos, nil, handler.NewTestRenderer())
	if err != nil {
		t.Fatal(err)
	}
	return engine
}

func loginChatTestUser(t *testing.T, engine http.Handler, username string) string {
	t.Helper()
	form := url.Values{"username": {username}, "password": {memory.DemoPassword}}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("login %s: got status %d, body %s", username, rec.Code, rec.Body.String())
	}
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == "movara" {
			return cookie.String()
		}
	}
	t.Fatalf("login %s did not set session cookie", username)
	return ""
}

func chatRequest(t *testing.T, engine http.Handler, method, target, cookie string, body io.Reader, contentType string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, body)
	req.Header.Set("Cookie", cookie)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec
}

func responseCookie(rec *httptest.ResponseRecorder, fallback string) string {
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == "movara" {
			return cookie.String()
		}
	}
	return fallback
}

func TestChatRoutesAuthorizeRolesAndConversationOwnership(t *testing.T) {
	engine := newChatTestEngine(t)
	adminCookie := loginChatTestUser(t, engine, "admin.a")

	adminHistory := chatRequest(t, engine, http.MethodGet, "/chat?pm_id=4", adminCookie, nil, "")
	if adminHistory.Code != http.StatusOK {
		t.Fatalf("admin history: got status %d, body %s", adminHistory.Code, adminHistory.Body.String())
	}
	var payload struct {
		ConversationID uint   `json:"conversation_id"`
		CSRFToken      string `json:"csrf_token"`
	}
	if err := json.Unmarshal(adminHistory.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.ConversationID == 0 {
		t.Fatal("expected conversation to be created for selected PM")
	}
	if payload.CSRFToken == "" {
		t.Fatal("expected csrf token in chat response")
	}
	adminCookie = responseCookie(adminHistory, adminCookie)
	otherConversation := chatRequest(t, engine, http.MethodGet, "/chat?pm_id=5", adminCookie, nil, "")
	if otherConversation.Code != http.StatusOK {
		t.Fatalf("admin other PM history: got status %d, body %s", otherConversation.Code, otherConversation.Body.String())
	}
	var otherPayload struct {
		ConversationID uint `json:"conversation_id"`
	}
	if err := json.Unmarshal(otherConversation.Body.Bytes(), &otherPayload); err != nil {
		t.Fatal(err)
	}

	form := url.Values{
		"conversation_id":   {formatUint(payload.ConversationID)},
		"client_message_id": {"550e8400-e29b-41d4-a716-446655440010"},
		"body":              {"Pesan dari admin"},
	}
	req := httptest.NewRequest(http.MethodPost, "/chat/messages", strings.NewReader(form.Encode()))
	req.Header.Set("Cookie", adminCookie)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-CSRF-Token", payload.CSRFToken)
	sentRec := httptest.NewRecorder()
	engine.ServeHTTP(sentRec, req)
	sent := sentRec
	if sent.Code != http.StatusOK {
		t.Fatalf("admin send: got status %d, body %s", sent.Code, sent.Body.String())
	}
	unauthenticated := chatRequest(t, engine, http.MethodGet, "/chat?pm_id=4", "", nil, "")
	if unauthenticated.Code != http.StatusFound {
		t.Fatalf("unauthenticated chat: got status %d", unauthenticated.Code)
	}

	managerCookie := loginChatTestUser(t, engine, "manager")
	managerChat := chatRequest(t, engine, http.MethodGet, "/chat?conversation_id="+formatUint(payload.ConversationID), managerCookie, nil, "")
	if managerChat.Code != http.StatusForbidden {
		t.Fatalf("manager chat: got status %d, body %s", managerChat.Code, managerChat.Body.String())
	}

	pmCookie := loginChatTestUser(t, engine, "puji")
	pmOwn := chatRequest(t, engine, http.MethodGet, "/chat?conversation_id="+formatUint(payload.ConversationID), pmCookie, nil, "")
	if pmOwn.Code != http.StatusOK {
		t.Fatalf("PM own conversation: got status %d, body %s", pmOwn.Code, pmOwn.Body.String())
	}

	otherPM := chatRequest(t, engine, http.MethodGet, "/chat?conversation_id="+formatUint(otherPayload.ConversationID), pmCookie, nil, "")
	if otherPM.Code != http.StatusForbidden {
		t.Fatalf("PM other conversation: got status %d, body %s", otherPM.Code, otherPM.Body.String())
	}
}

func formatUint(value uint) string {
	return strconv.FormatUint(uint64(value), 10)
}

func websocketTestServer(t *testing.T) (*httptest.Server, *chat.Hub) {
	t.Helper()
	repos, err := memory.New()
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
	cfg := &config.Config{AppEnv: "development", Port: port, SessionSecret: "test-session-secret-with-at-least-32-chars"}
	hub := chat.NewDefaultHub()
	engine, _, err := routerWithHubForTest(cfg, repos, hub)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewUnstartedServer(engine)
	server.Listener = listener
	server.Start()
	t.Cleanup(func() { hub.Shutdown(); server.Close() })
	return server, hub
}

func routerWithHubForTest(cfg *config.Config, repos *repository.Repos, hub *chat.Hub) (*gin.Engine, *chat.Hub, error) {
	return NewWithRendererAndHub(cfg, repos, nil, handler.NewTestRenderer(), hub)
}

func dialChat(t *testing.T, server *httptest.Server, cookie, origin string) *websocket.Conn {
	t.Helper()
	header := http.Header{"Cookie": []string{cookie}, "Origin": []string{origin}}
	conn, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/chat/ws", header)
	if err != nil {
		if response != nil {
			response.Body.Close()
		}
		t.Fatal(err)
	}
	return conn
}

func loginChatTestServer(t *testing.T, server *httptest.Server, username string) string {
	t.Helper()
	form := url.Values{"username": {username}, "password": {memory.DemoPassword}}
	req, err := http.NewRequest(http.MethodPost, server.URL+"/login", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := *server.Client()
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	response, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusFound {
		t.Fatalf("login %s: got %d", username, response.StatusCode)
	}
	for _, cookie := range response.Cookies() {
		if cookie.Name == "movara" {
			return cookie.String()
		}
	}
	t.Fatalf("login %s did not set session cookie", username)
	return ""
}

func TestChatWebSocketLifecycleAndOrigin(t *testing.T) {
	server, hub := websocketTestServer(t)
	adminCookie := loginChatTestServer(t, server, "admin.a")
	port := server.Listener.Addr().(*net.TCPAddr).Port
	origin := "http://localhost:" + strconv.Itoa(port)
	admin := dialChat(t, server, adminCookie, origin)
	defer admin.Close()
	if got := hub.ConnectionCount(1); got != 1 {
		t.Fatalf("admin hub count = %d", got)
	}

	pmCookie := loginChatTestServer(t, server, "puji")
	pm := dialChat(t, server, pmCookie, "http://127.0.0.1:"+strconv.Itoa(port))
	defer pm.Close()
	if got := hub.ConnectionCount(4); got != 1 {
		t.Fatalf("PM hub count = %d", got)
	}

	managerCookie := loginChatTestServer(t, server, "manager")
	if _, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/chat/ws", http.Header{"Cookie": []string{managerCookie}, "Origin": []string{origin}}); err == nil {
		response.Body.Close()
		t.Fatal("manager handshake unexpectedly succeeded")
	}
	if _, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/chat/ws", http.Header{"Origin": []string{origin}}); err == nil {
		t.Fatal("unauthenticated handshake unexpectedly succeeded")
	}
	if _, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/chat/ws", http.Header{"Cookie": []string{adminCookie}}); err == nil {
		t.Fatal("empty origin unexpectedly succeeded")
	}
	if _, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/chat/ws", http.Header{"Cookie": []string{adminCookie}, "Origin": []string{"http://localhost:1"}}); err == nil {
		t.Fatal("wrong origin unexpectedly succeeded")
	}

	hub.Shutdown()
	if got := hub.ConnectionCount(1) + hub.ConnectionCount(4); got != 0 {
		t.Fatalf("hub count after shutdown = %d", got)
	}
}

func TestChatWebSocketConnectionLimitAndDisconnectCleanup(t *testing.T) {
	server, hub := websocketTestServer(t)
	cookie := loginChatTestServer(t, server, "admin.a")
	port := strconv.Itoa(server.Listener.Addr().(*net.TCPAddr).Port)
	connections := make([]*websocket.Conn, 0, 3)
	for i := 0; i < 3; i++ {
		connections = append(connections, dialChat(t, server, cookie, "http://localhost:"+port))
	}
	if got := hub.ConnectionCount(1); got != 3 {
		t.Fatalf("connection count = %d", got)
	}
	// The fourth handshake is closed by the handler when Hub.Register rejects it.
	if fourth, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/chat/ws", http.Header{"Cookie": []string{cookie}, "Origin": []string{"http://localhost:" + port}}); err == nil {
		_ = fourth.Close()
	}
	if got := hub.ConnectionCount(1); got != 3 {
		t.Fatalf("connection limit changed = %d", got)
	}
	_ = connections[0].Close()
	for i := 0; i < 20 && hub.ConnectionCount(1) != 2; i++ {
		time.Sleep(5 * time.Millisecond)
	}
	if got := hub.ConnectionCount(1); got != 2 {
		t.Fatalf("disconnect cleanup count = %d", got)
	}
	for _, connection := range connections[1:] {
		_ = connection.Close()
	}
}
