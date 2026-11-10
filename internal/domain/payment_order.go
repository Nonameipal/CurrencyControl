package domain

import (
	"time"

	"gorm.io/gorm"
)

type PaymentOrder struct {
	ID                        int64          `gorm:"primaryKey" db:"id" json:"id"`
	ContractID                int64          `gorm:"not null;index" db:"contract_id" json:"contract_id"`
	InvoiceID                 int64          `gorm:"not null;index" db:"invoice_id" json:"invoice_id"`
	ValueDate                 time.Time      `gorm:"type:date;not null" db:"value_date" json:"value_date"`
	Amount                    float64        `gorm:"type:decimal(18,2);not null" db:"amount" json:"amount"`
	Currency                  string         `gorm:"type:varchar(10);not null" db:"currency" json:"currency"`
	SenderName                string         `gorm:"type:varchar(255);default:''" db:"sender_name" json:"sender_name"`
	ReceiverName              string         `gorm:"type:varchar(255);not null" db:"receiver_name" json:"receiver_name"`
	ReceiverBank              string         `gorm:"type:varchar(255);not null" db:"receiver_bank" json:"receiver_bank"`
	ReceiverCountry           string         `gorm:"type:varchar(255);not null" db:"receiver_country" json:"receiver_country"`
	PaymentPurpose            string         `gorm:"type:text;not null" db:"payment_purpose" json:"payment_purpose"`
	CreatedBy                 string         `gorm:"type:varchar(255);not null" db:"created_by" json:"created_by"`
	DocumentPath              *string        `db:"document_path" json:"document_path,omitempty"`
	ApprovalStatus            string         `gorm:"type:varchar(50);not null;default:'pending_currency_control';index" db:"approval_status" json:"approval_status"`
	CurrencyControlDecision   string         `gorm:"type:varchar(50);default:''" db:"currency_control_decision" json:"currency_control_decision,omitempty"`
	CurrencyControlComment    string         `gorm:"type:text;default:''" db:"currency_control_comment" json:"currency_control_comment,omitempty"`
	CurrencyControlReviewedBy string         `gorm:"type:varchar(255);default:''" db:"currency_control_reviewed_by" json:"currency_control_reviewed_by,omitempty"`
	CurrencyControlReviewedAt *time.Time     `gorm:"type:timestamptz" db:"currency_control_reviewed_at" json:"currency_control_reviewed_at,omitempty"`
	ComplianceDecision        string         `gorm:"type:varchar(50);default:''" db:"compliance_decision" json:"compliance_decision,omitempty"`
	ComplianceComment         string         `gorm:"type:text;default:''" db:"compliance_comment" json:"compliance_comment,omitempty"`
	ComplianceReviewedBy      string         `gorm:"type:varchar(255);default:''" db:"compliance_reviewed_by" json:"compliance_reviewed_by,omitempty"`
	ComplianceReviewedAt      *time.Time     `gorm:"type:timestamptz" db:"compliance_reviewed_at" json:"compliance_reviewed_at,omitempty"`
	RejectionReason           string         `gorm:"type:text;default:''" db:"rejection_reason" json:"rejection_reason,omitempty"`
	CreatedAt                 time.Time      `gorm:"not null;default:now()" db:"created_at" json:"created_at"`
	UpdatedAt                 time.Time      `gorm:"not null;default:now()" db:"updated_at" json:"updated_at"`
	DeletedAt                 gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
	UpdatedBy                 string         `gorm:"type:varchar(255);default:''" db:"updated_by" json:"updated_by,omitempty"`
	DeletedBy                 *string        `gorm:"type:varchar(255)" db:"deleted_by" json:"deleted_by,omitempty"`
	Creator                   *UserBrief     `gorm:"-" json:"creator,omitempty"`
	Updater                   *UserBrief     `gorm:"-" json:"updater,omitempty"`
	Deleter                   *UserBrief     `gorm:"-" json:"deleter,omitempty"`
	CurrencyControlReviewer   *UserBrief     `gorm:"-" json:"currency_control_reviewer,omitempty"`
	ComplianceReviewer        *UserBrief     `gorm:"-" json:"compliance_reviewer,omitempty"`
}
