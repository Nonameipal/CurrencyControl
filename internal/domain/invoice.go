package domain

import (
	"gorm.io/gorm"
	"time"
)

type Invoice struct {
	ID                        int64          `gorm:"primaryKey" db:"id" json:"id"`
	ContractID                int64          `gorm:"not null;index" db:"contract_id" json:"contract_id"`
	AdditionalAgreementID     *int64         `gorm:"index" db:"additional_agreement_id" json:"additional_agreement_id,omitempty"`
	InvoiceNumber             string         `gorm:"not null" db:"invoice_number" json:"invoice_number"`
	InvoiceDate               time.Time      `gorm:"type:date;not null" db:"invoice_date" json:"invoice_date"`
	Amount                    float64        `gorm:"type:decimal(18,2);not null" db:"amount" json:"amount"`
	Currency                  string         `gorm:"not null" db:"currency" json:"currency"`
	HSCode                    string         `gorm:"column:hs_code;type:varchar(50);default:''" db:"hs_code" json:"hs_code"`
	DeductAmount              float64        `gorm:"type:decimal(18,2)" db:"deduct_amount" json:"deduct_amount"`
	SenderName                string         `gorm:"type:varchar(255);default:''" db:"sender_name" json:"sender_name"`
	SenderBank                string         `gorm:"type:varchar(255);default:''" db:"sender_bank" json:"sender_bank"`
	SenderCountry             string         `gorm:"type:varchar(255);default:''" db:"sender_country" json:"sender_country"`
	DocumentPath              *string        `db:"document_path" json:"document_path,omitempty"`
	CreatedBy                 string         `gorm:"type:varchar(255);default:''" db:"created_by" json:"created_by"`
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

type InvoiceWithDetails struct {
	Invoice
	GTD                    *GTD           `json:"gtd,omitempty"`
	PaymentOrders          []PaymentOrder `json:"payment_orders,omitempty"`
	PaidAmount             float64        `json:"paid_amount"`
	RemainingPaymentAmount float64        `json:"remaining_payment_amount"`
}
