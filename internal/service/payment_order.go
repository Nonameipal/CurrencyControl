package service

import (
	"context"
	"fmt"

	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/service/ports"
)

type paymentOrderService struct {
	repo ports.PaymentOrderRepository
}

func NewPaymentOrderService(repo ports.PaymentOrderRepository) ports.PaymentOrderService {
	return &paymentOrderService{repo: repo}
}

func (s *paymentOrderService) Create(ctx context.Context, po domain.PaymentOrder) (domain.PaymentOrder, error) {
	return s.repo.Create(ctx, po)
}

func (s *paymentOrderService) GetByID(ctx context.Context, id int64) (*domain.PaymentOrder, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *paymentOrderService) GetByInvoiceID(ctx context.Context, invoiceID int64) ([]domain.PaymentOrder, error) {
	return s.repo.GetByInvoiceID(ctx, invoiceID)
}

func (s *paymentOrderService) GetByContractID(ctx context.Context, contractID int64) ([]domain.PaymentOrder, error) {
	return s.repo.GetByContractID(ctx, contractID)
}

func (s *paymentOrderService) Update(ctx context.Context, id int64, po domain.PaymentOrder) (*domain.PaymentOrder, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, fmt.Errorf("платежное поручение не найдено")
	}

	switch existing.ApprovalStatus {
	case domain.ApprovalStatusPendingCurrencyControl:
		return nil, fmt.Errorf("документ на рассмотрении валютного контроля — редактирование запрещено")
	case domain.ApprovalStatusPendingCompliance:
		return nil, fmt.Errorf("документ на рассмотрении комплаенс-контроля — редактирование запрещено")
	case domain.ApprovalStatusRevisionRequired:
		if po.UpdatedBy != existing.CreatedBy {
			return nil, fmt.Errorf("редактировать документ на доработке может только его создатель")
		}
		updated, err := s.repo.Update(ctx, id, po)
		if err != nil {
			return nil, err
		}
		_ = s.repo.ResetApprovalStatus(ctx, id)
		return updated, nil
	case domain.ApprovalStatusApproved:
	case domain.ApprovalStatusRejectedCurrencyControl, domain.ApprovalStatusRejectedCompliance:
		return nil, fmt.Errorf("документ отклонён и перемещён в корзину — редактирование невозможно")
	}

	return s.repo.Update(ctx, id, po)
}

func (s *paymentOrderService) SoftDelete(ctx context.Context, id int64) error {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("платежное поручение не найдено")
	}

	switch existing.ApprovalStatus {
	case domain.ApprovalStatusPendingCurrencyControl:
		return fmt.Errorf("документ на рассмотрении валютного контроля — удаление запрещено")
	case domain.ApprovalStatusPendingCompliance:
		return fmt.Errorf("документ на рассмотрении комплаенс-контроля — удаление запрещено")
	case domain.ApprovalStatusRevisionRequired:
		return fmt.Errorf("нельзя удалить документ, отправленный на доработку")
	case domain.ApprovalStatusRejectedCurrencyControl, domain.ApprovalStatusRejectedCompliance:
		return fmt.Errorf("документ уже отклонён и находится в корзине")
	}

	return s.repo.SoftDelete(ctx, id)
}
