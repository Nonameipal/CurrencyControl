package domain

import "time"

type AdditionalAgreement struct {
	ID                       int64      `db:"id" json:"id"`
	ContractID               int64      `db:"contract_id" json:"contract_id"`
	DeliveryConditions       *string    `db:"delivery_conditions" json:"delivery_conditions,omitempty"`
	DeliveryTermDays         *int       `db:"delivery_term_days" json:"delivery_term_days,omitempty"`
	ReturnTermDays           *int       `db:"return_term_days" json:"return_term_days,omitempty"`
	Subject                  *string    `db:"subject" json:"subject,omitempty"`
	ExtendDateTo             *time.Time `db:"extend_date_to" json:"extend_date_to,omitempty"`
	ForeignAmount            *float64   `db:"foreign_amount" json:"foreign_amount,omitempty"`
	ForeignCurrency          *string    `db:"foreign_currency" json:"foreign_currency,omitempty"`
	AmountInContractCurrency float64    `db:"amount_in_contract_currency" json:"amount_in_contract_currency"`
	DocumentPath             string     `db:"document_path" json:"document_path"`
	OriginalDocumentName     string     `db:"original_document_name" json:"original_document_name"`
	CreatedAt                time.Time  `db:"created_at" json:"created_at"`
}