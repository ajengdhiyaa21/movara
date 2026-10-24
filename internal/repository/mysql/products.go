package mysql

import (
	"movara/internal/models"
	"movara/internal/repository"
	"strings"
)

func (s *Store) ByBarcode(v string, outletID uint) (*models.Product, error) {
	var p models.Product
	if err := s.db.Where("barcode = ? AND outlet_id = ? AND is_active = ?", strings.TrimSpace(v), outletID, true).First(&p).Error; err != nil {
		return nil, repository.ErrNotFound
	}
	return &p, nil
}
