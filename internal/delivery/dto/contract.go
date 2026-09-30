package dto

import (
	"CurrencyControl/internal/domain"
)



type ContractDetailsResponse struct {
	domain.Contract
	Invoices             []domain.InvoiceWithDetails  `json:"invoices"`
	AdditionalAgreements []domain.AdditionalAgreement `json:"additional_agreements"`
}
