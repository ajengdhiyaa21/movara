package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

const CSRFSessionKey = "csrf_token"

// CSRFToken returns the session-bound token and creates it on safe requests.
// Templates/HTMX clients can send it back in X-CSRF-Token or _csrf.
func CSRFToken(c *gin.Context) (string, error) {
	s := sessions.Default(c)
	if token, ok := s.Get(CSRFSessionKey).(string); ok && token != "" {
		return token, nil
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	s.Set(CSRFSessionKey, token)
	if err := s.Save(); err != nil {
		return "", err
	}
	return token, nil
}

// RequireCSRF protects unsafe requests using the token stored in the session.
// SameOrigin remains a separate defense and is not treated as a substitute.
func RequireCSRF() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodPost && c.Request.Method != http.MethodPut && c.Request.Method != http.MethodPatch && c.Request.Method != http.MethodDelete {
			c.Next()
			return
		}
		s := sessions.Default(c)
		expected, ok := s.Get(CSRFSessionKey).(string)
		if !ok || expected == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "csrf token tidak valid"})
			return
		}
		provided := c.GetHeader("X-CSRF-Token")
		if provided == "" {
			provided = c.PostForm("_csrf")
		}
		if !csrfEqual(expected, provided) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "csrf token tidak valid"})
			return
		}
		c.Next()
	}
}

func csrfEqual(expected, provided string) bool {
	if expected == "" || provided == "" || len(expected) != len(provided) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(provided)) == 1
}
