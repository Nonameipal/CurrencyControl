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
	switch existing.ApprovalStatus {
	case domain.ApprovalStatusPendingCurrencyControl:
		return domain.AdditionalAgreement{}, fmt.Errorf("документ на рассмотрении валютного контроля редактирование запрещено")
	case domain.ApprovalStatusPendingCompliance:
		return domain.AdditionalAgreement{}, fmt.Errorf("документ на рассмотрении комплаенс-контроля редактирование запрещено")
	case domain.ApprovalStatusRevisionRequired:
		if ag.UpdatedBy != existing.CreatedBy {
			return domain.AdditionalAgreement{}, fmt.Errorf("редактировать документ на доработке может только его создатель")
		}
		updated, err := s.repo.Update(ctx, id, ag)
		if err != nil {
			return domain.AdditionalAgreement{}, err
		}
		// Сбрасываем статус обратно в pending_currency_control после успешного редактирования
		_ = s.repo.ResetApprovalStatus(ctx, id)
		return updated, nil
	case domain.ApprovalStatusApproved:
		// разрешено
	case domain.ApprovalStatusRejectedCurrencyControl, domain.ApprovalStatusRejectedCompliance:
		return domain.AdditionalAgreement{}, fmt.Errorf("документ отклонён и перемещён в корзину — редактирование невозможно")
	}
	return s.repo.Update(ctx, id, ag)
}
func (s *additionalAgreementService) SoftDelete(ctx context.Context, id int64) error {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	switch existing.ApprovalStatus {
	case domain.ApprovalStatusPendingCurrencyControl:
		return fmt.Errorf("документ на рассмотрении валютного контроля — удаление запрещено")
	case domain.ApprovalStatusPendingCompliance:
		return fmt.Errorf("документ на рассмотрении комплаенс-контроля — удаление запрещено")
	case domain.ApprovalStatusRevisionRequired:
		return fmt.Errorf("нельзя удалить документ, отправленный на доработку")
	case domain.ApprovalStatusRejectedCurrencyControl, domain.ApprovalStatusRejectedCompliance:
		return fmt.Errorf("документ уже в корзине")
	}
	return s.repo.SoftDelete(ctx, id)
}
func (s *additionalAgreementService) RestoreAdditionalAgreement(ctx context.Context, id int64) error {
	return s.repo.RestoreAdditionalAgreement(ctx, id)
}
