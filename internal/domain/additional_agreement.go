package domain

import (
    "time"
    "gorm.io/gorm"
)

type AdditionalAgreement struct {
	ID                       int64          `gorm:"primaryKey" db:"id" json:"id"`
	ContractID               int64          `gorm:"not null;index" db:"contract_id" json:"contract_id"`
	AgreementNumber          *string        `db:"agreement_number" json:"agreement_number"`
	AgreementDate            *time.Time     `gorm:"type:date" db:"agreement_date" json:"agreement_date"`
	DeliveryConditions       *string        `gorm:"column:new_delivery_conditions" db:"new_delivery_conditions" json:"delivery_conditions,omitempty"`
	DeliveryTermDays         *int           `gorm:"column:new_delivery_term_days" db:"new_delivery_term_days" json:"delivery_term_days,omitempty"`
	ReturnTermDays           *int           `gorm:"column:new_return_term_days" db:"new_return_term_days" json:"return_term_days,omitempty"`
	Subject                  *string        `db:"subject" json:"subject,omitempty"`
	ExtendDateTo             *time.Time     `gorm:"type:date;column:extend_date_to" db:"extend_date_to" json:"extend_date_to,omitempty"`
	ForeignAmount            *float64       `gorm:"type:decimal(18,2)" db:"foreign_amount" json:"foreign_amount,omitempty"`
	ForeignCurrency          *string        `gorm:"column:currency" db:"currency" json:"foreign_currency,omitempty"`
	AmountInContractCurrency float64        `gorm:"type:decimal(18,2)" db:"amount_in_contract_currency" json:"amount_in_contract_currency"`
	DocumentPath             string         `db:"document_path" json:"document_path"`
	OriginalDocumentName     string         `db:"original_document_name" json:"original_document_name"`
	CreatedBy                string         `gorm:"type:varchar(255);default:''" db:"created_by" json:"created_by"`
	CreatedAt                time.Time      `gorm:"not null;default:now()" db:"created_at" json:"created_at"`
	UpdatedAt                time.Time      `gorm:"not null;default:now()" db:"updated_at" json:"updated_at"`
	DeletedAt                gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}
