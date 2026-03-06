package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserRole definisce i ruoli disponibili nel sistema
type UserRole string

const (
	RoleSuperAdmin  UserRole = "super_admin"
	RoleAdmin       UserRole = "admin"
	RoleDistributor UserRole = "distributor"
	RoleCustomer    UserRole = "customer"
	RoleCourier     UserRole = "courier"
)

// User rappresenta un utente nel sistema
type User struct {
	ID                uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Email             string    `gorm:"uniqueIndex;not null" json:"email"`
	Phone             string    `gorm:"index" json:"phone"`
	PasswordHash      string    `gorm:"not null" json:"-"`
	FirstName         string    `json:"first_name"`
	LastName          string    `json:"last_name"`
	Role              UserRole  `gorm:"type:varchar(20);not null;index" json:"role"`
	IsActive          bool      `gorm:"default:true" json:"is_active"`
	IsEmailVerified   bool      `gorm:"default:false" json:"is_email_verified"`
	IsPhoneVerified   bool      `gorm:"default:false" json:"is_phone_verified"`
	MarketingOptIn    bool      `gorm:"default:false" json:"marketing_opt_in"`
	LastLoginAt       *time.Time `json:"last_login_at"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`

	// Relazioni
	Orders            []Order           `gorm:"foreignKey:UserID" json:"orders,omitempty"`
	DistributorInfo   *Distributor      `gorm:"foreignKey:UserID" json:"distributor_info,omitempty"`
	QRScanHistory     []QRScanHistory   `gorm:"foreignKey:UserID" json:"qr_scan_history,omitempty"`
}

// Distributor contiene informazioni specifiche per i distributori
type Distributor struct {
	ID              uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID          uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"user_id"`
	BusinessName    string    `json:"business_name"`
	TaxID           string    `gorm:"uniqueIndex" json:"tax_id"`
	RegionID        *uuid.UUID `gorm:"type:uuid" json:"region_id"`
	RegionName      string    `json:"region_name"`
	Address         string    `json:"address"`
	City            string    `json:"city"`
	Country         string    `json:"country"`
	PostalCode      string    `json:"postal_code"`
	IsApproved      bool      `gorm:"default:false" json:"is_approved"`
	ApprovedAt      *time.Time `json:"approved_at"`
	ApprovedBy      *uuid.UUID `gorm:"type:uuid" json:"approved_by"`
	LoyaltyPoints   int       `gorm:"default:0" json:"loyalty_points"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// Relazioni
	User            User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Orders          []Order   `gorm:"foreignKey:DistributorID" json:"orders,omitempty"`
	PriceQuotes     []PriceQuote `gorm:"foreignKey:DistributorID" json:"price_quotes,omitempty"`
}

// AdminPermission definisce i permessi per gli admin
type AdminPermission struct {
	ID              uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	AdminID         uuid.UUID `gorm:"type:uuid;not null;index" json:"admin_id"`
	CanUpdateInventory bool   `gorm:"default:false" json:"can_update_inventory"`
	CanGenerateQR   bool      `gorm:"default:false" json:"can_generate_qr"`
	CanManageOrders bool      `gorm:"default:true" json:"can_manage_orders"`
	CanManageUsers  bool      `gorm:"default:false" json:"can_manage_users"`
	CanSendPromotions bool    `gorm:"default:false" json:"can_send_promotions"`
	CanViewRawStore bool      `gorm:"default:false" json:"can_view_raw_store"`
	CanEditRawStore bool      `gorm:"default:false" json:"can_edit_raw_store"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	Admin           User      `gorm:"foreignKey:AdminID" json:"admin,omitempty"`
}

// BeforeCreate hook per generare UUID
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

func (d *Distributor) BeforeCreate(tx *gorm.DB) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return nil
}

func (a *AdminPermission) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}
