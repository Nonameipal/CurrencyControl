package service

import (
	"context"

	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/repository"
)

type InvoiceService interface {
	GetByContractID(ctx context.Context, contractID int64) ([]domain.InvoiceWithDetails, error)
	Create(ctx context.Context, inv domain.Invoice) (domain.Invoice, error)
	GetByID(ctx context.Context, id int64) (domain.Invoice, error)
}

type GTDService interface {
	Create(ctx context.Context, g domain.GTD) (domain.GTD, error)
}

type invoiceService struct {
	repo repository.InvoiceRepository
}

type gtdService struct {
	repo repository.GTDRepository
}

func NewInvoiceService(repo repository.InvoiceRepository) InvoiceService {
	return &invoiceService{repo: repo}
}

func NewGTDService(repo repository.GTDRepository) GTDService {
	return &gtdService{repo: repo}
}

func (s *invoiceService) GetByContractID(ctx context.Context, contractID int64) ([]domain.InvoiceWithDetails, error) {
	return s.repo.GetByContractID(ctx, contractID)
}

func (s *invoiceService) Create(ctx context.Context, inv domain.Invoice) (domain.Invoice, error) {
	return s.repo.Create(ctx, inv)
}

func (s *invoiceService) GetByID(ctx context.Context, id int64) (domain.Invoice, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *gtdService) Create(ctx context.Context, g domain.GTD) (domain.GTD, error) {
	return s.repo.Create(ctx, g)
}