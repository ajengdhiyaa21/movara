package middleware

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"movara/internal/models"
)

type outletFixture struct {
	rows []models.Outlet
	err  error
}

func (f outletFixture) List() ([]models.Outlet, error) { return f.rows, f.err }

func runRequireOutlet(t *testing.T, fixture outletFixture, user *models.User) int {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set(ctxUserKey, user); c.Next() })
	r.Use(RequireOutlet(fixture))
	called := false
	r.GET("/mutasi", func(c *gin.Context) { called = true; c.Status(204) })
	req := httptest.NewRequest("GET", "/mutasi", nil)
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	if called && res.Code != 204 {
		t.Fatalf("expected successful handler response, got %d", res.Code)
	}
	return res.Code
}

func TestRequireOutletRejectsInactiveAndMissingOutlet(t *testing.T) {
	id := uint(7)
	user := &models.User{Role: models.RolePM, OutletID: &id}
	if got := runRequireOutlet(t, outletFixture{rows: []models.Outlet{{ID: id, IsActive: false}}}, user); got != 403 {
		t.Fatalf("inactive outlet: expected 403, got %d", got)
	}
	if got := runRequireOutlet(t, outletFixture{rows: []models.Outlet{{ID: 8, IsActive: true}}}, user); got != 403 {
		t.Fatalf("missing outlet: expected 403, got %d", got)
	}
	if got := runRequireOutlet(t, outletFixture{err: errors.New("repository unavailable")}, user); got != 403 {
		t.Fatalf("repository error: expected 403, got %d", got)
	}
}

func TestRequireOutletAllowsActiveOutlet(t *testing.T) {
	id := uint(7)
	user := &models.User{Role: models.RolePM, OutletID: &id}
	if got := runRequireOutlet(t, outletFixture{rows: []models.Outlet{{ID: id, IsActive: true}}}, user); got != 204 {
		t.Fatalf("active outlet: expected 204, got %d", got)
	}
}
