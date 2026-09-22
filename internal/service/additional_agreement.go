package service

import (
	"context"
	"fmt"

	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/service/ports"
)

type additionalAgreementService struct {
	repo ports.AdditionalAgreementRepository
}

func NewAdditionalAgreementService(repo ports.AdditionalAgreementRepository) ports.AdditionalAgreementService {
	return &additionalAgreementService{repo: repo}
}

func (s *additionalAgreementService) Create(ctx context.Context, ag domain.AdditionalAgreement) (domain.AdditionalAgreement, error) {
	return s.repo.Create(ctx, ag)
}
func (s *additionalAgreementService) GetByContractID(ctx context.Context, contractID int64) ([]domain.AdditionalAgreement, error) {
	return s.repo.GetByContractID(ctx, contractID)
}
func (s *additionalAgreementService) GetByID(ctx context.Context, id int64) (domain.AdditionalAgreement, error) {
	return s.repo.GetByID(ctx, id)
}
func (s *additionalAgreementService) Update(ctx context.Context, id int64, ag domain.AdditionalAgreement) (domain.AdditionalAgreement, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return domain.AdditionalAgreement{}, err
	}
	if existing.ApprovalStatus != domain.ApprovalStatusApproved {
		return domain.AdditionalAgreement{}, fmt.Errorf("нельзя редактировать доп. соглашение, находящееся на стадии согласования (текущий статус: %s). Редактирование возможно только после подтверждения", existing.ApprovalStatus)
	}
	return s.repo.Update(ctx, id, ag)
}
func (s *additionalAgreementService) SoftDelete(ctx context.Context, id int64) error {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing.ApprovalStatus != domain.ApprovalStatusApproved {
		return fmt.Errorf("нельзя удалить доп. соглашение, находящееся на стадии согласования (текущий статус: %s). Удаление возможно только после подтверждения", existing.ApprovalStatus)
	}
	return s.repo.SoftDelete(ctx, id)
}
func (s *additionalAgreementService) RestoreAdditionalAgreement(ctx context.Context, id int64) error {
	return s.repo.RestoreAdditionalAgreement(ctx, id)
}
