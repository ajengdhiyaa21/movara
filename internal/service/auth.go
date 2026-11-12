package service

import (
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"movara/internal/models"
	"movara/internal/repository"
)

var (
	ErrInvalidCredentials   = errors.New("Username atau password salah.")
	ErrAccountSessionActive = errors.New("Akun ini masih aktif di perangkat lain. Silakan logout dari perangkat tersebut terlebih dahulu.")
)

type AuthService struct{ users repository.Users }

func NewAuthService(users repository.Users) *AuthService { return &AuthService{users: users} }

func (s *AuthService) LookupByCode(code string) (*models.User, error) {
	return s.users.ByUserCode(strings.TrimSpace(code))
}

func (s *AuthService) Authenticate(username, password string) (*models.User, error) {
	u, err := s.users.ByUsername(strings.ToLower(strings.TrimSpace(username)))
	if err != nil || !u.IsActive {
		return nil, ErrInvalidCredentials
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return nil, ErrInvalidCredentials
	}
	return u, nil
}

func (s *AuthService) AuthenticateByCode(code, password string) (*models.User, error) {
	u, err := s.users.ByUserCode(strings.TrimSpace(code))
	if err != nil || !u.IsActive {
		return nil, ErrInvalidCredentials
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return nil, ErrInvalidCredentials
	}
	return u, nil
}

func (s *AuthService) ClaimSession(id uint, token string, until, now time.Time) error {
	claimed, err := s.users.ClaimSession(id, token, until, now)
	if err != nil {
		return err
	}
	if !claimed {
		return ErrAccountSessionActive
	}
	return nil
}

func (s *AuthService) ReleaseSession(id uint, token string) {
	_ = s.users.ReleaseSession(id, token)
}
