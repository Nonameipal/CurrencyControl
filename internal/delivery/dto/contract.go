package dto

import (
	"CurrencyControl/internal/domain"
)



type ContractFilter struct {
	Amount *float64 `json:"amount,omitempty"`
}

type ContractDetailsResponse struct {
	domain.Contract
	Invoices             []domain.InvoiceWithDetails  `json:"invoices"`
	AdditionalAgreements []domain.AdditionalAgreement `json:"additional_agreements"`
}
