package memory

import (
	"math/rand"
	"sort"
	"time"

	"movara/internal/clock"
	"movara/internal/models"
)

// seedBulk menambah PM untuk semua outlet dan ~45 hari riwayat mutasi
// agar halaman Rekap (pagination, filter, perbandingan bulan lalu) terlihat nyata.
// Acak dengan seed tetap, jadi hasilnya konsisten setiap kali server dijalankan.
func (s *Store) seedBulk(hash string) {
	pms := []struct {
		name, username string
		outlet         uint
	}{
		{"Dewi Anggraini", "dewi", 1}, {"Sari Utami", "sari", 3}, {"Nanda Prasetyo", "nanda", 4},
		{"Wulan Sari", "wulan", 6}, {"Eko Wahyudi", "eko", 7}, {"Fitri Handayani", "fitri", 8},
		{"Yoga Pratama", "yoga", 9}, {"Citra Dewi", "citra", 10}, {"Bagas Saputra", "bagas", 11},
	}
	userByOutlet := map[uint]uint{2: 4, 5: 5}
	nextID := uint(6)
	for _, p := range pms {
		s.users[nextID] = models.User{
			ID: nextID, Name: p.name, Username: p.username, PasswordHash: hash,
			Role: models.RolePM, OutletID: uptr(p.outlet), IsActive: true,
		}
		userByOutlet[p.outlet] = nextID
		nextID++
	}

	barcodes := make([]string, 0, len(s.products))
	for b := range s.products {
		barcodes = append(barcodes, b)
	}
	sort.Strings(barcodes)

	rng := rand.New(rand.NewSource(42))
	bas := []string{"SEFFIA", "ALYA", "NISA", "RIRIN", "DINDA", "", ""}
	now := clock.Now()
	today, _ := clock.DayBounds(now)

	for d := 0; d <= 45; d++ {
		day := today.AddDate(0, 0, -d)
		for outletID := uint(1); outletID <= 11; outletID++ {
			if d == 0 && outletID == 2 {
				continue // MK 2 hari ini sudah punya data contoh tetap (riwayat Puji)
			}
			n := 2 + rng.Intn(4)
			for i := 0; i < n; i++ {
				minutes := 9*60 + rng.Intn(12*60) // 09:00 - 20:59
				p := s.products[barcodes[rng.Intn(len(barcodes))]]
				qty := 1 + rng.Intn(8)
				ba := bas[rng.Intn(len(bas))]
				at := day.Add(time.Duration(minutes) * time.Minute)
				if at.After(now) {
					continue // jangan buat data di masa depan
				}
				s.nextMutID++
				m := models.Mutation{
					ID: s.nextMutID, ProductID: uintPtr(p.ID), Barcode: p.Barcode, Code: p.Code,
					ProductName: p.Name, Qty: qty, OutletID: outletID,
					UserID: userByOutlet[outletID], CreatedAt: at,
				}
				if ba != "" {
					b := ba
					m.BAName = &b
				}
				s.mutations = append(s.mutations, m)
			}
		}
	}
}
