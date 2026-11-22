package middleware

import (
	"github.com/gin-gonic/gin"
	"movara/internal/models"
	"net/http/httptest"
	"testing"
)

func TestRequireSuperAdminAccounts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name string
		user *models.User
		want int
	}{
		{"admin.a", &models.User{Username: "admin.a", Role: models.RoleSuperAdmin}, 204},
		{"admin.b", &models.User{Username: "admin.b", Role: models.RoleAdmin}, 403},
		{"admin.c", &models.User{Username: "admin.c", Role: models.RoleSuperAdmin}, 204},
		{"admin.d", &models.User{Username: "admin.d", Role: models.RoleSuperAdmin}, 204},
		{"admin.e", &models.User{Username: "admin.e", Role: models.RoleSuperAdmin}, 204},
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
