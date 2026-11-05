package service

import (
	"time"

	"movara/internal/models"
	"movara/internal/repository"
)

const DefaultPerPage = 10

type RecapService struct {
	mutations repository.Mutations
	outlets   repository.Outlets
}

func NewRecapService(m repository.Mutations, o repository.Outlets) *RecapService {
	return &RecapService{mutations: m, outlets: o}
}

type RecapQuery struct {
	OutletID      *uint
	CategoryID    *uint
	Direction     string
	From, To      time.Time // To eksklusif; zero = tanpa batas
	Q             string
	Page, PerPage int
	All           bool
}

func (q RecapQuery) filter() repository.MutationFilter {
	return repository.MutationFilter{OutletID: q.OutletID, CategoryID: q.CategoryID, Direction: q.Direction, From: q.From, To: q.To, Query: q.Q}
}

type RecapResult struct {
	Rows     []models.Mutation
	Total    int // jumlah baris yang cocok filter
	TotalQty int // total pcs yang cocok filter (seluruh halaman)
	Page     int
	Pages    int
	PerPage  int
	ByOutlet []repository.OutletTotal
}

type RecapSummary struct {
	MonthQty  int     // bulan berjalan sampai hari ini
	Growth    float64 // persen vs periode yang sama bulan lalu
	HasGrowth bool
	TodayQty  int
	TopOutlet *repository.OutletTotal // outlet teraktif hari ini (semua outlet)
	TopShare  float64                 // persen dari volume harian semua outlet
}

func (s *RecapService) Outlets() ([]models.Outlet, error) { return s.outlets.List() }

func (s *RecapService) Search(q RecapQuery) (*RecapResult, error) {
	if q.PerPage <= 0 {
		q.PerPage = DefaultPerPage
	}
	if q.Page < 1 {
		q.Page = 1
	}
	f := q.filter()

	rows, total, err := s.mutations.Search(f, (q.Page-1)*q.PerPage, q.PerPage)
	if err != nil {
		return nil, err
	}
	if q.All {
		rows, _, err = s.mutations.Search(f, 0, total)
		if err != nil {
			return nil, err
		}
		q.PerPage = total
		if q.PerPage < 1 {
			q.PerPage = 1
		}
	}
	pages := (total + q.PerPage - 1) / q.PerPage
	if pages < 1 {
		pages = 1
	}
	if q.Page > pages { // halaman di luar jangkauan: mundur ke halaman terakhir
		q.Page = pages
		if rows, _, err = s.mutations.Search(f, (q.Page-1)*q.PerPage, q.PerPage); err != nil {
			return nil, err
		}
	}

	qty, err := s.mutations.TotalQty(f)
	if err != nil {
		return nil, err
	}
	by, err := s.mutations.QtyByOutlet(f)
	if err != nil {
		return nil, err
	}
	return &RecapResult{Rows: rows, Total: total, TotalQty: qty, Page: q.Page,
		Pages: pages, PerPage: q.PerPage, ByOutlet: by}, nil
}

// ExportRows mengembalikan SEMUA baris yang cocok filter (tanpa pagination).
func (s *RecapService) ExportRows(q RecapQuery) ([]models.Mutation, error) {
	rows, _, err := s.mutations.Search(q.filter(), 0, 0)
	return rows, err
}

// Summary untuk kartu ringkasan. outletID membatasi total bulan ini & hari ini;
// "outlet teraktif" selalu dihitung dari semua outlet.
func (s *RecapService) Summary(query RecapQuery) (*RecapSummary, error) {
	base := query.filter()
	selected, err := s.mutations.TotalQty(base)
	if err != nil {
		return nil, err
	}
	byOutlet, err := s.mutations.QtyByOutlet(base)
	if err != nil {
		return nil, err
	}
	sum := &RecapSummary{MonthQty: selected, TodayQty: selected, TopOutlet: topOutlet(byOutlet)}
	inputCount := 0
	for _, row := range byOutlet {
		inputCount += row.Count
	}
	sum.MonthQty = inputCount
	if sum.TopOutlet != nil {
		all := 0
		for _, row := range byOutlet {
			all += row.Count
		}
		if all > 0 {
			sum.TopShare = float64(sum.TopOutlet.Count) / float64(all) * 100
		}
	}
	return sum, nil
}

func topOutlet(rows []repository.OutletTotal) *repository.OutletTotal {
	if len(rows) == 0 {
		return nil
	}
	top := rows[0]
	return &top
}
