package handler

import (
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"movara/internal/clock"
	"movara/internal/middleware"
	"movara/internal/models"
	"movara/web"
)

// File template yang dipakai bersama oleh semua halaman.
var sharedTemplates = map[string]bool{"layout.html": true, "icons.html": true}

var funcs = template.FuncMap{
	"dict": func(values ...interface{}) map[string]interface{} {
		out := map[string]interface{}{}
		for i := 0; i+1 < len(values); i += 2 {
			if key, ok := values[i].(string); ok {
				out[key] = values[i+1]
			}
		}
		return out
	},
	"jam": func(t time.Time) string { return t.In(clock.WIB).Format("15:04") },
	"initials": func(name string) string {
		parts := strings.Fields(name)
		first := func(s string) string { return strings.ToUpper(string([]rune(s)[:1])) }
		switch len(parts) {
		case 0:
			return "?"
		case 1:
			return first(parts[0])
		}
		return first(parts[0]) + first(parts[len(parts)-1])
	},
	"tgl": func(t time.Time) string { return t.In(clock.WIB).Format("02/01/2006") },
	"num": formatNum,
	"inc": func(n int) int { return n + 1 },
	"add": func(a, b int) int { return a + b },
	"mul": func(a, b int) int { return a * b },
	"sub": func(a, b int) int { return a - b },
	"seq": func(n int) []int {
		out := make([]int, n)
		for i := range out {
			out[i] = i + 1
		}
		return out
	},
	"outletShort": func(o models.Outlet) string { return shortOutlet(o.Code) },
	"firstName": func(name string) string {
		if f := strings.Fields(name); len(f) > 0 {
			return strings.ToUpper(f[0])
		}
		return ""
	},
	"roleLabel": func(r models.Role) string {
		switch r {
		case models.RolePM:
			return "Pramuniaga"
		case models.RoleManager:
			return "Manager"
		case models.RoleAdmin:
			return "Admin"
		}
		return string(r)
	},
}

// Renderer menyimpan satu template set per halaman (layout + ikon + halaman),
// sehingga block "content" tiap halaman tidak saling menimpa.
type Renderer struct {
	pages         map[string]*template.Template
	adminWhatsApp string
	Demo          bool // true selama masih memakai data contoh
}

func NewRenderer(adminWhatsApp string) (*Renderer, error) {
	entries, err := fs.ReadDir(web.Templates, "templates")
	if err != nil {
		return nil, err
	}
	r := &Renderer{pages: map[string]*template.Template{}, adminWhatsApp: digitsOnly(adminWhatsApp)}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || sharedTemplates[name] || !strings.HasSuffix(name, ".html") {
			continue
		}
		t, err := template.New(name).Funcs(funcs).
			ParseFS(web.Templates, "templates/layout.html", "templates/icons.html", "templates/"+name)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", name, err)
		}
		r.pages[name] = t
	}
	return r, nil
}

// NewTestRenderer creates a renderer without parsing production templates.
// It is intended for router tests whose handlers return JSON.
func NewTestRenderer() *Renderer {
	return &Renderer{pages: map[string]*template.Template{}, adminWhatsApp: ""}
}

// WhatsAppURL membuat link wa.me dengan teks otomatis. Kosong jika nomor belum diatur.
func (r *Renderer) WhatsAppURL(text string) string {
	if r.adminWhatsApp == "" {
		return ""
	}
	return "https://wa.me/" + r.adminWhatsApp + "?text=" +
		strings.ReplaceAll(url.QueryEscape(text), "+", "%20")
}

func (r *Renderer) prepare(c *gin.Context, page string, data gin.H) (*template.Template, gin.H, bool) {
	t, ok := r.pages[page]
	if !ok {
		c.String(http.StatusInternalServerError, "template tidak ditemukan: %s", page)
		return nil, nil, false
	}
	if data == nil {
		data = gin.H{}
	}
	if _, ok := data["Active"]; !ok {
		data["Active"] = ""
	}
	data["User"] = middleware.CurrentUser(c)
	data["Demo"] = r.Demo
	data["Year"] = clock.Now().Year()
	return t, data, true
}

// HTML merender halaman penuh (layout + content).
func (r *Renderer) HTML(c *gin.Context, status int, page string, data gin.H) {
	t, data, ok := r.prepare(c, page, data)
	if !ok {
		return
	}
	c.Status(status)
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(c.Writer, "layout", data); err != nil {
		_ = c.Error(err)
	}
}

// Partial merender satu potongan template (untuk respons HTMX).
func (r *Renderer) Partial(c *gin.Context, status int, page, name string, data gin.H) {
	t, data, ok := r.prepare(c, page, data)
	if !ok {
		return
	}
	c.Status(status)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Header("Cache-Control", "no-store")
	if err := t.ExecuteTemplate(c.Writer, name, data); err != nil {
		_ = c.Error(err)
	}
}

func digitsOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// formatNum: 1482 -> "1.482" (format Indonesia).
func formatNum(n int) string {
	neg := n < 0
	if neg {
		n = -n
	}
	s := strconv.Itoa(n)
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(r)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

// shortOutlet: "MK2" -> "MK 2", "MINI1" -> "MINI 1".
func shortOutlet(code string) string {
	for i, r := range code {
		if i > 0 && r >= '0' && r <= '9' {
			return code[:i] + " " + code[i:]
		}
	}
	return code
}
