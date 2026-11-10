package handler

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"movara/internal/middleware"
	"movara/internal/models"
	"movara/internal/service"
)

type AuthHandler struct {
	r        *Renderer
	auth     *service.AuthService
	audit    *service.AuditService
	mu       sync.Mutex
	failures map[string][]time.Time
}

func NewAuthHandler(r *Renderer, a *service.AuthService, au *service.AuditService) *AuthHandler {
	return &AuthHandler{r: r, auth: a, audit: au, failures: make(map[string][]time.Time)}
}

func (h *AuthHandler) ShowLogin(c *gin.Context) {
	setNoStore(c)
	if middleware.CurrentUser(c) != nil {
		c.Redirect(http.StatusFound, "/")
		return
	}
	h.r.HTML(c, http.StatusOK, "login.html", gin.H{"Title": "Masuk", "Error": "", "UserCode": "", "Username": ""})
}

func (h *AuthHandler) Login(c *gin.Context) {
	userCode := strings.TrimSpace(c.PostForm("user_code"))
	username := strings.TrimSpace(c.PostForm("username"))
	password := c.PostForm("password")
	key := c.ClientIP() + ":" + strings.ToLower(userCode)
	if !h.allowed(key) {
		h.r.HTML(c, http.StatusTooManyRequests, "login.html", gin.H{"Title": "Masuk", "Error": "Terlalu banyak percobaan login. Coba lagi dalam satu menit.", "Username": username})
		return
	}

	u, err := h.auth.AuthenticateByCode(userCode, password)
	if err != nil {
		h.recordFailure(key)
		h.r.HTML(c, http.StatusUnauthorized, "login.html", gin.H{
			"Title": "Masuk", "Error": err.Error(), "UserCode": userCode,
		})
		return
	}
	h.clearFailures(key)

	s := sessions.Default(c)
	s.Clear()
	s.Set(middleware.SessionUserKey, u.ID)
	if err := s.Save(); err != nil {
		c.String(http.StatusInternalServerError, "gagal menyimpan session")
		return
	}
	h.audit.Log(&u.ID, u.OutletID, "login", u.Name+" masuk ke sistem")
	c.Redirect(http.StatusFound, "/")
}

func (h *AuthHandler) LookupUser(c *gin.Context) {
	code := strings.TrimSpace(c.Query("user_code"))
	u, err := h.auth.LookupByCode(code)
	if err != nil || !u.IsActive {
		c.JSON(http.StatusNotFound, gin.H{"error": "ID pengguna tidak ditemukan."})
		return
	}
	c.JSON(http.StatusOK, gin.H{"username": u.Username, "name": u.Name})
}

func (h *AuthHandler) allowed(key string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	now := time.Now()
	cutoff := now.Add(-time.Minute)
	kept := h.failures[key][:0]
	for _, t := range h.failures[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	h.failures[key] = kept
	return len(kept) < 5
}

func (h *AuthHandler) recordFailure(key string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.failures[key] = append(h.failures[key], time.Now())
}
func (h *AuthHandler) clearFailures(key string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.failures, key)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	if u := middleware.CurrentUser(c); u != nil {
		h.audit.Log(&u.ID, u.OutletID, "logout", u.Name+" keluar dari sistem")
	}
	s := sessions.Default(c)
	s.Clear()
	_ = s.Save()
	setNoStore(c)
	c.Header("Clear-Site-Data", `"cache"`)
	c.Redirect(http.StatusFound, "/login")
}

func setNoStore(c *gin.Context) {
	c.Header("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")
}

// Home mengarahkan user ke halaman utama sesuai role.
func Home(c *gin.Context) {
	u := middleware.CurrentUser(c)
	if u.Role == models.RolePM {
		c.Redirect(http.StatusFound, "/mutasi")
		return
	}
	c.Redirect(http.StatusFound, "/rekap")
}
