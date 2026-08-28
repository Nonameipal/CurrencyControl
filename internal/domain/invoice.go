package domain

import "time"

type Invoice struct {
	ID            int64     `db:"id" json:"id"`
	ContractID    int64     `db:"contract_id" json:"contract_id"`
	InvoiceNumber string    `db:"invoice_number" json:"invoice_number"`
	InvoiceName   string    `db:"invoice_name" json:"invoice_name"`
	InvoiceDate   time.Time `db:"invoice_date" json:"invoice_date"`
	Amount        float64   `db:"amount" json:"amount"`
	Currency      string    `db:"currency" json:"currency"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
}

type InvoiceWithDetails struct {
	Invoice
	Document *Document `json:"document,omitempty"`
	GTD      *GTD      `json:"gtd,omitempty"`
}