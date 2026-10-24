package mysql

import "movara/internal/models"

func (s *Store) Add(a *models.AuditLog) error { return s.db.Create(a).Error }
