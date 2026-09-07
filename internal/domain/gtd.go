package domain

import (
    "time"
    "gorm.io/gorm"
)

type GTD struct {
	ID                   int64          `gorm:"primaryKey" db:"id" json:"id"`
	ContractID           int64          `gorm:"not null;index" db:"contract_id" json:"contract_id"`
	InvoiceID            int64          `gorm:"index" db:"invoice_id" json:"invoice_id"`
	GTDNumber            string         `gorm:"not null;column:gtd_number" db:"gtd_number" json:"gtd_number"`
	GTDCurrency          *string        `gorm:"not null;column:gtd_currency" db:"gtd_currency" json:"gtd_currency"`
	GTDDate              *time.Time     `gorm:"type:date;column:gtd_date" db:"gtd_date" json:"gtd_date"`
	GTDAmount            float64        `gorm:"type:decimal(18,2);not null;column:gtd_amount" db:"gtd_amount" json:"gtd_amount"`
	ClosesAmount         float64        `gorm:"type:decimal(18,2);not null;default:0" db:"closes_amount" json:"closes_amount"`
	DocumentPath         *string        `db:"document_path" json:"document_path,omitempty"`
	OriginalDocumentName *string        `db:"original_document_name" json:"original_document_name,omitempty"`
	CreatedBy            string         `gorm:"type:varchar(255);default:''" db:"created_by" json:"created_by"`
	CreatedAt            time.Time      `gorm:"not null;default:now()" db:"created_at" json:"created_at"`
	UpdatedAt            time.Time      `gorm:"not null;default:now()" db:"updated_at" json:"updated_at"`
	DeletedAt            gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

func (GTD) TableName() string {
    return "gtd"
}
