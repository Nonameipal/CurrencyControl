package domain

import "time"

type Invoice struct {
	ID                   int64     `db:"id" json:"id"`
	ContractID           int64     `db:"contract_id" json:"contract_id"`
	InvoiceNumber        string    `db:"invoice_number" json:"invoice_number"`
	InvoiceName          string    `db:"invoice_name" json:"invoice_name"`
	InvoiceDate          time.Time `db:"invoice_date" json:"invoice_date"`
	Amount               float64   `db:"amount" json:"amount"`
	Currency             string    `db:"currency" json:"currency"`
	DeductAmount         float64   `db:"deduct_amount" json:"deduct_amount"`
	DocumentPath         *string   `db:"document_path" json:"document_path,omitempty"`
	OriginalDocumentName *string   `db:"original_document_name" json:"original_document_name,omitempty"`
	CreatedBy            string    `db:"created_by" json:"created_by"`
	CreatedAt            time.Time `db:"created_at" json:"created_at"`
	UpdatedAt            time.Time `db:"updated_at" json:"updated_at"`
}

type InvoiceWithDetails struct {
	Invoice
	InvoiceRemaining float64   `json:"invoice_remaining"`
	GTD              *GTD      `json:"gtd,omitempty"`
}