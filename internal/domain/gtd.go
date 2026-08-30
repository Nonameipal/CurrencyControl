package domain

import "time"

type GTD struct {
	ID                   int64      `db:"id" json:"id"`
	InvoiceID            int64      `db:"invoice_id" json:"invoice_id"`
	GTDNumber            string     `db:"gtd_number" json:"gtd_number"`
	GTDAmount            float64    `db:"gtd_amount" json:"gtd_amount"`
	GTDCurrency          *string    `db:"gtd_currency" json:"gtd_currency,omitempty"`
	GTDDate              *time.Time `db:"gtd_date" json:"gtd_date"`
	ClosesAmount         float64    `db:"closes_amount" json:"closes_amount"`
	DocumentPath         *string    `db:"document_path" json:"document_path,omitempty"`
	OriginalDocumentName *string    `db:"original_document_name" json:"original_document_name,omitempty"`
	CreatedAt            time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt            time.Time  `db:"updated_at" json:"updated_at"`
}