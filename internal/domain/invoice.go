package domain

import (
    "time"
    "gorm.io/gorm"
)

type Invoice struct {
	ID                    int64          `gorm:"primaryKey" db:"id" json:"id"`
	ContractID            int64          `gorm:"not null;index" db:"contract_id" json:"contract_id"`
	AdditionalAgreementID *int64         `gorm:"index" db:"additional_agreement_id" json:"additional_agreement_id,omitempty"`
	InvoiceNumber         string         `gorm:"not null" db:"invoice_number" json:"invoice_number"`
	InvoiceDate  time.Time      `gorm:"type:date;not null" db:"invoice_date" json:"invoice_date"`
	Amount       float64        `gorm:"type:decimal(18,2);not null" db:"amount" json:"amount"`
	Currency     string         `gorm:"not null" db:"currency" json:"currency"`
	HSCode       string         `gorm:"column:hs_code;type:varchar(50);default:''" db:"hs_code" json:"hs_code"` // Код ТН ВЭД (HS CODE)
	DeductAmount float64        `gorm:"type:decimal(18,2)" db:"deduct_amount" json:"deduct_amount"`
	DocumentPath *string        `db:"document_path" json:"document_path,omitempty"`
	CreatedBy    string         `gorm:"type:varchar(255);default:''" db:"created_by" json:"created_by"`
	CreatedAt    time.Time      `gorm:"not null;default:now()" db:"created_at" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"not null;default:now()" db:"updated_at" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

type InvoiceWithDetails struct {
	Invoice
	GTD *GTD `json:"gtd,omitempty"`
}
