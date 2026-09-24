package service

import (
	"context"
	"fmt"

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
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return domain.GTD{}, err
	}
	if existing == nil {
		return domain.GTD{}, fmt.Errorf("ГТД не найдена")
	}
	switch existing.ApprovalStatus {
	case domain.ApprovalStatusPendingCurrencyControl:
		return domain.GTD{}, fmt.Errorf("документ на рассмотрении валютного контроля — редактирование запрещено")
	case domain.ApprovalStatusPendingCompliance:
		return domain.GTD{}, fmt.Errorf("документ на рассмотрении комплаенс-контроля — редактирование запрещено")
	case domain.ApprovalStatusRevisionRequired:
		if g.UpdatedBy != existing.CreatedBy {
			return domain.GTD{}, fmt.Errorf("редактировать документ на доработке может только его создатель")
		}
		updated, err := s.repo.Update(ctx, id, g)
		if err != nil {
			return domain.GTD{}, err
		}
		// Сбрасываем статус обратно в pending_currency_control после успешного редактирования
		_ = s.repo.ResetApprovalStatus(ctx, id)
		return updated, nil
	case domain.ApprovalStatusApproved:
		// разрешено
	case domain.ApprovalStatusRejectedCurrencyControl, domain.ApprovalStatusRejectedCompliance:
		return domain.GTD{}, fmt.Errorf("документ отклонён и перемещён в корзину редактирование невозможно")
	}
	return s.repo.Update(ctx, id, g)
}
func (s *gtdService) SoftDelete(ctx context.Context, id int64) error {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("ГТД не найдена")
	}
	switch existing.ApprovalStatus {
	case domain.ApprovalStatusPendingCurrencyControl:
		return fmt.Errorf("документ на рассмотрении валютного контроля удаление запрещено")
	case domain.ApprovalStatusPendingCompliance:
		return fmt.Errorf("документ на рассмотрении комплаенс-контроля удаление запрещено")
	case domain.ApprovalStatusRevisionRequired:
		return fmt.Errorf("нельзя удалить документ, отправленный на доработку")
	case domain.ApprovalStatusRejectedCurrencyControl, domain.ApprovalStatusRejectedCompliance:
		return fmt.Errorf("документ уже в корзине")
	}
	return s.repo.SoftDelete(ctx, id)
}
