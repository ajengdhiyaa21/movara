package middleware

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func csrfTestRouter() *gin.Engine {
	r := gin.New()
	r.Use(sessions.Sessions("movara", cookie.NewStore([]byte("csrf-test-session-secret-with-32"))))
	r.GET("/token", func(c *gin.Context) {
		token, err := CSRFToken(c)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.String(http.StatusOK, token)
	})
	r.POST("/protected", RequireCSRF(), func(c *gin.Context) { c.Status(http.StatusNoContent) })
	return r
}

func csrfSession(t *testing.T, r http.Handler) (string, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/token", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("token request: got %d", rec.Code)
	}
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == "movara" {
			return cookie.String(), rec.Body.String()
		}
	}
	t.Fatal("session cookie not set")
	return "", ""
}

func TestRequireCSRFRejectsMissingInvalidAndUnauthenticatedToken(t *testing.T) {
	r := csrfTestRouter()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/protected", nil)
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("missing session: got %d", rec.Code)
	}

	cookie, _ := csrfSession(t, r)
	for _, token := range []string{"", "wrong-token"} {
		req = httptest.NewRequest(http.MethodPost, "/protected", nil)
		req.Header.Set("Cookie", cookie)
		req.Header.Set("X-CSRF-Token", token)
		rec = httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("token %q: got %d", token, rec.Code)
		}
	}
}

func TestRequireCSRFAcceptsHeaderAndFormToken(t *testing.T) {
	r := csrfTestRouter()
	cookie, token := csrfSession(t, r)
	for _, setup := range []func(*http.Request){
		func(req *http.Request) { req.Header.Set("X-CSRF-Token", token) },
		func(req *http.Request) {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Body = http.NoBody
		},
	} {
		req := httptest.NewRequest(http.MethodPost, "/protected", nil)
		req.Header.Set("Cookie", cookie)
		setup(req)
		if req.Header.Get("X-CSRF-Token") == "" {
			form := url.Values{"_csrf": {token}}
			req = httptest.NewRequest(http.MethodPost, "/protected", strings.NewReader(form.Encode()))
			req.Header.Set("Cookie", cookie)
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("valid token: got %d", rec.Code)
		}
	}
}

func TestRequireCSRFRejectsCrossOriginEvenWithValidToken(t *testing.T) {
	r := gin.New()
	r.Use(sessions.Sessions("movara", cookie.NewStore([]byte("csrf-test-session-secret-with-32"))), SameOrigin())
	r.GET("/token", func(c *gin.Context) {
		token, err := CSRFToken(c)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.String(http.StatusOK, token)
	})
	r.POST("/protected", RequireCSRF(), func(c *gin.Context) { c.Status(http.StatusNoContent) })
	cookie, token := csrfSession(t, r)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "http://app.local/protected", nil)
	req.Header.Set("Cookie", cookie)
	req.Header.Set("Origin", "https://evil.example")
	req.Header.Set("X-CSRF-Token", token)
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("cross origin: got %d", rec.Code)
	}
}
