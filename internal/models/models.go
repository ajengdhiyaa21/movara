package models

import "time"

type Role string

const (
	RolePM      Role = "pm"
	RoleManager Role = "manager"
	RoleAdmin   Role = "admin"
)

type Outlet struct {
	ID        uint   `gorm:"primaryKey"`
	Code      string `gorm:"size:20;uniqueIndex;not null"` // MK1, MINI1, ...
	Name      string `gorm:"size:100;not null"`
	Type      string `gorm:"size:10;not null"` // MK | MINI
	IsActive  bool   `gorm:"not null;default:true"`
	CreatedAt time.Time
}

type ProductCategory struct {
	ID        uint   `gorm:"primaryKey"`
	Name      string `gorm:"size:100;uniqueIndex;not null"`
	Slug      string `gorm:"size:100;uniqueIndex;not null"`
	IsActive  bool   `gorm:"not null;default:true"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type User struct {
	ID                 uint   `gorm:"primaryKey"`
	Name               string `gorm:"size:100;not null"`
	UserCode           string `gorm:"size:50;index"`
	Username           string `gorm:"size:50;uniqueIndex;not null"`
	PasswordHash       string `gorm:"size:100;not null"`
	PasswordCiphertext string `gorm:"size:512"`
	Role               Role   `gorm:"size:20;not null;index"`
	OutletID           *uint  `gorm:"index"` // wajib untuk PM
	Outlet             *Outlet
	CategoryID         *uint `gorm:"index"`
	Category           *ProductCategory
	IsActive           bool `gorm:"not null;default:true"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// Barcode & Code bertipe string: ada kode berawalan nol (mis. 070068).
type Product struct {
	ID         uint   `gorm:"primaryKey"`
	Barcode    string `gorm:"size:20;not null;uniqueIndex:idx_product_outlet_barcode,priority:2;uniqueIndex:idx_product_outlet_code,priority:2;uniqueIndex:idx_product_outlet_name,priority:2"`
	Code       string `gorm:"size:30;index;not null;uniqueIndex:idx_product_outlet_code,priority:3"`
	Name       string `gorm:"size:255;not null;uniqueIndex:idx_product_outlet_name,priority:3"`
	IsActive   bool   `gorm:"not null;default:true"`
	OutletID   uint   `gorm:"not null;uniqueIndex:idx_product_outlet_barcode,priority:1;uniqueIndex:idx_product_outlet_code,priority:1;uniqueIndex:idx_product_outlet_name,priority:1"`
	Outlet     Outlet
	CategoryID *uint `gorm:"index"`
	Category   *ProductCategory
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// ProductRequest menampung pengajuan produk baru dari PM sebelum diverifikasi.
// Data ini sengaja dipisahkan dari Product dan Mutation agar belum tampil di
// master data maupun jurnal mutasi sebelum Super Admin menyelesaikannya.
type ProductRequest struct {
	ID          uint      `gorm:"primaryKey"`
	Barcode     string    `gorm:"size:20;not null;index"`
	ProductName string    `gorm:"size:255;not null"`
	Code        string    `gorm:"size:30"`
	CategoryID  *uint     `gorm:"index"`
	Qty         int       `gorm:"not null"`
	BAName      *string   `gorm:"size:100"`
	Direction   string    `gorm:"size:20;not null"`
	OutletID    uint      `gorm:"not null;index"`
	UserID      uint      `gorm:"not null;index"`
	Status      string    `gorm:"size:20;not null;default:pending;index"`
	RequestedAt time.Time `gorm:"index"`
	ReviewedAt  *time.Time
	ReviewedBy  *uint
	Outlet      Outlet
	User        User
	Category    *ProductCategory
}

// Mutation menyimpan snapshot produk agar riwayat tidak berubah
// saat master data diedit.
type Mutation struct {
	ID          uint   `gorm:"primaryKey"`
	ProductID   *uint  `gorm:"index"`
	Barcode     string `gorm:"size:20;not null;index"`
	Code        string `gorm:"size:30;index"`
	ProductName string `gorm:"size:255;not null"`
	CategoryID  *uint  `gorm:"index;index:idx_mut_category_date,priority:1"`
	Category    *ProductCategory
	Direction   string    `gorm:"size:20;not null;default:counter_to_market;index;index:idx_mut_direction_date,priority:1"`
	Status      string    `gorm:"size:20;not null;default:approved;index"`
	Qty         int       `gorm:"not null"`
	BAName      *string   `gorm:"size:100"`
	OutletID    uint      `gorm:"not null;index:idx_mut_outlet_date,priority:1"`
	UserID      uint      `gorm:"not null;index"`
	CreatedAt   time.Time `gorm:"index:idx_mut_outlet_date,priority:2;index:idx_mut_category_date,priority:2;index:idx_mut_direction_date,priority:2"`

	Outlet Outlet
	User   User
}

type MutationEditLog struct {
	ID         uint      `gorm:"primaryKey"`
	MutationID uint      `gorm:"not null;index"`
	UserID     uint      `gorm:"not null;index"`
	OldQty     int       `gorm:"not null"`
	NewQty     int       `gorm:"not null"`
	CreatedAt  time.Time `gorm:"index"`
	User       User
}

type TransferStatus string

const (
	TransferPending  TransferStatus = "pending"
	TransferApproved TransferStatus = "approved"
	TransferRejected TransferStatus = "rejected"
)

type OutletTransferRequest struct {
	ID           uint           `gorm:"primaryKey"`
	UserID       uint           `gorm:"not null;index"`
	FromOutletID uint           `gorm:"not null"`
	ToOutletID   uint           `gorm:"not null"`
	Status       TransferStatus `gorm:"size:20;not null;default:pending;index"`
	RequestedAt  time.Time
	DecidedBy    *uint
	DecidedAt    *time.Time
}

type AuditLog struct {
	ID        uint  `gorm:"primaryKey"`
	UserID    *uint `gorm:"index"`
	OutletID  *uint `gorm:"index"`
	User      *User
	Action    string    `gorm:"size:50;not null"`
	Detail    string    `gorm:"type:text"`
	CreatedAt time.Time `gorm:"index"`
}
