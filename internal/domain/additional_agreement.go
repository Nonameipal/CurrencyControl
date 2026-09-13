package domain

import (
	"gorm.io/gorm"
	"time"
)

const (
	DocTypeAdditionalAgreement = "additional_agreement"
	DocTypeSpecification       = "specification"
	DocTypeAppendix            = "appendix"

	AgreementStatusActive   = "active"
	AgreementStatusArchived = "archived"
)

type AdditionalAgreement struct {
	ID                        int64          `gorm:"primaryKey" db:"id" json:"id"`
	ContractID                int64          `gorm:"not null;index" db:"contract_id" json:"contract_id"`
	DocType                   string         `gorm:"column:doc_type;type:varchar(50);default:'additional_agreement'" db:"doc_type" json:"doc_type"`
	AgreementNumber           *string        `db:"agreement_number" json:"agreement_number"`
	AgreementDate             *time.Time     `gorm:"type:date" db:"agreement_date" json:"agreement_date"`
	Subject                   *string        `db:"subject" json:"subject,omitempty"`
	DeliveryDate              *time.Time     `gorm:"type:date;column:delivery_date" db:"delivery_date" json:"delivery_date,omitempty"`
	ReturnDate                *time.Time     `gorm:"type:date;column:return_date" db:"return_date" json:"return_date,omitempty"`
	ExtendDateTo              *time.Time     `gorm:"type:date;column:extend_date_to" db:"extend_date_to" json:"extend_date_to,omitempty"`
	AgreementEndDate          *time.Time     `gorm:"-" json:"agreement_end_date,omitempty"`
	Amount                    *float64       `gorm:"-" json:"amount,omitempty"`
	Currency                  *string        `gorm:"-" json:"currency,omitempty"`
	ForeignAmount             *float64       `gorm:"type:decimal(18,2)" db:"foreign_amount" json:"foreign_amount,omitempty"`
	ForeignCurrency           *string        `gorm:"column:currency" db:"currency" json:"foreign_currency,omitempty"`
	RemainingAmount           float64        `gorm:"type:decimal(18,2);not null;default:0" db:"remaining_amount" json:"remaining_amount"`
	Status                    string         `gorm:"type:varchar(20);not null;default:'active';index" db:"status" json:"status"`
	ArchivedAt                *time.Time     `gorm:"type:timestamptz" db:"archived_at" json:"archived_at,omitempty"`
	DocumentPath              string         `db:"document_path" json:"document_path"`
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
}

func (ag *AdditionalAgreement) Normalize() {
	if ag.Amount == nil && ag.ForeignAmount != nil {
		ag.Amount = ag.ForeignAmount
	} else if ag.ForeignAmount == nil && ag.Amount != nil {
		ag.ForeignAmount = ag.Amount
	}

	if ag.Currency == nil && ag.ForeignCurrency != nil {
		ag.Currency = ag.ForeignCurrency
	} else if ag.ForeignCurrency == nil && ag.Currency != nil {
		ag.ForeignCurrency = ag.Currency
	}

	if ag.AgreementEndDate == nil && ag.ExtendDateTo != nil {
		ag.AgreementEndDate = ag.ExtendDateTo
	} else if ag.ExtendDateTo == nil && ag.AgreementEndDate != nil {
		ag.ExtendDateTo = ag.AgreementEndDate
	}

	if ag.RemainingAmount == 0 && ag.Amount != nil {
		ag.RemainingAmount = *ag.Amount
	}
}
