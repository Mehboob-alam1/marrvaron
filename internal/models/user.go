package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

// UserRole definisce i ruoli disponibili nel sistema
type UserRole string

const (
	RoleSuperAdmin  UserRole = "super_admin"
	RoleAdmin       UserRole = "admin"
	RoleDistributor UserRole = "distributor"
	RoleUser        UserRole = "user"     // default signup: common end user
	RoleCustomer    UserRole = "customer" // legacy end user (treated like user)
	RoleCourier     UserRole = "courier"
	RoleVendor      UserRole = "vendor"
)

// User rappresenta un utente nel sistema
type User struct {
	ID                uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Email             string    `gorm:"uniqueIndex;not null" json:"email"`
	Phone             string    `gorm:"index" json:"phone"`
	PasswordHash      string    `json:"-"` // empty until password step of registration
	FullName          string    `json:"full_name"`
	FirstName         string    `json:"first_name"`
	LastName          string    `json:"last_name"`
	StreetAddress     string    `json:"street_address"`
	City              string    `json:"city"`
	StateRegion       string    `json:"state"`
	PostCode          string    `json:"post_code"`
	Country           string    `json:"country"`
	// Roles: all roles this user may use; Role is the active context for JWT / middleware
	Roles             pq.StringArray `gorm:"type:text[]" json:"roles"`
	Role              UserRole  `gorm:"type:varchar(30);not null;index" json:"active_role"`
	ReferralCode      *string   `gorm:"uniqueIndex" json:"referral_code,omitempty"` // set when registration completes
	ReferredByUserID  *uuid.UUID `gorm:"type:uuid" json:"referred_by_user_id,omitempty"`
	RegistrationComplete bool   `gorm:"default:true" json:"registration_complete"`
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
	u.normalizeRolesAndActive()
	return nil
}

// AfterFind backfills roles from legacy single Role column when roles array is empty
func (u *User) AfterFind(tx *gorm.DB) error {
	if len(u.Roles) == 0 && u.Role != "" {
		u.Roles = pq.StringArray{string(u.Role)}
	}
	if u.Role == "" && len(u.Roles) > 0 {
		u.Role = UserRole(u.Roles[0])
	}
	return nil
}

func (u *User) normalizeRolesAndActive() {
	if len(u.Roles) == 0 {
		if u.Role != "" {
			u.Roles = pq.StringArray{string(u.Role)}
			return
		}
		u.Role = RoleUser
		u.Roles = pq.StringArray{string(RoleUser)}
		return
	}
	if u.Role == "" {
		u.Role = UserRole(u.Roles[0])
		return
	}
	for _, x := range u.Roles {
		if UserRole(x) == u.Role {
			return
		}
	}
	u.Role = UserRole(u.Roles[0])
}

// HasRole reports whether the user may assume this role (switch context to it)
func (u *User) HasRole(r UserRole) bool {
	for _, x := range u.Roles {
		if UserRole(x) == r {
			return true
		}
	}
	return false
}

// RolesAsStrings returns roles as a plain []string for JWT / JSON
func (u *User) RolesAsStrings() []string {
	out := make([]string, 0, len(u.Roles))
	for _, x := range u.Roles {
		out = append(out, x)
	}
	if len(out) == 0 && u.Role != "" {
		return []string{string(u.Role)}
	}
	return out
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
