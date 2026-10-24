package middleware

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"net/url"
	"strings"
)

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
