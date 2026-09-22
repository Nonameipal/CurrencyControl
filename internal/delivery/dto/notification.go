package dto

import "time"

type NotificationResponse struct {
	Type             string    `json:"type"`
	Title            string    `json:"title"`
	CompanyID        int64     `json:"company_id"`
	CompanyName      string    `json:"company_name"`
	ContractID       int64     `json:"contract_id"`
	ContractNumber   string    `json:"contract_number"`
	InvoiceID        *int64    `json:"invoice_id,omitempty"`
	InvoiceNumber    string    `json:"invoice_number,omitempty"`
	InvoiceAmount    float64   `json:"invoice_amount,omitempty"`
	ClosedAmount     float64   `json:"closed_amount,omitempty"`
	UnclosedAmount   float64   `json:"unclosed_amount,omitempty"`
	Currency         string    `json:"currency,omitempty"`
	DeadlineDate     time.Time `json:"deadline_date"`
	EffectiveEndDate time.Time `json:"effective_end_date"`
	DaysLeft         int       `json:"days_left"`
	Status           string    `json:"status"`
}
