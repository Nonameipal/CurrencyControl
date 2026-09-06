package service

import (
	"context"

	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/repository"
)

type InvoiceService interface {
	GetByContractID(ctx context.Context, contractID int64) ([]domain.InvoiceWithDetails, error)
	Create(ctx context.Context, inv domain.Invoice, contractCurrency string) (domain.Invoice, error)
	GetByID(ctx context.Context, id int64) (domain.Invoice, error)
	Update(ctx context.Context, id int64, inv domain.Invoice) (domain.Invoice, error)
	SoftDelete(ctx context.Context, id int64) error
}

type invoiceService struct{ repo repository.InvoiceRepository }

func NewInvoiceService(repo repository.InvoiceRepository) InvoiceService {
	return &invoiceService{repo: repo}
}

func (s *invoiceService) GetByContractID(ctx context.Context, contractID int64) ([]domain.InvoiceWithDetails, error) {
	return s.repo.GetByContractID(ctx, contractID)
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

