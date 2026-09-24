package service

import (
	"context"
	"fmt"

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
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return domain.Invoice{}, err
	}
	switch existing.ApprovalStatus {
	case domain.ApprovalStatusPendingCurrencyControl:
		return domain.Invoice{}, fmt.Errorf("документ на рассмотрении валютного контроля — редактирование запрещено")
	case domain.ApprovalStatusPendingCompliance:
		return domain.Invoice{}, fmt.Errorf("документ на рассмотрении комплаенс-контроля — редактирование запрещено")
	case domain.ApprovalStatusRevisionRequired:
		if inv.UpdatedBy != existing.CreatedBy {
			return domain.Invoice{}, fmt.Errorf("редактировать документ на доработке может только его создатель")
		}
		updated, err := s.repo.Update(ctx, id, inv)
		if err != nil {
			return domain.Invoice{}, err
		}
		_ = s.repo.ResetApprovalStatus(ctx, id)
		return updated, nil
	case domain.ApprovalStatusApproved:
	case domain.ApprovalStatusRejectedCurrencyControl, domain.ApprovalStatusRejectedCompliance:
		return domain.Invoice{}, fmt.Errorf("документ отклонён и перемещён в корзину — редактирование невозможно")
	}
	return s.repo.Update(ctx, id, inv)
}
func (s *invoiceService) SoftDelete(ctx context.Context, id int64) error {
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
