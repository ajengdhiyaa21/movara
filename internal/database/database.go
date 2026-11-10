package database

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"movara/internal/models"
)

func Connect(dsn string) (*gorm.DB, error) {
	return gorm.Open(mysql.Open(dsn), &gorm.Config{})
}

// Migrate memakai AutoMigrate agar cepat saat awal pengembangan.
// Untuk produksi, ganti dengan golang-migrate (file SQL berversi).
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&models.Outlet{}); err != nil {
		return err
	}
	if err := db.AutoMigrate(&models.ProductCategory{}); err != nil {
		return err
	}
	// Kompatibilitas database lama: products sebelumnya belum memiliki outlet_id.
	// Tambahkan nullable dulu, isi data lama ke outlet pertama, baru AutoMigrate
	// memasang NOT NULL dan foreign key dengan aman.
	if db.Migrator().HasTable(&models.Product{}) && !db.Migrator().HasColumn(&models.Product{}, "OutletID") {
		if err := db.Exec("ALTER TABLE products ADD COLUMN outlet_id BIGINT UNSIGNED NULL").Error; err != nil {
			return err
		}
	}
	if db.Migrator().HasTable(&models.Product{}) && db.Migrator().HasColumn(&models.Product{}, "OutletID") {
		if db.Migrator().HasIndex(&models.Product{}, "uni_products_barcode") {
			_ = db.Migrator().DropIndex(&models.Product{}, "uni_products_barcode")
		}
		var first models.Outlet
		if err := db.Order("id").First(&first).Error; err == nil {
			if err := db.Exec("UPDATE products SET outlet_id = ? WHERE outlet_id IS NULL OR outlet_id = 0", first.ID).Error; err != nil {
				return err
			}
		}
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.Product{},
		&models.ProductRequest{},
		&models.Mutation{},
		&models.MutationEditLog{},
		&models.OutletTransferRequest{},
		&models.AuditLog{},
	); err != nil {
		return err
	}
	return db.Exec("UPDATE users SET user_code = username WHERE user_code IS NULL OR user_code = ''").Error
}
