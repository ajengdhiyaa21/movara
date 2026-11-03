package service

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"movara/internal/clock"
	"movara/internal/models"
	"movara/internal/repository"
)

const MaxQty = 999

var (
	ErrNoOutlet         = errors.New("Akun Anda belum ditempatkan di outlet. Hubungi admin.")
	ErrBarcodeEmpty     = errors.New("Scan atau ketik barkode produk terlebih dahulu.")
	ErrProductNotFound  = errors.New("Produk belum terdaftar di master data.")
	ErrQtyInvalid       = errors.New("Jumlah mutasi harus antara 1 sampai 999 pcs.")
	ErrBATooLong        = errors.New("Nama BA maksimal 100 karakter.")
	ErrProductRequest   = errors.New("Klik tombol Ajukan Produk Baru untuk menyimpan barcode yang belum terdaftar.")
	ErrInvalidDirection = errors.New("Arah mutasi tidak valid.")
)

type MutationService struct {
	products  repository.Products
	mutations repository.Mutations
	edits     repository.MutationEdits
	audit     *AuditService
}

func NewMutationService(p repository.Products, m repository.Mutations, e repository.MutationEdits, a *AuditService) *MutationService {
	return &MutationService{products: p, mutations: m, edits: e, audit: a}
}

func (s *MutationService) UpdateQtyForPM(u *models.User, id, qty uint) error {
	if u == nil || u.OutletID == nil || u.Role != models.RolePM {
		return repository.ErrNotFound
	}
	if qty > MaxQty {
		return ErrQtyInvalid
	}
	return s.edits.UpdateQty(id, *u.OutletID, u.ID, int(qty), clock.Now())
}

func (s *MutationService) EditHistory(u *models.User, id uint) ([]models.MutationEditLog, error) {
	if u == nil || u.OutletID == nil || u.Role != models.RolePM {
		return nil, repository.ErrNotFound
	}
	return s.edits.ListByMutation(id, *u.OutletID)
}

type TodayResult struct {
	Rows     []models.Mutation // sudah difilter pencarian
	TotalQty int               // total seluruh mutasi hari ini (tidak terpengaruh pencarian)
	Date     string
}

// Period mengembalikan riwayat pada rentang waktu tertentu untuk outlet PM.
func (s *MutationService) Period(outletID uint, categoryID *uint, q, direction string, from, to time.Time) ([]models.Mutation, error) {
	rows, _, err := s.mutations.Search(repository.MutationFilter{OutletID: &outletID, CategoryID: categoryID, Direction: direction, From: from, To: to, Query: q}, 0, 0)
	return rows, err
}

// Today mengembalikan mutasi hari ini (WIB) di satu outlet, terbaru di atas.
func (s *MutationService) Today(outletID uint, q string) (*TodayResult, error) {
	return s.ByDate(outletID, nil, q, "")
}

func (s *MutationService) ByDate(outletID uint, categoryID *uint, q, date string) (*TodayResult, error) {
	now := clock.Now()
	if date != "" {
		if parsed, e := time.ParseInLocation("2006-01-02", date, clock.WIB); e == nil {
			now = parsed
		}
	}
	from, to := clock.DayBounds(now)
	all, _, err := s.mutations.Search(repository.MutationFilter{OutletID: &outletID, CategoryID: categoryID, From: from, To: to}, 0, 0)
	if err != nil {
		return nil, err
	}
	q = strings.ToLower(strings.TrimSpace(q))
	res := &TodayResult{Date: from.Format("2006-01-02")}
	for _, m := range all {
		res.TotalQty += m.Qty
		if q == "" ||
			strings.Contains(strings.ToLower(m.Barcode), q) ||
			strings.Contains(strings.ToLower(m.Code), q) ||
			strings.Contains(strings.ToLower(m.ProductName), q) {
			res.Rows = append(res.Rows, m)
		}
	}
	return res, nil
}

func (s *MutationService) FindProduct(barcode string, outletID uint) (*models.Product, error) {
	barcode = strings.TrimSpace(barcode)
	if barcode == "" {
		return nil, ErrBarcodeEmpty
	}
	p, err := s.products.ByBarcode(barcode, outletID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrProductNotFound
	}
	return p, err
}

func (s *MutationService) FindProductForUser(barcode string, u *models.User) (*models.Product, error) {
	if u == nil || u.OutletID == nil {
		return nil, ErrNoOutlet
	}
	p, err := s.FindProduct(barcode, *u.OutletID)
	if err != nil {
		return nil, err
	}
	if u.CategoryID != nil && (p.CategoryID == nil || *p.CategoryID != *u.CategoryID) {
		return nil, ErrProductNotFound
	}
	return p, nil
}

// DuplicateToday: berapa kali barkode ini sudah diinput hari ini di outlet tsb.
func (s *MutationService) DuplicateToday(outletID uint, barcode string) (count, qty int) {
	from, to := clock.DayBounds(clock.Now())
	all, err := s.mutations.ListByOutletBetween(outletID, from, to)
	if err != nil {
		return 0, 0
	}
	for _, m := range all {
		if m.Barcode == barcode {
			count++
			qty += m.Qty
		}
	}
	return count, qty
}

// Add menyimpan mutasi. Outlet SELALU diambil dari akun yang login,
// nama/kode produk SELALU dari master data (bukan dari input form).
func (s *MutationService) Add(u *models.User, barcode, productName, direction string, qty int, ba string) (*models.Mutation, error) {
	if u == nil || u.OutletID == nil {
		return nil, ErrNoOutlet
	}
	p, err := s.FindProductForUser(barcode, u)
	productName = strings.TrimSpace(productName)
	if direction != "market_to_counter" && direction != "counter_to_market" {
		return nil, ErrInvalidDirection
	}
	if err != nil && !errors.Is(err, ErrProductNotFound) {
		return nil, err
	}
	if qty < 1 || qty > MaxQty {
		return nil, ErrQtyInvalid
	}
	ba = strings.ToUpper(strings.TrimSpace(ba))
	if utf8.RuneCountInString(ba) > 100 {
		return nil, ErrBATooLong
	}

	m := &models.Mutation{Barcode: barcode, Qty: qty, OutletID: *u.OutletID, UserID: u.ID, CreatedAt: clock.Now(), Status: "approved", Direction: direction}
	if p != nil {
		m.ProductID, m.Code, m.ProductName, m.CategoryID = &p.ID, p.Code, p.Name, p.CategoryID
	} else {
		if productName == "" {
			productName = "Menunggu informasi admin"
		}
		m.ProductName, m.Status = productName, "pending_code"
	}
	if ba != "" {
		m.BAName = &ba
	}
	if err := s.mutations.Create(m); err != nil {
		return nil, err
	}
	action := "mutation_create"
	if m.Status == "pending_code" {
		action = "mutation_product_request"
	}
	s.audit.Log(&u.ID, u.OutletID, action,
		fmt.Sprintf("Input %d pcs %s (Ref: #MUT-%s-%03d)", qty, m.ProductName, m.CreatedAt.Format("20060102"), m.ID))
	return m, nil
}
