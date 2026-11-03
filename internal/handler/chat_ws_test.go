package handler

import (
	"testing"
)

func TestOriginAllowedUsesConfiguredPort(t *testing.T) {
	tests := []struct {
		name   string
		origin string
		want   bool
	}{
		{"localhost", "http://localhost:8080", true},
		{"loopback", "http://127.0.0.1:8080", true},
		{"empty", "", false},
		{"wrong port", "http://localhost:9090", false},
		{"foreign host", "http://evil.example:8080", false},
		{"https is not local dev origin", "https://localhost:8080", false},
		{"path", "http://localhost:8080/chat", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := originAllowed(test.origin, "8080"); got != test.want {
				t.Fatalf("originAllowed(%q) = %v, want %v", test.origin, got, test.want)
			}
		})
	}
}
