package mysql

import (
	"gorm.io/gorm"
	"movara/internal/models"
	"movara/internal/repository"
	"strings"
	"time"
)

func (s *Store) UpdateQty(mutationID, outletID, userID uint, qty int, now time.Time) error {
	from := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return s.db.Transaction(func(tx *gorm.DB) error {
		var m models.Mutation
		if err := tx.Where("id = ? AND outlet_id = ? AND created_at >= ? AND created_at < ?", mutationID, outletID, from, from.AddDate(0, 0, 1)).First(&m).Error; err != nil {
			return err
		}
		if m.Qty == qty {
			return nil
		}
		if err := tx.Model(&m).Update("qty", qty).Error; err != nil {
			return err
		}
		return tx.Create(&models.MutationEditLog{MutationID: m.ID, UserID: userID, OldQty: m.Qty, NewQty: qty, CreatedAt: now}).Error
	})
}

func (s *Store) ListByMutation(mutationID, outletID uint) ([]models.MutationEditLog, error) {
	var v []models.MutationEditLog
	err := s.db.Preload("User").Joins("JOIN mutations ON mutations.id = mutation_edit_logs.mutation_id").Where("mutation_edit_logs.mutation_id = ? AND mutations.outlet_id = ?", mutationID, outletID).Order("mutation_edit_logs.created_at DESC").Find(&v).Error
	return v, err
}

func (s *Store) ListByOutletBetween(id uint, from, to time.Time) ([]models.Mutation, error) {
	var v []models.Mutation
	err := s.db.Preload("Outlet").Preload("User").Preload("Category").Where("outlet_id = ? AND created_at >= ? AND created_at < ?", id, from, to).Order("created_at DESC").Order("id DESC").Find(&v).Error
	return v, err
}
func (s *Store) Create(m *models.Mutation) error { return s.db.Create(m).Error }
func (s *Store) Search(f repository.MutationFilter, off, lim int) ([]models.Mutation, int, error) {
	q := s.db.Model(&models.Mutation{})
	if f.OutletID != nil {
		q = q.Where("outlet_id = ?", *f.OutletID)
	}
	if f.CategoryID != nil {
		q = q.Where("category_id = ?", *f.CategoryID)
	}
	if f.Direction != "" {
		q = q.Where("direction = ?", f.Direction)
	}
	if !f.From.IsZero() {
		q = q.Where("created_at >= ?", f.From)
	}
	if !f.To.IsZero() {
		q = q.Where("created_at < ?", f.To)
	}
	if x := strings.TrimSpace(f.Query); x != "" {
		like := "%" + x + "%"
		q = q.Where("barcode LIKE ? OR code LIKE ? OR product_name LIKE ?", like, like, like)
	}
	var n int64
	q.Count(&n)
	var v []models.Mutation
	q = q.Order("created_at DESC").Order("id DESC")
	if off > 0 {
		q = q.Offset(off)
	}
	if lim > 0 {
		q = q.Limit(lim)
	}
	err := q.Preload("Outlet").Preload("User").Preload("Category").Find(&v).Error
	return v, int(n), err
}
func (s *Store) TotalQty(f repository.MutationFilter) (int, error) {
	v, _, err := s.Search(f, 0, 0)
	n := 0
	for _, m := range v {
		n += m.Qty
	}
	return n, err
}
func (s *Store) QtyByOutlet(f repository.MutationFilter) ([]repository.OutletTotal, error) {
	v, _, err := s.Search(f, 0, 0)
	if err != nil {
		return nil, err
	}
	m := map[uint]*repository.OutletTotal{}
	for _, x := range v {
		z := m[x.OutletID]
		if z == nil {
			z = &repository.OutletTotal{OutletID: x.OutletID, Code: x.Outlet.Code, Name: x.Outlet.Name}
			m[x.OutletID] = z
		}
		z.Qty += x.Qty
	}
	out := make([]repository.OutletTotal, 0, len(m))
	for _, x := range m {
		out = append(out, *x)
	}
	return out, nil
}
