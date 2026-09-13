package service

import (
	"context"

	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/service/ports"
)

func NewGTDService(repo ports.GTDRepository) ports.GTDService {
	return &gtdService{repo: repo}
}

type gtdService struct{ repo ports.GTDRepository }

func (s *gtdService) Create(ctx context.Context, g domain.GTD) (domain.GTD, error) {
	return s.repo.Create(ctx, g)
}
func (s *gtdService) GetByID(ctx context.Context, id int64) (*domain.GTD, error) {
	return s.repo.GetByID(ctx, id)
}
func (s *gtdService) GetByInvoiceID(ctx context.Context, invoiceID int64) (*domain.GTD, error) {
	return s.repo.GetByInvoiceID(ctx, invoiceID)
}
func (s *gtdService) GetListByInvoiceID(ctx context.Context, invoiceID int64) ([]domain.GTD, error) {
	return s.repo.GetListByInvoiceID(ctx, invoiceID)
}
func (s *gtdService) GetByContractID(ctx context.Context, contractID int64) ([]domain.GTD, error) {
	return s.repo.GetByContractID(ctx, contractID)
}
func (s *gtdService) GetByAdditionalAgreementID(ctx context.Context, agreementID int64) ([]domain.GTD, error) {
	return s.repo.GetByAdditionalAgreementID(ctx, agreementID)
}
func (s *gtdService) Update(ctx context.Context, id int64, g domain.GTD) (domain.GTD, error) {
	return s.repo.Update(ctx, id, g)
}
func (s *gtdService) SoftDelete(ctx context.Context, id int64) error {
	return s.repo.SoftDelete(ctx, id)
}

