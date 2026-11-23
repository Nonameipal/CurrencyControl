package service

import (
	"context"
	"fmt"

	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/service/ports"
)

type gtdService struct {
	repo ports.GTDRepository
}

func NewGTDService(repo ports.GTDRepository) ports.GTDService {
	return &gtdService{repo: repo}
}

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

func (s *gtdService) Update(ctx context.Context, id int64, g domain.GTD) (domain.GTD, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return domain.GTD{}, err
	}
	if existing == nil {
		return domain.GTD{}, fmt.Errorf("ГТД не найдена")
	}
	shouldReset, err := validateApprovalStatusForUpdate(existing.ApprovalStatus)
	if err != nil {
		return domain.GTD{}, err
	}
	updated, err := s.repo.Update(ctx, id, g)
	if err != nil {
		return domain.GTD{}, err
	}
	if shouldReset {
		_ = s.repo.ResetApprovalStatus(ctx, id)
	}
	return updated, nil
}

func (s *gtdService) SoftDelete(ctx context.Context, id int64) error {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("ГТД не найдена")
	}
	if err := validateApprovalStatusForDelete(existing.ApprovalStatus); err != nil {
		return err
	}
	return s.repo.SoftDelete(ctx, id)
}
