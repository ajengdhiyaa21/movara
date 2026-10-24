package memory

import (
	"sort"
	"strings"

	"movara/internal/models"
	"movara/internal/repository"
)

func matches(m *models.Mutation, f repository.MutationFilter, q string) bool {
	if f.OutletID != nil && m.OutletID != *f.OutletID {
		return false
	}
	if f.CategoryID != nil && (m.CategoryID == nil || *m.CategoryID != *f.CategoryID) {
		return false
	}
	if f.Direction != "" && m.Direction != f.Direction {
		return false
	}
	if !f.From.IsZero() && m.CreatedAt.Before(f.From) {
		return false
	}
	if !f.To.IsZero() && !m.CreatedAt.Before(f.To) {
		return false
	}
	if q != "" &&
		!strings.Contains(strings.ToLower(m.Barcode), q) &&
		!strings.Contains(strings.ToLower(m.Code), q) &&
		!strings.Contains(strings.ToLower(m.ProductName), q) {
		return false
	}
	return true
}

// ---- repository.Outlets ----

func (s *Store) List() ([]models.Outlet, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]models.Outlet, 0, len(s.outlets))
	for _, o := range s.outlets {
		out = append(out, o)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// ---- repository.Mutations (laporan) ----

func (s *Store) Search(f repository.MutationFilter, offset, limit int) ([]models.Mutation, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	q := strings.ToLower(strings.TrimSpace(f.Query))

	var all []models.Mutation
	for i := range s.mutations {
		m := s.mutations[i]
		if matches(&m, f, q) {
			all = append(all, m)
		}
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].CreatedAt.Equal(all[j].CreatedAt) {
			return all[i].ID > all[j].ID
		}
		return all[i].CreatedAt.After(all[j].CreatedAt)
	})

	total := len(all)
	if offset < 0 {
		offset = 0
	}
	if offset > total {
		offset = total
	}
	end := total
	if limit > 0 && offset+limit < total {
		end = offset + limit
	}
	page := all[offset:end]
	for i := range page {
		page[i].Outlet = s.outlets[page[i].OutletID]
		page[i].User = s.users[page[i].UserID]
	}
	return page, total, nil
}

func (s *Store) TotalQty(f repository.MutationFilter) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	q := strings.ToLower(strings.TrimSpace(f.Query))
	sum := 0
	for i := range s.mutations {
		if matches(&s.mutations[i], f, q) {
			sum += s.mutations[i].Qty
		}
	}
	return sum, nil
}

func (s *Store) QtyByOutlet(f repository.MutationFilter) ([]repository.OutletTotal, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	q := strings.ToLower(strings.TrimSpace(f.Query))
	sums := map[uint]int{}
	for i := range s.mutations {
		if matches(&s.mutations[i], f, q) {
			sums[s.mutations[i].OutletID] += s.mutations[i].Qty
		}
	}
	out := make([]repository.OutletTotal, 0, len(sums))
	for id, qty := range sums {
		o := s.outlets[id]
		out = append(out, repository.OutletTotal{OutletID: id, Code: o.Code, Name: o.Name, Qty: qty})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Qty == out[j].Qty {
			return out[i].OutletID < out[j].OutletID
		}
		return out[i].Qty > out[j].Qty
	})
	return out, nil
}
