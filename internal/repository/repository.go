// Package repository mendefinisikan kontrak akses data.
// Tahap tampilan memakai implementasi memory (data contoh);
// tahap database nanti cukup menambah implementasi PostgreSQL
// dengan interface yang sama, tanpa mengubah service/handler.
package repository

import (
	"errors"
	"time"

	"movara/internal/models"
)

var ErrNotFound = errors.New("data tidak ditemukan")

type Users interface {
	ByID(id uint) (*models.User, error)
	ByUsername(username string) (*models.User, error)
	ByUserCode(code string) (*models.User, error)
}

type Products interface {
	ByBarcode(barcode string, outletID uint) (*models.Product, error)
}

type Outlets interface {
	List() ([]models.Outlet, error)
}

// MutationFilter: From inklusif, To eksklusif; zero value = tanpa batas.
// Query dicocokkan ke barkode, kode, atau nama produk (tidak peka huruf besar/kecil).
type MutationFilter struct {
	OutletID   *uint
	CategoryID *uint
	Direction  string
	From, To   time.Time
	Query      string
}

type OutletTotal struct {
	OutletID uint
	Code     string
	Name     string
	Qty      int
	Count    int // jumlah input/transaksi, bukan total PCS
}

type Mutations interface {
	// ListByOutletBetween: from inklusif, to eksklusif. Terbaru di atas.
	ListByOutletBetween(outletID uint, from, to time.Time) ([]models.Mutation, error)
	Create(m *models.Mutation) error

	// Search: terbaru di atas. limit <= 0 berarti semua baris (untuk export).
	// Mengembalikan juga jumlah total baris yang cocok (sebelum pagination).
	Search(f MutationFilter, offset, limit int) ([]models.Mutation, int, error)
	TotalQty(f MutationFilter) (int, error)
	// QtyByOutlet: total qty per outlet, terbesar di atas.
	QtyByOutlet(f MutationFilter) ([]OutletTotal, error)
}

type MutationEdits interface {
	UpdateQty(mutationID, outletID, userID uint, qty int, now time.Time) error
	ListByMutation(mutationID, outletID uint) ([]models.MutationEditLog, error)
}

type Audits interface {
	Add(a *models.AuditLog) error
}

type Repos struct {
	Users         Users
	Products      Products
	Outlets       Outlets
	Mutations     Mutations
	MutationEdits MutationEdits
	Audits        Audits
}
