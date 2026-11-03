package middleware

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"movara/internal/models"
	"movara/internal/repository"
)

const (
	ctxUserKey     = "currentUser"
	SessionUserKey = "uid"
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
			if err == nil && u.IsActive {
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
func RequireOutlet(outlets repository.Outlets) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := CurrentUser(c)
		if u == nil || u.OutletID == nil {
			c.String(http.StatusForbidden, "Akun Anda belum ditempatkan di outlet. Hubungi admin.")
			c.Abort()
			return
		}
		rows, err := outlets.List()
		if err != nil {
			c.String(http.StatusForbidden, "Outlet akun tidak dapat diverifikasi. Hubungi admin.")
			c.Abort()
			return
		}
		found := false
		for _, outlet := range rows {
			if outlet.ID == *u.OutletID {
				found = true
				if !outlet.IsActive {
					c.String(http.StatusForbidden, "Outlet akun sedang nonaktif. Hubungi admin.")
					c.Abort()
					return
				}
				break
			}
		}
		if !found {
			c.String(http.StatusForbidden, "Outlet akun tidak ditemukan. Hubungi admin.")
			c.Abort()
			return
		}
		c.Next()
	}
}
