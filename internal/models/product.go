package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ProductStatus definisce lo stato del prodotto
type ProductStatus string

const (
	ProductStatusActive   ProductStatus = "active"
	ProductStatusInactive ProductStatus = "inactive"
	ProductStatusDraft    ProductStatus = "draft"
)

// Product rappresenta un prodotto nel catalogo
type Product struct {
	ID              uuid.UUID     `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name            string        `gorm:"not null" json:"name"`
	Description     string        `gorm:"type:text" json:"description"`
	SKU             string        `gorm:"uniqueIndex;not null" json:"sku"`
	Barcode         string        `gorm:"index" json:"barcode"`
	Category        string        `gorm:"index" json:"category"`
	Brand           string        `json:"brand"`
	BasePrice       float64       `gorm:"not null" json:"base_price"`
	Currency        string        `gorm:"default:USD" json:"currency"`
	Status          ProductStatus `gorm:"type:varchar(20);default:'active'" json:"status"`
	ImageURL        string        `json:"image_url"`
	Weight          float64       `json:"weight"`
	Dimensions      string        `json:"dimensions"`
	IsAuthenticatable bool        `gorm:"default:true" json:"is_authenticatable"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`

	// Relazioni
	InventoryItems  []InventoryItem `gorm:"foreignKey:ProductID" json:"inventory_items,omitempty"`
	QRCodes         []QRCode        `gorm:"foreignKey:ProductID" json:"qr_codes,omitempty"`
	OrderItems      []OrderItem      `gorm:"foreignKey:ProductID" json:"order_items,omitempty"`
	PriceQuotes     []PriceQuote     `gorm:"foreignKey:ProductID" json:"price_quotes,omitempty"`
}

// InventoryItem rappresenta un item specifico nell'inventario con batch e seriale
type InventoryItem struct {
	ID              uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ProductID       uuid.UUID `gorm:"type:uuid;not null;index" json:"product_id"`
	BatchNumber     string    `gorm:"not null;index" json:"batch_number"`
	SerialNumber    string    `gorm:"uniqueIndex;not null" json:"serial_number"`
	DistributorID   *uuid.UUID `gorm:"type:uuid;index" json:"distributor_id"`
	RegionID        *uuid.UUID `gorm:"type:uuid;index" json:"region_id"`
	QRCodeID        *uuid.UUID `gorm:"type:uuid;uniqueIndex" json:"qr_code_id"`
	Quantity        int       `gorm:"default:1" json:"quantity"`
	CostPrice       float64   `json:"cost_price"`
	Status          string    `gorm:"default:'in_stock'" json:"status"` // in_stock, sold, reserved, damaged
	Location        string    `json:"location"`
	ReceivedAt      time.Time `json:"received_at"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// Relazioni
	Product         Product   `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	QRCode          *QRCode   `gorm:"foreignKey:QRCodeID" json:"qr_code,omitempty"`
}

// QRCode rappresenta un QR code crittografato per un prodotto
type QRCode struct {
	ID              uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ProductID       uuid.UUID `gorm:"type:uuid;not null;index" json:"product_id"`
	InventoryItemID *uuid.UUID `gorm:"type:uuid;uniqueIndex" json:"inventory_item_id"`
	EncryptedToken  string    `gorm:"uniqueIndex;not null" json:"encrypted_token"`
	DigitalSignature string   `gorm:"not null" json:"digital_signature"`
	BatchNumber     string    `gorm:"index" json:"batch_number"`
	SerialNumber    string    `gorm:"index" json:"serial_number"`
	DistributorID   *uuid.UUID `gorm:"type:uuid;index" json:"distributor_id"`
	RegionID        *uuid.UUID `gorm:"type:uuid;index" json:"region_id"`
	IsActive        bool      `gorm:"default:true" json:"is_active"`
	IsVerified      bool      `gorm:"default:false" json:"is_verified"`
	DisplayInfo     string    `gorm:"type:jsonb" json:"display_info"` // Info personalizzabili dall'admin
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	VerifiedAt      *time.Time `json:"verified_at"`

	// Relazioni
	Product         Product         `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	ScanHistory     []QRScanHistory `gorm:"foreignKey:QRCodeID" json:"scan_history,omitempty"`
}

// QRScanHistory registra tutte le scansioni QR
type QRScanHistory struct {
	ID              uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	QRCodeID        uuid.UUID `gorm:"type:uuid;not null;index" json:"qr_code_id"`
	UserID          *uuid.UUID `gorm:"type:uuid;index" json:"user_id"` // Nullable per scansioni guest
	DeviceID        string    `gorm:"index" json:"device_id"`
	DeviceType      string    `json:"device_type"` // ios, android, web
	LocationLat     float64   `json:"location_lat"`
	LocationLng     float64   `json:"location_lng"`
	LocationAddress string    `json:"location_address"`
	IsAuthentic     bool      `gorm:"default:true" json:"is_authentic"`
	IsFakeAlert     bool      `gorm:"default:false" json:"is_fake_alert"`
	ScanMethod      string    `json:"scan_method"` // camera, file_upload
	CreatedAt       time.Time `json:"created_at"`

	// Relazioni
	QRCode          QRCode    `gorm:"foreignKey:QRCodeID" json:"qr_code,omitempty"`
	User            *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// PriceQuote rappresenta una richiesta di preventivo da un distributore
type PriceQuote struct {
	ID              uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	DistributorID   uuid.UUID `gorm:"type:uuid;not null;index" json:"distributor_id"`
	ProductID       uuid.UUID `gorm:"type:uuid;not null;index" json:"product_id"`
	Quantity        int       `gorm:"not null" json:"quantity"`
	RequestedPrice  float64   `json:"requested_price"`
	QuotedPrice     *float64  `json:"quoted_price"`
	Status          string    `gorm:"default:'pending'" json:"status"` // pending, approved, rejected
	Notes           string    `gorm:"type:text" json:"notes"`
	QuotedBy        *uuid.UUID `gorm:"type:uuid" json:"quoted_by"`
	QuotedAt        *time.Time `json:"quoted_at"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// Relazioni
	Distributor     Distributor `gorm:"foreignKey:DistributorID" json:"distributor,omitempty"`
	Product         Product     `gorm:"foreignKey:ProductID" json:"product,omitempty"`
}

// BeforeCreate hooks
func (p *Product) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

func (i *InventoryItem) BeforeCreate(tx *gorm.DB) error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	return nil
}

func (q *QRCode) BeforeCreate(tx *gorm.DB) error {
	if q.ID == uuid.Nil {
		q.ID = uuid.New()
	}
	return nil
}

func (q *QRScanHistory) BeforeCreate(tx *gorm.DB) error {
	if q.ID == uuid.Nil {
		q.ID = uuid.New()
	}
	return nil
}

func (pq *PriceQuote) BeforeCreate(tx *gorm.DB) error {
	if pq.ID == uuid.Nil {
		pq.ID = uuid.New()
	}
	return nil
}
