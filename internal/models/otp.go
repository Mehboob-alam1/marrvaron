package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OTPRecord stores OTP for verification when Redis is not used
type OTPRecord struct {
	ID         uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Identifier string         `gorm:"index;not null" json:"identifier"` // email or phone
	OTP        string         `gorm:"not null" json:"-"`
	ExpiresAt  time.Time      `gorm:"not null;index" json:"expires_at"`
	CreatedAt  time.Time      `json:"created_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

func (o *OTPRecord) BeforeCreate(tx *gorm.DB) error {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	return nil
}
