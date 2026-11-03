package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"

	"movara/internal/middleware"
	"movara/internal/repository/memory"
	"movara/internal/service"
)

func newChatHandlerTest(t *testing.T) (*gin.Engine, string) {
	t.Helper()
	repos, err := memory.New()
	if err != nil {
		t.Fatal(err)
	}
	r := gin.New()
	store := cookie.NewStore([]byte("test-chat-session-secret-with-32-chars"))
	r.Use(sessions.Sessions("movara", store), middleware.LoadUser(repos.Users))
	r.GET("/set/:id", func(c *gin.Context) {
		id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
		s := sessions.Default(c)
		s.Set(middleware.SessionUserKey, uint(id))
		_ = s.Save()
		c.Status(http.StatusNoContent)
	})
	chatH := NewChatHandler(service.NewAuthorizedChatService(repos.Chat, repos.Users))
	r.GET("/chat", chatH.Index)
	r.POST("/chat/messages", chatH.SendMessage)
	return r, "/set/"
}

func chatSession(t *testing.T, r http.Handler, path string) string {
	t.Helper()
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("session setup: got status %d", rec.Code)
	}
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == "movara" {
			return cookie.String()
		}
	}
	t.Fatal("session cookie not set")
	return ""
}

func TestChatHandlerUsesAuthenticatedIdentityAndAuthorization(t *testing.T) {
	r, setPath := newChatHandlerTest(t)
	adminCookie := chatSession(t, r, setPath+"1")

	req := httptest.NewRequest(http.MethodGet, "/chat?pm_id=4", nil)
	req.Header.Set("Cookie", adminCookie)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin history: got status %d, body %s", rec.Code, rec.Body.String())
	}
	var payload struct {
		ConversationID uint `json:"conversation_id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil || payload.ConversationID == 0 {
		t.Fatalf("invalid conversation response: %s", rec.Body.String())
	}
	otherAdmin := httptest.NewRequest(http.MethodGet, "/chat?pm_id=5", nil)
	otherAdmin.Header.Set("Cookie", adminCookie)
	otherAdminRec := httptest.NewRecorder()
	r.ServeHTTP(otherAdminRec, otherAdmin)
	var otherPayload struct {
		ConversationID uint `json:"conversation_id"`
	}
	if err := json.Unmarshal(otherAdminRec.Body.Bytes(), &otherPayload); err != nil || otherPayload.ConversationID == 0 {
		t.Fatalf("invalid other conversation response: %s", otherAdminRec.Body.String())
	}

	pmCookie := chatSession(t, r, setPath+"4")
	own := httptest.NewRequest(http.MethodGet, "/chat?conversation_id="+strconv.FormatUint(uint64(payload.ConversationID), 10), nil)
	own.Header.Set("Cookie", pmCookie)
	ownRec := httptest.NewRecorder()
	r.ServeHTTP(ownRec, own)
	if ownRec.Code != http.StatusOK {
		t.Fatalf("PM own conversation: got status %d", ownRec.Code)
	}

	other := httptest.NewRequest(http.MethodGet, "/chat?conversation_id="+strconv.FormatUint(uint64(otherPayload.ConversationID), 10), nil)
	other.Header.Set("Cookie", pmCookie)
	otherRec := httptest.NewRecorder()
	r.ServeHTTP(otherRec, other)
	if otherRec.Code != http.StatusForbidden {
		t.Fatalf("PM other conversation: got status %d", otherRec.Code)
	}

	managerCookie := chatSession(t, r, setPath+"3")
	manager := httptest.NewRequest(http.MethodGet, "/chat?conversation_id="+strconv.FormatUint(uint64(payload.ConversationID), 10), nil)
	manager.Header.Set("Cookie", managerCookie)
	managerRec := httptest.NewRecorder()
	r.ServeHTTP(managerRec, manager)
	if managerRec.Code != http.StatusForbidden {
		t.Fatalf("manager chat: got status %d", managerRec.Code)
	}
}

func TestChatHandlerDoesNotAcceptSenderIdentityFromForm(t *testing.T) {
	r, setPath := newChatHandlerTest(t)
	adminCookie := chatSession(t, r, setPath+"1")
	first := httptest.NewRequest(http.MethodGet, "/chat?pm_id=4", nil)
	first.Header.Set("Cookie", adminCookie)
	firstRec := httptest.NewRecorder()
	r.ServeHTTP(firstRec, first)
	var payload struct {
		ConversationID uint `json:"conversation_id"`
	}
	if err := json.Unmarshal(firstRec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	form := url.Values{
		"conversation_id":   {strconv.FormatUint(uint64(payload.ConversationID), 10)},
		"client_message_id": {"550e8400-e29b-41d4-a716-446655440020"},
		"body":              {"Pesan valid"},
		"sender_user_id":    {"4"},
	}
	req := httptest.NewRequest(http.MethodPost, "/chat/messages", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", adminCookie)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("send message: got status %d, body %s", rec.Code, rec.Body.String())
	}
}
