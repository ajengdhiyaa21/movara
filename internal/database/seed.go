package database

import (
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"movara/internal/models"
)

func Seed(db *gorm.DB, adminPassword string) error {
	if adminPassword == "" {
		adminPassword = "admin123!"
	}
	outlets := []models.Outlet{
		{Code: "MK1", Name: "MK 1", Type: "MK"},
		{Code: "MK2", Name: "MK 2", Type: "MK"},
		{Code: "MK3", Name: "MK 3", Type: "MK"},
		{Code: "MK4", Name: "MK 4", Type: "MK"},
		{Code: "MK5", Name: "MK 5", Type: "MK"},
		{Code: "MK6", Name: "MK 6", Type: "MK"},
		{Code: "MK7", Name: "MK 7", Type: "MK"},
		{Code: "MK8", Name: "MK 8", Type: "MK"},
		{Code: "MINI1", Name: "MINI 1", Type: "MINI"},
		{Code: "MINI2", Name: "MINI 2", Type: "MINI"},
		{Code: "MINI3", Name: "MINI 3", Type: "MINI"},
	}
	for _, o := range outlets {
		o.IsActive = true
		if err := db.Where("code = ?", o.Code).FirstOrCreate(&o).Error; err != nil {
			return err
		}
	}
	// Normalisasi label outlet lama agar MK2 tidak lagi menampilkan keterangan "Pusat".
	if err := db.Model(&models.Outlet{}).Where("code = ?", "MK2").Update("name", "MK 2").Error; err != nil {
		return err
	}
	categories := []models.ProductCategory{
		{Name: "Obat", Slug: "obat", IsActive: true},
		{Name: "Kosmetik", Slug: "kosmetik", IsActive: true},
	}
	for _, category := range categories {
		if err := db.Where("slug = ?", category.Slug).FirstOrCreate(&category).Error; err != nil {
			return err
		}
	}
	var cosmetic models.ProductCategory
	if err := db.Where("slug = ?", "kosmetik").First(&cosmetic).Error; err != nil {
		return err
	}
	var obat models.ProductCategory
	if err := db.Where("slug = ?", "obat").First(&obat).Error; err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	admins := []models.User{
		{Name: "Admin A", Username: "admin.a"},
		{Name: "Admin B", Username: "admin.b"},
	}
	for _, a := range admins {
		a.PasswordHash = string(hash)
		a.Role = models.RoleAdmin
		a.IsActive = true
		// FirstOrCreate: tidak menimpa akun yang sudah ada
		if err := db.Where("username = ?", a.Username).FirstOrCreate(&a).Error; err != nil {
			return err
		}
	}
	var managerOutlet uint = 1
	users := []models.User{
		{Name: "Budi Santoso", Username: "manager", Role: models.RoleManager, IsActive: true},
		{Name: "PM Kosmetik", Username: "pmkosmetik", Role: models.RolePM, OutletID: &managerOutlet, CategoryID: &cosmetic.ID, IsActive: true},
		{Name: "PM Obat", Username: "pmobat", Role: models.RolePM, OutletID: &managerOutlet, CategoryID: &obat.ID, IsActive: true},
	}
	for _, u := range users {
		u.PasswordHash = string(hash)
		if err := db.Where("username = ?", u.Username).FirstOrCreate(&u).Error; err != nil {
			return err
		}
	}
	if err := db.Model(&models.User{}).Where("username = ?", "manager").Update("name", "Manager").Error; err != nil {
		return err
	}
	// Migrasikan akun demo yang masih memakai password lama. Password yang
	// sudah diubah Admin tidak disentuh lagi pada startup berikutnya.
	for _, username := range []string{"admin.a", "admin.b", "manager", "pmkosmetik", "pmobat"} {
		var existing models.User
		if err := db.Where("username = ?", username).First(&existing).Error; err != nil {
			continue
		}
		if bcrypt.CompareHashAndPassword([]byte(existing.PasswordHash), []byte("pm12345!")) == nil || bcrypt.CompareHashAndPassword([]byte(existing.PasswordHash), []byte("admin123!")) == nil {
			if err := db.Model(&existing).Update("password_hash", string(hash)).Error; err != nil {
				return err
			}
		}
	}
	products := []models.Product{
		{Barcode: "8992222075220", Code: "133601", Name: "PIXY-UVP SUNSC.30G HYDRA CRM SPF50+ PA++", CategoryID: &cosmetic.ID, IsActive: true, OutletID: 2},
		{Barcode: "8992222075213", Code: "133602", Name: "PIXY-UVP SUNSC.30G AQUA GEL SPF50+ PA+++", CategoryID: &cosmetic.ID, IsActive: true, OutletID: 2},
		{Barcode: "8992222072779", Code: "130311", Name: "PIXY-CLEAN.EXPRES 150M BRIGHTNG", CategoryID: &cosmetic.ID, IsActive: true, OutletID: 2},
	}
	for _, p := range products {
		if err := db.Where("barcode = ?", p.Barcode).FirstOrCreate(&p).Error; err != nil {
			return err
		}
	}
	return nil
}
