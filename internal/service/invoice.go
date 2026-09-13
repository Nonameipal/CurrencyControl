package service

import (
	"context"

	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/service/ports"
)

type invoiceService struct{ repo ports.InvoiceRepository }

func NewInvoiceService(repo ports.InvoiceRepository) ports.InvoiceService {
	return &invoiceService{repo: repo}
}

func (s *invoiceService) GetByContractID(ctx context.Context, contractID int64) ([]domain.InvoiceWithDetails, error) {
	return s.repo.GetByContractID(ctx, contractID)
}
func (s *invoiceService) GetByAdditionalAgreementID(ctx context.Context, agreementID int64) ([]domain.InvoiceWithDetails, error) {
	return s.repo.GetByAdditionalAgreementID(ctx, agreementID)
}
func (s *invoiceService) Create(ctx context.Context, inv domain.Invoice, contractCurrency string) (domain.Invoice, error) {
	return s.repo.Create(ctx, inv, contractCurrency)
}
func (s *invoiceService) GetByID(ctx context.Context, id int64) (domain.Invoice, error) {
	return s.repo.GetByID(ctx, id)
}
func (s *invoiceService) Update(ctx context.Context, id int64, inv domain.Invoice) (domain.Invoice, error) {
	return s.repo.Update(ctx, id, inv)
}
func (s *invoiceService) SoftDelete(ctx context.Context, id int64) error {
	return s.repo.SoftDelete(ctx, id)
}
