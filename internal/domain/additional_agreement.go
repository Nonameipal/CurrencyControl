package domain

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

const (
	AgreementStatusActive   = "active"
	AgreementStatusArchived = "archived"
)

type AdditionalAgreement struct {
	ID                        int64          `gorm:"primaryKey" db:"id" json:"id"`
	ContractID                int64          `gorm:"not null;index" db:"contract_id" json:"contract_id"`
	AgreementNumber           *string        `db:"agreement_number" json:"agreement_number"`
	AgreementDate             *time.Time     `gorm:"type:date" db:"agreement_date" json:"agreement_date"`
	Subject                   *string        `db:"subject" json:"subject,omitempty"`
	DeliveryDate              *time.Time     `gorm:"type:date;column:delivery_date" db:"delivery_date" json:"delivery_date,omitempty"`
	ReturnDays                *int           `gorm:"column:return_days" db:"return_days" json:"return_days,omitempty"`
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
	ReceiverName              string         `gorm:"type:varchar(255);default:''" db:"receiver_name" json:"receiver_name"`
	ReceiverBank              string         `gorm:"type:varchar(255);default:''" db:"receiver_bank" json:"receiver_bank"`
	ReceiverCountry           string         `gorm:"type:varchar(255);default:''" db:"receiver_country" json:"receiver_country"`
	SenderName                string         `gorm:"type:varchar(255);default:''" db:"sender_name" json:"sender_name"`
	SenderBank                string         `gorm:"type:varchar(255);default:''" db:"sender_bank" json:"sender_bank"`
	SenderCountry             string         `gorm:"type:varchar(255);default:''" db:"sender_country" json:"sender_country"`
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

func (ag *AdditionalAgreement) Normalize() {
	if ag.Amount == nil {
		ag.Amount = ag.ForeignAmount
	} else if ag.ForeignAmount == nil {
		ag.ForeignAmount = ag.Amount
	}

	if ag.Currency == nil {
		ag.Currency = ag.ForeignCurrency
	} else if ag.ForeignCurrency == nil {
		ag.ForeignCurrency = ag.Currency
	}

	if ag.AgreementEndDate == nil {
		ag.AgreementEndDate = ag.ExtendDateTo
	} else if ag.ExtendDateTo == nil {
		ag.ExtendDateTo = ag.AgreementEndDate
	}

	if ag.RemainingAmount == 0 && ag.Amount != nil {
		ag.RemainingAmount = *ag.Amount
	}
}

func (a *AdditionalAgreement) ValidateDates() error {
	if a.AgreementDate == nil || a.AgreementDate.IsZero() || a.DeliveryDate == nil || a.DeliveryDate.IsZero() || a.AgreementEndDate == nil || a.AgreementEndDate.IsZero() {
		return nil
	}
	dContract := time.Date(a.AgreementDate.Year(), a.AgreementDate.Month(), a.AgreementDate.Day(), 0, 0, 0, 0, time.UTC)
	dDelivery := time.Date(a.DeliveryDate.Year(), a.DeliveryDate.Month(), a.DeliveryDate.Day(), 0, 0, 0, 0, time.UTC)
	dEnd := time.Date(a.AgreementEndDate.Year(), a.AgreementEndDate.Month(), a.AgreementEndDate.Day(), 0, 0, 0, 0, time.UTC)

	if dDelivery.Before(dContract) {
		return fmt.Errorf("срок поставки товара не может быть раньше даты доп. соглашения")
	}
	if dEnd.Before(dContract) {
		return fmt.Errorf("дата окончания не может быть раньше даты доп. соглашения")
	}
	if dDelivery.After(dEnd) {
		return fmt.Errorf("срок поставки товара не может быть позже даты окончания")
	}
	if dEnd.Before(dDelivery) {
		return fmt.Errorf("дата окончания не может быть раньше срока поставки товара")
	}
	return nil
}
