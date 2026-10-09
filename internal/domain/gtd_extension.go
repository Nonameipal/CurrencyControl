package domain

import (
	"time"

	"gorm.io/gorm"
)

const (
	ExtensionStatusPendingCurrencyControl  = "pending_currency_control"
	ExtensionStatusPendingCompliance       = "pending_compliance"
	ExtensionStatusRevisionRequired        = "revision_required"
	ExtensionStatusRejectedCurrencyControl = "rejected_currency_control"
	ExtensionStatusRejectedCompliance      = "rejected_compliance"
	ExtensionStatusApproved                = "approved"

	// Сохраняем для обратной совместимости
	ExtensionStatusPending  = ExtensionStatusPendingCurrencyControl
	ExtensionStatusRejected = ExtensionStatusRejectedCurrencyControl
)

type GTDExtensionRequest struct {
	ID                         int64          `gorm:"primaryKey" db:"id" json:"id"`
	GTDID                      int64          `gorm:"not null;index" db:"gtd_id" json:"gtd_id"`
	ContractID                 int64          `gorm:"not null;index" db:"contract_id" json:"contract_id"`
	InvoiceID                  int64          `gorm:"not null;index" db:"invoice_id" json:"invoice_id"`
	CurrentDeadline            *time.Time     `gorm:"type:date" db:"current_deadline" json:"current_deadline,omitempty"`
	RequestedDeadline          time.Time      `gorm:"type:date;not null" db:"requested_deadline" json:"requested_deadline"`
	DocumentPath               string         `gorm:"type:text;not null" db:"document_path" json:"document_path"`
	Status                     string         `gorm:"type:varchar(50);not null;default:'pending_currency_control';index" db:"status" json:"status"`
	CreatedBy                  string         `gorm:"type:varchar(255);not null" db:"created_by" json:"created_by"`
	CurrencyControlDecision    string         `gorm:"type:varchar(50);default:''" db:"currency_control_decision" json:"currency_control_decision,omitempty"`
	CurrencyControlComment     string         `gorm:"type:text;default:''" db:"currency_control_comment" json:"currency_control_comment,omitempty"`
	CurrencyControlReviewedBy  string         `gorm:"type:varchar(255);default:''" db:"currency_control_reviewed_by" json:"currency_control_reviewed_by,omitempty"`
	CurrencyControlReviewedAt  *time.Time     `gorm:"type:timestamptz" db:"currency_control_reviewed_at" json:"currency_control_reviewed_at,omitempty"`
	ComplianceDecision         string         `gorm:"type:varchar(50);default:''" db:"compliance_decision" json:"compliance_decision,omitempty"`
	ComplianceComment          string         `gorm:"type:text;default:''" db:"compliance_comment" json:"compliance_comment,omitempty"`
	ComplianceReviewedBy       string         `gorm:"type:varchar(255);default:''" db:"compliance_reviewed_by" json:"compliance_reviewed_by,omitempty"`
	ComplianceReviewedAt       *time.Time     `gorm:"type:timestamptz" db:"compliance_reviewed_at" json:"compliance_reviewed_at,omitempty"`
	RejectionReason            string         `gorm:"type:text;default:''" db:"rejection_reason" json:"rejection_reason,omitempty"`
	ReviewedBy                 *string        `gorm:"type:varchar(255)" db:"reviewed_by" json:"reviewed_by,omitempty"`
	ReviewedAt                 *time.Time     `gorm:"type:timestamptz" db:"reviewed_at" json:"reviewed_at,omitempty"`
	Comment                    string         `gorm:"type:text;default:''" db:"comment" json:"comment,omitempty"`
	CreatedAt                  time.Time      `gorm:"not null;default:now()" db:"created_at" json:"created_at"`
	UpdatedAt                  time.Time      `gorm:"not null;default:now()" db:"updated_at" json:"updated_at"`
	DeletedAt                  gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
	Creator                    *UserBrief     `gorm:"-" json:"creator,omitempty"`
	Reviewer                   *UserBrief     `gorm:"-" json:"reviewer,omitempty"`
	CurrencyControlReviewer    *UserBrief     `gorm:"-" json:"currency_control_reviewer,omitempty"`
	ComplianceReviewer         *UserBrief     `gorm:"-" json:"compliance_reviewer,omitempty"`
}

func (GTDExtensionRequest) TableName() string {
	return "gtd_extension_requests"
}
