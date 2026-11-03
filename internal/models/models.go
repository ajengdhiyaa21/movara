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
	ID           uint   `gorm:"primaryKey"`
	Name         string `gorm:"size:100;not null"`
	Username     string `gorm:"size:50;uniqueIndex;not null"`
	PasswordHash string `gorm:"size:100;not null"`
	Role         Role   `gorm:"size:20;not null;index"`
	OutletID     *uint  `gorm:"index"` // wajib untuk PM
	Outlet       *Outlet
	CategoryID   *uint `gorm:"index"`
	Category     *ProductCategory
	IsActive     bool `gorm:"not null;default:true"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Barcode & Code bertipe string: ada kode berawalan nol (mis. 070068).
type Product struct {
	ID         uint   `gorm:"primaryKey"`
	Barcode    string `gorm:"size:20;not null;uniqueIndex:idx_product_outlet_barcode,priority:2"`
	Code       string `gorm:"size:30;index;not null"`
	Name       string `gorm:"size:255;not null"`
	IsActive   bool   `gorm:"not null;default:true"`
	OutletID   uint   `gorm:"not null;uniqueIndex:idx_product_outlet_barcode,priority:1"`
	Outlet     Outlet
	CategoryID *uint `gorm:"index"`
	Category   *ProductCategory
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Mutation menyimpan snapshot produk agar riwayat tidak berubah
// saat master data diedit.
type Mutation struct {
	ID          uint   `gorm:"primaryKey"`
	ProductID   *uint  `gorm:"index"`
	Barcode     string `gorm:"size:20;not null;index"`
	Code        string `gorm:"size:30;index"`
	ProductName string `gorm:"size:255;not null"`
	CategoryID  *uint  `gorm:"index"`
	Category    *ProductCategory
	Direction   string    `gorm:"size:20;not null;default:counter_to_market;index"`
	Status      string    `gorm:"size:20;not null;default:approved;index"`
	Qty         int       `gorm:"not null"`
	BAName      *string   `gorm:"size:100"`
	OutletID    uint      `gorm:"not null;index:idx_mut_outlet_date,priority:1"`
	UserID      uint      `gorm:"not null;index"`
	CreatedAt   time.Time `gorm:"index:idx_mut_outlet_date,priority:2"`

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
	ID        uint      `gorm:"primaryKey"`
	UserID    *uint     `gorm:"index"`
	OutletID  *uint     `gorm:"index"`
	Action    string    `gorm:"size:50;not null"`
	Detail    string    `gorm:"type:text"`
	CreatedAt time.Time `gorm:"index"`
}

// ChatConversation stores one shared Admin-PM conversation.
type ChatConversation struct {
	ID        uint      `gorm:"primaryKey"`
	PMUserID  uint      `gorm:"not null;uniqueIndex"`
	CreatedAt time.Time `gorm:"precision:6;not null"`
	UpdatedAt time.Time `gorm:"precision:6;not null"`
}

// ChatMessage is immutable after creation in the MVP.
type ChatMessage struct {
	ID              uint      `gorm:"primaryKey"`
	ConversationID  uint      `gorm:"not null;index:idx_chat_messages_conversation_id_id,priority:1;uniqueIndex:idx_chat_messages_client_id,priority:1"`
	SenderUserID    uint      `gorm:"not null;index"`
	ClientMessageID string    `gorm:"size:100;not null;uniqueIndex:idx_chat_messages_client_id,priority:2"`
	Body            string    `gorm:"size:3000;not null"`
	CreatedAt       time.Time `gorm:"precision:6;not null;index:idx_chat_messages_conversation_id_id,priority:2"`
}

// ChatReadState stores an individual user's monotonic read cursor.
type ChatReadState struct {
	ID                uint       `gorm:"primaryKey"`
	ConversationID    uint       `gorm:"not null;uniqueIndex:idx_chat_read_states_conversation_user,priority:1"`
	UserID            uint       `gorm:"not null;uniqueIndex:idx_chat_read_states_conversation_user,priority:2;index"`
	LastReadMessageID *uint      `gorm:"index"`
	LastReadAt        *time.Time `gorm:"precision:6"`
	CreatedAt         time.Time  `gorm:"precision:6;not null"`
	UpdatedAt         time.Time  `gorm:"precision:6;not null"`
}
