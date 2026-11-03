package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/url"
	"strings"
)

const CSRFSessionKey = "csrf_token"

func CSRFToken(c *gin.Context) string {
	s := sessions.Default(c)
	if token, ok := s.Get(CSRFSessionKey).(string); ok && token != "" {
		return token
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	s.Set(CSRFSessionKey, token)
	_ = s.Save()
	return token
}

func CSRF() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := CSRFToken(c)
		if c.Request.Method == http.MethodPost || c.Request.Method == http.MethodPut || c.Request.Method == http.MethodDelete {
			provided := c.PostForm("csrf_token")
			if provided == "" {
				provided = c.GetHeader("X-CSRF-Token")
			}
			if token == "" || provided == "" || provided != token {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "token CSRF tidak valid"})
				return
			}
		}
		c.Next()
	}
}

// SameOrigin melindungi request mutasi browser dari form lintas situs.
// Untuk client non-browser, Origin/Referer boleh kosong; request dengan header
// tersebut wajib berasal dari host aplikasi yang sama.
func SameOrigin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodPost && c.Request.Method != http.MethodPut && c.Request.Method != http.MethodDelete {
			c.Next()
			return
		}
		for _, raw := range []string{c.GetHeader("Origin"), c.GetHeader("Referer")} {
			if raw == "" {
				continue
			}
			u, err := url.Parse(raw)
			if err != nil || !strings.EqualFold(u.Host, c.Request.Host) {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "origin request tidak valid"})
				return
			}
		}
		c.Next()
	}
}
