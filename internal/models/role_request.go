package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RoleRequestStatus is the moderation state of a role upgrade request.
type RoleRequestStatus string

const (
	RoleRequestPending   RoleRequestStatus = "pending"
	RoleRequestApproved  RoleRequestStatus = "approved"
	RoleRequestRejected  RoleRequestStatus = "rejected"
)

// RoleRequest is a user's request for an elevated role (approved by admin).
type RoleRequest struct {
	ID             uuid.UUID         `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID         uuid.UUID         `gorm:"type:uuid;not null;index" json:"user_id"`
	RequestedRole  UserRole          `gorm:"type:varchar(30);not null" json:"requested_role"`
	Status         RoleRequestStatus `gorm:"type:varchar(20);not null;default:'pending';index" json:"status"`
	ReviewedBy     *uuid.UUID        `gorm:"type:uuid" json:"reviewed_by,omitempty"`
	ReviewedAt     *time.Time        `json:"reviewed_at,omitempty"`
	AdminNote      string            `gorm:"type:text" json:"admin_note,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`

	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (r *RoleRequest) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	if r.Status == "" {
		r.Status = RoleRequestPending
	}
	return nil
}
