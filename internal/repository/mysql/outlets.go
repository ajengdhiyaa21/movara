package mysql

import (
	"movara/internal/models"
	"movara/internal/repository"
)

func (s *Store) List() ([]models.Outlet, error) {
	var v []models.Outlet
	err := s.db.Where("is_active = ?", true).Order("id").Find(&v).Error
	return v, err
}

var _ repository.Outlets = (*Store)(nil)
