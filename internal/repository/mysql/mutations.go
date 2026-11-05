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
		oldQty := m.Qty
		if err := tx.Model(&m).Update("qty", qty).Error; err != nil {
			return err
		}
		return tx.Create(&models.MutationEditLog{MutationID: m.ID, UserID: userID, OldQty: oldQty, NewQty: qty, CreatedAt: now}).Error
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
	q := s.db.Model(&models.Mutation{})
	q = applyMutationFilter(q, f)
	var total int64
	if err := q.Select("COALESCE(SUM(qty), 0)").Scan(&total).Error; err != nil {
		return 0, err
	}
	return int(total), nil
}
func (s *Store) QtyByOutlet(f repository.MutationFilter) ([]repository.OutletTotal, error) {
	type row struct {
		OutletID   uint
		Code, Name string
		Qty        int
		Count      int
	}
	var rows []row
	q := s.db.Table("mutations").Select("mutations.outlet_id, outlets.code, outlets.name, SUM(mutations.qty) AS qty, COUNT(mutations.id) AS count").Joins("JOIN outlets ON outlets.id = mutations.outlet_id")
	q = applyMutationFilter(q, f)
	if err := q.Group("mutations.outlet_id, outlets.code, outlets.name").Order("count DESC").Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]repository.OutletTotal, 0, len(rows))
	for _, r := range rows {
		out = append(out, repository.OutletTotal{OutletID: r.OutletID, Code: r.Code, Name: r.Name, Qty: r.Qty, Count: r.Count})
	}
	return out, nil
}

func applyMutationFilter(q *gorm.DB, f repository.MutationFilter) *gorm.DB {
	if f.OutletID != nil {
		q = q.Where("mutations.outlet_id = ?", *f.OutletID)
	}
	if f.CategoryID != nil {
		q = q.Where("mutations.category_id = ?", *f.CategoryID)
	}
	if f.Direction != "" {
		q = q.Where("mutations.direction = ?", f.Direction)
	}
	if !f.From.IsZero() {
		q = q.Where("mutations.created_at >= ?", f.From)
	}
	if !f.To.IsZero() {
		q = q.Where("mutations.created_at < ?", f.To)
	}
	if x := strings.TrimSpace(f.Query); x != "" {
		like := "%" + x + "%"
		q = q.Where("(mutations.barcode LIKE ? OR mutations.code LIKE ? OR mutations.product_name LIKE ?)", like, like, like)
	}
	return q
}
