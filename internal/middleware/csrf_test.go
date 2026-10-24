package middleware

import (
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"testing"
)

func TestSameOriginAllowsPostWithoutBrowserOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SameOrigin())
	called := false
	r.POST("/x", func(c *gin.Context) { called = true; c.Status(204) })
	req := httptest.NewRequest("POST", "/x", nil)
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	if res.Code != 204 || !called {
		t.Fatalf("expected allowed request, got %d", res.Code)
	}
}

func TestSameOriginRejectsForeignOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SameOrigin())
	r.POST("/x", func(c *gin.Context) { t.Fatal("handler must not run") })
	req := httptest.NewRequest("POST", "http://app.local/x", nil)
	req.Header.Set("Origin", "https://evil.example")
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	if res.Code != 403 {
		t.Fatalf("expected 403, got %d", res.Code)
	}
}
