package domain

import (
	"time"

	"gorm.io/gorm"
)

const (
	ExtensionStatusPending  = "pending"
	ExtensionStatusApproved = "approved"
	ExtensionStatusRejected = "rejected"
)

type GTDExtensionRequest struct {
	ID                int64          `gorm:"primaryKey" db:"id" json:"id"`
	GTDID             int64          `gorm:"not null;index" db:"gtd_id" json:"gtd_id"`
	ContractID        int64          `gorm:"not null;index" db:"contract_id" json:"contract_id"`
	InvoiceID         int64          `gorm:"not null;index" db:"invoice_id" json:"invoice_id"`
	CurrentDeadline   *time.Time     `gorm:"type:date" db:"current_deadline" json:"current_deadline,omitempty"`
	RequestedDeadline time.Time      `gorm:"type:date;not null" db:"requested_deadline" json:"requested_deadline"`
	DocumentPath      string         `gorm:"type:text;not null" db:"document_path" json:"document_path"`
	Status            string         `gorm:"type:varchar(50);not null;default:'pending';index" db:"status" json:"status"`
	CreatedBy         string         `gorm:"type:varchar(255);not null" db:"created_by" json:"created_by"`
	ReviewedBy        *string        `gorm:"type:varchar(255)" db:"reviewed_by" json:"reviewed_by,omitempty"`
	ReviewedAt        *time.Time     `gorm:"type:timestamptz" db:"reviewed_at" json:"reviewed_at,omitempty"`
	Comment           string         `gorm:"type:text;default:''" db:"comment" json:"comment,omitempty"`
	CreatedAt         time.Time      `gorm:"not null;default:now()" db:"created_at" json:"created_at"`
	UpdatedAt         time.Time      `gorm:"not null;default:now()" db:"updated_at" json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
	Creator           *UserBrief     `gorm:"-" json:"creator,omitempty"`
	Reviewer          *UserBrief     `gorm:"-" json:"reviewer,omitempty"`
}

func (GTDExtensionRequest) TableName() string {
	return "gtd_extension_requests"
}
