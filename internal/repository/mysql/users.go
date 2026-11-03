package mysql

import (
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

func (s *Store) ListByRole(role models.Role) ([]models.User, error) {
	var users []models.User
	err := s.db.Where("role = ?", role).Find(&users).Error
	return users, err
}
