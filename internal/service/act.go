package service

import (
	"context"

	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/service/ports"
)

type actService struct {
	repo ports.ActRepository
}

func NewActService(repo ports.ActRepository) ports.ActService {
	return &actService{repo: repo}
}

func (s *actService) Create(ctx context.Context, act domain.Act) (domain.Act, error) {
	return s.repo.Create(ctx, act)
}

func (s *actService) GetByID(ctx context.Context, id int64) (*domain.Act, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *actService) GetByInvoiceID(ctx context.Context, invoiceID int64) ([]domain.Act, error) {
	return s.repo.GetByInvoiceID(ctx, invoiceID)
}

func (s *actService) GetByContractID(ctx context.Context, contractID int64) ([]domain.Act, error) {
	return s.repo.GetByContractID(ctx, contractID)
}

func (s *actService) Update(ctx context.Context, id int64, act domain.Act) (domain.Act, error) {
	return s.repo.Update(ctx, id, act)
}

func (s *actService) SoftDelete(ctx context.Context, id int64) error {
	return s.repo.SoftDelete(ctx, id)
}
