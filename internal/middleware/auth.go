package middleware

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"movara/internal/clock"
	"movara/internal/models"
	"movara/internal/repository"
)

const (
	ctxUserKey      = "currentUser"
	SessionUserKey  = "uid"
	SessionTokenKey = "session_token"
)

// CurrentUser mengembalikan user yang sedang login (nil jika belum login).
func CurrentUser(c *gin.Context) *models.User {
	if v, ok := c.Get(ctxUserKey); ok {
		if u, ok := v.(*models.User); ok {
			return u
		}
	}
	return nil
}

// LoadUser memuat user dari session pada setiap request.
// User dibaca ulang dari penyimpanan supaya akun yang dinonaktifkan atau
// dipindah outlet langsung berlaku tanpa menunggu session habis.
func LoadUser(users repository.Users) gin.HandlerFunc {
	return func(c *gin.Context) {
		s := sessions.Default(c)
		if id, ok := s.Get(SessionUserKey).(uint); ok {
			u, err := users.ByID(id)
			token, hasToken := s.Get(SessionTokenKey).(string)
			sessionValid := u != nil && hasToken && token != "" && u.ActiveSessionToken == token && u.ActiveSessionUntil != nil && u.ActiveSessionUntil.After(clock.Now())
			if err == nil && u.IsActive && sessionValid {
				c.Set(ctxUserKey, u)
			} else {
				s.Clear()
				_ = s.Save()
			}
		}
		c.Next()
	}
}

// RequireAuth: wajib login.
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Halaman authenticated tidak boleh disimpan browser. Tanpa ini,
		// tombol Back dapat menampilkan layar akun lama walaupun session sudah logout.
		c.Header("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")
		if CurrentUser(c) == nil {
			if c.GetHeader("HX-Request") == "true" {
				c.Header("HX-Redirect", "/login")
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequireRole: hanya role tertentu yang boleh lewat.
func RequireRole(roles ...models.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := CurrentUser(c)
		if u == nil {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}
		for _, r := range roles {
			if u.Role == r {
				c.Next()
				return
			}
		}
		c.String(http.StatusForbidden, "Anda tidak memiliki akses ke halaman ini.")
		c.Abort()
	}
}

// RequireOutlet: PM wajib punya outlet aktif sebelum bisa input mutasi.
func RequireOutlet() gin.HandlerFunc {
	return func(c *gin.Context) {
		u := CurrentUser(c)
		if u == nil || u.OutletID == nil {
			c.String(http.StatusForbidden, "Akun Anda belum ditempatkan di outlet. Hubungi admin.")
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequireSuperAdmin membatasi operasi sensitif ke akun admin.a.
// Role admin saja tidak cukup karena admin.b juga memiliki RoleAdmin.
func RequireSuperAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		u := CurrentUser(c)
		if u == nil || u.Role != models.RoleAdmin || u.Username != "admin.a" {
			c.String(http.StatusForbidden, "Fitur ini hanya tersedia untuk Super Admin.")
			c.Abort()
			return
		}
		c.Next()
	}
}
