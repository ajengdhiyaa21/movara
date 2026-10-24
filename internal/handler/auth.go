package handler

import (
	"net/http"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"movara/internal/middleware"
	"movara/internal/models"
	"movara/internal/service"
)

type AuthHandler struct {
	r     *Renderer
	auth  *service.AuthService
	audit *service.AuditService
}

func NewAuthHandler(r *Renderer, a *service.AuthService, au *service.AuditService) *AuthHandler {
	return &AuthHandler{r: r, auth: a, audit: au}
}

func (h *AuthHandler) ShowLogin(c *gin.Context) {
	setNoStore(c)
	if middleware.CurrentUser(c) != nil {
		c.Redirect(http.StatusFound, "/")
		return
	}
	h.r.HTML(c, http.StatusOK, "login.html", gin.H{"Title": "Masuk", "Error": "", "Username": ""})
}

func (h *AuthHandler) Login(c *gin.Context) {
	username := strings.TrimSpace(c.PostForm("username"))
	password := c.PostForm("password")

	u, err := h.auth.Authenticate(username, password)
	if err != nil {
		h.r.HTML(c, http.StatusUnauthorized, "login.html", gin.H{
			"Title": "Masuk", "Error": err.Error(), "Username": username,
		})
		return
	}

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
