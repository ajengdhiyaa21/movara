package mysql

import (
	"time"

	"movara/internal/models"
	"movara/internal/repository"
)

func (s *Store) ByID(id uint) (*models.User, error) {
	var u models.User
	if err := s.db.Preload("Outlet").Preload("Category").First(&u, id).Error; err != nil {
		return nil, repository.ErrNotFound
	}
	return &u, nil
}
func (s *Store) ByUsername(v string) (*models.User, error) {
	var u models.User
	if err := s.db.Preload("Outlet").Preload("Category").Where("username = ?", v).First(&u).Error; err != nil {
		return nil, repository.ErrNotFound
	}
	return &u, nil
}
func (s *Store) ByUserCode(v string) (*models.User, error) {
	var u models.User
	if err := s.db.Preload("Outlet").Preload("Category").Where("user_code = ?", v).First(&u).Error; err != nil {
		return nil, repository.ErrNotFound
	}
	return &u, nil
}

func (s *Store) ClaimSession(id uint, token string, until, now time.Time) (bool, error) {
	result := s.db.Model(&models.User{}).
		Where("id = ? AND (active_session_token = '' OR active_session_token IS NULL OR active_session_until IS NULL OR active_session_until <= ?)", id, now).
		Updates(map[string]interface{}{"active_session_token": token, "active_session_until": until})
	return result.RowsAffected == 1, result.Error
}

func (s *Store) ReleaseSession(id uint, token string) error {
	return s.db.Model(&models.User{}).
		Where("id = ? AND active_session_token = ?", id, token).
		Updates(map[string]interface{}{"active_session_token": "", "active_session_until": nil}).Error
}
