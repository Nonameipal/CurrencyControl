package domain

import (
	"time"

	"gorm.io/gorm"
)

type PaymentOrder struct {
	ID                    int64          `gorm:"primaryKey" db:"id" json:"id"`
	ContractID            int64          `gorm:"not null;index" db:"contract_id" json:"contract_id"`
	AdditionalAgreementID *int64         `gorm:"index" db:"additional_agreement_id" json:"additional_agreement_id,omitempty"`
	InvoiceID             int64          `gorm:"not null;index" db:"invoice_id" json:"invoice_id"`
	OperationDate         time.Time      `gorm:"type:date;not null" db:"operation_date" json:"operation_date"`
	PaymentOrderNumber    string         `gorm:"type:varchar(255);not null" db:"payment_order_number" json:"payment_order_number"`
	Amount                float64        `gorm:"type:decimal(18,2);not null" db:"amount" json:"amount"`
	Currency              string         `gorm:"type:varchar(10);not null" db:"currency" json:"currency"`
	Payer                 string         `gorm:"type:varchar(255);not null" db:"payer" json:"payer"`
	ReceiverName          string         `gorm:"type:varchar(255);not null" db:"receiver_name" json:"receiver_name"`
	ReceiverBank          string         `gorm:"type:varchar(255);not null" db:"receiver_bank" json:"receiver_bank"`
	PaymentPurpose        string         `gorm:"type:text;not null" db:"payment_purpose" json:"payment_purpose"`
	ReceiverCountry       string         `gorm:"type:varchar(255);not null" db:"receiver_country" json:"receiver_country"`
	ContractNumber        string         `gorm:"type:varchar(255);not null" db:"contract_number" json:"contract_number"`
	InvoiceNumber         string         `gorm:"type:varchar(255);not null" db:"invoice_number" json:"invoice_number"`
	ValueDate             time.Time      `gorm:"type:date;not null" db:"value_date" json:"value_date"`
	CreatedBy             string         `gorm:"type:varchar(255);not null" db:"created_by" json:"created_by"`
	DocumentPath          *string        `db:"document_path" json:"document_path,omitempty"`
	CreatedAt             time.Time      `gorm:"not null;default:now()" db:"created_at" json:"created_at"`
	UpdatedAt             time.Time      `gorm:"not null;default:now()" db:"updated_at" json:"updated_at"`
	DeletedAt             gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}
