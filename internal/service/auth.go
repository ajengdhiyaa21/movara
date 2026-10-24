package service

import (
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"movara/internal/models"
	"movara/internal/repository"
)

var ErrInvalidCredentials = errors.New("Username atau password salah.")

type AuthService struct{ users repository.Users }

func NewAuthService(users repository.Users) *AuthService { return &AuthService{users: users} }

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
