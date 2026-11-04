package middleware

import (
	"github.com/gin-gonic/gin"
	"movara/internal/models"
	"net/http/httptest"
	"testing"
)

func TestRequireSuperAdminOnlyAdminA(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name string
		user *models.User
		want int
	}{
		{"admin.a", &models.User{Username: "admin.a", Role: models.RoleAdmin}, 204},
		{"admin.b", &models.User{Username: "admin.b", Role: models.RoleAdmin}, 403},
		{"manager", &models.User{Username: "manager", Role: models.RoleManager}, 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			r.Use(func(c *gin.Context) { c.Set(ctxUserKey, tc.user); c.Next() }, RequireSuperAdmin())
			r.GET("/x", func(c *gin.Context) { c.Status(204) })
			res := httptest.NewRecorder()
			r.ServeHTTP(res, httptest.NewRequest("GET", "/x", nil))
			if res.Code != tc.want {
				t.Fatalf("got %d want %d", res.Code, tc.want)
			}
		})
	}
}
