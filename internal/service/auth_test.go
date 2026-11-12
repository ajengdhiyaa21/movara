package service_test

import (
	"errors"
	"testing"
	"time"

	"movara/internal/repository/memory"
	"movara/internal/service"
)

func TestAccountSessionIsLimitedToOneActiveDevice(t *testing.T) {
	repos, err := memory.New()
	if err != nil {
		t.Fatalf("membuat repository memori: %v", err)
	}
	auth := service.NewAuthService(repos.Users)
	account, err := auth.AuthenticateByCode("manager", memory.DemoPassword)
	if err != nil {
		t.Fatalf("autentikasi akun: %v", err)
	}

	now := time.Now()
	if err := auth.ClaimSession(account.ID, "session-perangkat-pertama", now.Add(time.Hour), now); err != nil {
		t.Fatalf("klaim session pertama: %v", err)
	}
	if err := auth.ClaimSession(account.ID, "session-perangkat-kedua", now.Add(time.Hour), now); !errors.Is(err, service.ErrAccountSessionActive) {
		t.Fatalf("login kedua seharusnya ditolak, mendapat: %v", err)
	}

	auth.ReleaseSession(account.ID, "session-perangkat-pertama")
	if err := auth.ClaimSession(account.ID, "session-perangkat-kedua", now.Add(time.Hour), now); err != nil {
		t.Fatalf("klaim session setelah logout: %v", err)
	}
}
