// Package memory: repository berisi data contoh untuk pengembangan tampilan.
// Data hilang saat server dimatikan.
package memory

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"movara/internal/clock"
	"movara/internal/models"
	"movara/internal/repository"
)

func uintPtr(v uint) *uint { return &v }

// DemoPassword adalah password semua akun contoh.
const DemoPassword = "demo123"

type Store struct {
	mu        sync.RWMutex
	outlets   map[uint]models.Outlet
	users     map[uint]models.User
	products  map[string]models.Product
	mutations []models.Mutation
	audits    []models.AuditLog
	editLogs  []models.MutationEditLog
	nextMutID uint
}

func New() (*repository.Repos, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(DemoPassword), bcrypt.MinCost)
	if err != nil {
		return nil, err
	}
	s := &Store{
		outlets:  map[uint]models.Outlet{},
		users:    map[uint]models.User{},
		products: map[string]models.Product{},
	}
	s.seed(string(hash))
	return &repository.Repos{Users: s, Products: s, Outlets: s, Mutations: s, MutationEdits: s, Audits: s}, nil
}

func uptr(v uint) *uint { return &v }

func (s *Store) seed(hash string) {
	outlets := []struct{ code, name, typ string }{
		{"MK1", "MK 1", "MK"}, {"MK2", "MK 2", "MK"}, {"MK3", "MK 3", "MK"},
		{"MK4", "MK 4", "MK"}, {"MK5", "MK 5", "MK"}, {"MK6", "MK 6", "MK"},
		{"MK7", "MK 7", "MK"}, {"MK8", "MK 8", "MK"},
		{"MINI1", "MINI 1", "MINI"}, {"MINI2", "MINI 2", "MINI"}, {"MINI3", "MINI 3", "MINI"},
	}
	for i, o := range outlets {
		id := uint(i + 1)
		s.outlets[id] = models.Outlet{ID: id, Code: o.code, Name: o.name, Type: o.typ, IsActive: true}
	}

	users := []models.User{
		{ID: 1, Name: "Lina Marlina", Username: "admin.a", Role: models.RoleAdmin},
		{ID: 2, Name: "Admin B", Username: "admin.b", Role: models.RoleAdmin},
		{ID: 3, Name: "Budi Santoso", Username: "manager", Role: models.RoleManager},
		{ID: 4, Name: "Puji Lestari", Username: "puji", Role: models.RolePM, OutletID: uptr(2)},
		{ID: 5, Name: "Rina Wardani", Username: "rina", Role: models.RolePM, OutletID: uptr(5)},
	}
	for _, u := range users {
		u.UserCode = u.Username
		u.PasswordHash = hash
		u.IsActive = true
		s.users[u.ID] = u
	}

	products := []struct{ barcode, code, name string }{
		{"8992222075220", "133601", "PIXY-UVP SUNSC.30G HYDRA CRM SPF50+ PA++"},
		{"8992222075213", "133602", "PIXY-UVP SUNSC.30G AQUA GEL SPF50+ PA+++"},
		{"8992222072779", "130311", "PIXY-CLEAN.EXPRES 150M BRIGHTNG"},
		{"8992222072465", "130169", "PIXY-CLEAN.EXPRES 100M BRIGHTNG"},
		{"8992222072168", "131203", "PIXY-FRS TONER 100M BRIGHTENING"},
		{"8992222072885", "070068", "PIXY-FF 100G BRIGHTENING"},
		{"8992222071192", "133608", "PIXY-CLEANSING EXPRESS BRIGHTENING 100ML"},
		{"8993883801456", "133618", "IMPLORA-SUNSC.50M SPF30 PA+++"},
		{"8997208072590", "132177", "AZARI-CALM M.A SUNS.MOIS 40G SPF35 PA+++"},
		{"8997208072606", "132176", "AZARI-CICAM.BAR SUNS.MOIS 40G SPF35 PA++"},
		{"8997208072361", "133694", "AZARI-HYD.SUNSC GEL 30M SPF45 PA++++"},
		{"8997024490184", "133603", "AZARINE-HYDRASOOTHE SUNSCREEN GEL SPF45 50ML"},
		{"8993137704205", "133604", "WARDAH-LIGHTENING FACE TONER ADVANCED NIACINAMIDE 125ML"},
		{"8993137707442", "133605", "EMINA-BRIGHT STUFF FACE WASH FOR GLOWING SKIN 50ML"},
		{"8993137599023", "133606", "MAKEOVER-ULTRA HI-MATTE LIPSTICK 002 ENVY"},
		{"8997232230157", "133607", "SOMETHINC-10% NIACINAMIDE + MOISTURE BEET SERUM 20ML"},
	}
	for i, p := range products {
		s.products[p.barcode] = models.Product{
			ID: uint(i + 1), Barcode: p.barcode, Code: p.code, Name: p.name, IsActive: true,
		}
	}

	// Riwayat contoh hari ini (WIB).
	seedMut := []struct {
		user    uint
		barcode string
		qty     int
		h, m    int
		ba      string
	}{
		{4, "8992222075220", 4, 17, 16, "SEFFIA"},
		{4, "8992222075213", 4, 17, 16, "SEFFIA"},
		{4, "8992222072779", 4, 17, 17, "SEFFIA"},
		{4, "8992222072465", 3, 17, 19, "SEFFIA"},
		{4, "8992222072168", 2, 17, 19, "SEFFIA"},
		{4, "8992222072885", 2, 17, 20, "SEFFIA"},
		{4, "8993883801456", 3, 17, 57, "ALYA"},
		{4, "8997208072590", 6, 18, 10, "ALYA"},
		{5, "8997208072606", 3, 9, 30, ""}, // outlet lain: tidak boleh tampil di riwayat Puji
	}
	day, _ := clock.DayBounds(clock.Now())
	for _, sm := range seedMut {
		p := s.products[sm.barcode]
		u := s.users[sm.user]
		s.nextMutID++
		m := models.Mutation{
			ID: s.nextMutID, ProductID: uintPtr(p.ID), Barcode: p.Barcode, Code: p.Code,
			ProductName: p.Name, Qty: sm.qty, OutletID: *u.OutletID, UserID: u.ID,
			CreatedAt: day.Add(time.Duration(sm.h)*time.Hour + time.Duration(sm.m)*time.Minute),
		}
		if sm.ba != "" {
			ba := sm.ba
			m.BAName = &ba
		}
		s.mutations = append(s.mutations, m)
	}

	s.seedBulk(hash)
}

func (s *Store) withOutlet(u models.User) *models.User {
	if u.OutletID != nil {
		if o, ok := s.outlets[*u.OutletID]; ok {
			oc := o
			u.Outlet = &oc
		}
	}
	return &u
}

// ---- repository.Users ----

func (s *Store) ByID(id uint) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return s.withOutlet(u), nil
}

func (s *Store) ByUsername(username string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, u := range s.users {
		if u.Username == username {
			return s.withOutlet(u), nil
		}
	}
	return nil, repository.ErrNotFound
}

func (s *Store) ByUserCode(code string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, u := range s.users {
		if u.UserCode == code {
			return s.withOutlet(u), nil
		}
	}
	return nil, repository.ErrNotFound
}

// ---- repository.Products ----

func (s *Store) ByBarcode(barcode string, outletID uint) (*models.Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.products[barcode]
	if !ok || !p.IsActive {
		return nil, repository.ErrNotFound
	}
	return &p, nil
}

// ---- repository.Mutations ----

func (s *Store) ListByOutletBetween(outletID uint, from, to time.Time) ([]models.Mutation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []models.Mutation
	for _, m := range s.mutations {
		if m.OutletID != outletID || m.CreatedAt.Before(from) || !m.CreatedAt.Before(to) {
			continue
		}
		m.Outlet = s.outlets[m.OutletID]
		m.User = s.users[m.UserID]
		out = append(out, m)
	}
	// Terbaru di atas.
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ID > out[j].ID
		}
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}

func (s *Store) Create(m *models.Mutation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextMutID++
	m.ID = s.nextMutID
	if m.CreatedAt.IsZero() {
		m.CreatedAt = clock.Now()
	}
	s.mutations = append(s.mutations, *m)
	return nil
}

func (s *Store) UpdateQty(mutationID, outletID, userID uint, qty int, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.mutations {
		m := &s.mutations[i]
		if m.ID == mutationID && m.OutletID == outletID && m.CreatedAt.Format("2006-01-02") == now.Format("2006-01-02") {
			if m.Qty != qty {
				s.audits = append(s.audits, models.AuditLog{UserID: &userID, OutletID: &outletID, Action: "mutation_qty_update", Detail: fmt.Sprintf("%d: %d -> %d", mutationID, m.Qty, qty), CreatedAt: now})
				s.editLogs = append(s.editLogs, models.MutationEditLog{ID: uint(len(s.editLogs) + 1), MutationID: mutationID, UserID: userID, OldQty: m.Qty, NewQty: qty, CreatedAt: now, User: s.users[userID]})
				m.Qty = qty
			}
			return nil
		}
	}
	return repository.ErrNotFound
}

func (s *Store) ListByMutation(mutationID, outletID uint) ([]models.MutationEditLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []models.MutationEditLog
	for _, v := range s.editLogs {
		for _, m := range s.mutations {
			if v.MutationID == mutationID && m.ID == mutationID && m.OutletID == outletID {
				out = append(out, v)
				break
			}
		}
	}
	return out, nil
}

// ---- repository.Audits ----

func (s *Store) Add(a *models.AuditLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	a.ID = uint(len(s.audits) + 1)
	if a.CreatedAt.IsZero() {
		a.CreatedAt = clock.Now()
	}
	s.audits = append(s.audits, *a)
	return nil
}
