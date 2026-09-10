package dto

import (
	"time"

	"CurrencyControl/internal/domain"
)

type CreateContractRequest struct {
	ContractNumber      string  `json:"contract_number"`
	ContractDate        string  `json:"contract_date"`
	AdditionalAgreement string  `json:"additional_agreement"`
	Subject             string  `json:"subject"`
	TotalAmount         float64 `json:"total_amount"`
	ContractCurrency    string  `json:"contract_currency"`
	ContractEndDate     string  `json:"contract_end_date"`
}

type UpdateContractRequest struct {
	ContractNumber      string  `json:"contract_number"`
	ContractDate        string  `json:"contract_date"`
	AdditionalAgreement string  `json:"additional_agreement"`
	Subject             string  `json:"subject"`
	ContractCurrency    string  `json:"contract_currency"`
	ContractEndDate     string  `json:"contract_end_date"`
}

type ContractResponse struct {
	ID                  int64      `json:"id"`
	ContractNumber      string     `json:"contract_number"`
	ContractDate        time.Time  `json:"contract_date"`
	AdditionalAgreement string     `json:"additional_agreement"`
	Subject             string     `json:"subject"`
	TotalAmount         float64    `json:"total_amount"`
	RemainingAmount     float64    `json:"remaining_amount"`
	ContractCurrency    string     `json:"contract_currency"`
	ContractEndDate     *time.Time `json:"contract_end_date"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

type ContractDetailsResponse struct {
	domain.Contract
	Invoices             []domain.InvoiceWithDetails    `json:"invoices"`
	AdditionalAgreements []domain.AdditionalAgreement `json:"additional_agreements"`
}
