package domain

import (
    "time"
    "gorm.io/gorm"
)

type Invoice struct {
	ID                   int64          `gorm:"primaryKey" db:"id" json:"id"`
	ContractID           int64          `gorm:"not null;index" db:"contract_id" json:"contract_id"`
	InvoiceNumber        string         `gorm:"not null" db:"invoice_number" json:"invoice_number"`
	InvoiceName          string         `gorm:"not null" db:"invoice_name" json:"invoice_name"`
	InvoiceDate          time.Time      `gorm:"type:date;not null" db:"invoice_date" json:"invoice_date"`
	Amount               float64        `gorm:"type:decimal(18,2);not null" db:"amount" json:"amount"`
	Currency             string         `gorm:"not null" db:"currency" json:"currency"`
	DeductAmount         float64        `gorm:"type:decimal(18,2)" db:"deduct_amount" json:"deduct_amount"`
	DocumentPath         *string        `db:"document_path" json:"document_path,omitempty"`
	OriginalDocumentName *string        `db:"original_document_name" json:"original_document_name,omitempty"`
	CreatedBy            string         `gorm:"type:varchar(255);default:''" db:"created_by" json:"created_by"`
	CreatedAt            time.Time      `gorm:"not null;default:now()" db:"created_at" json:"created_at"`
	UpdatedAt            time.Time      `gorm:"not null;default:now()" db:"updated_at" json:"updated_at"`
	DeletedAt            gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

type InvoiceWithDetails struct {
	Invoice
	GTD *GTD `json:"gtd,omitempty"`
}
