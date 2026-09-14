package service

import (
	"context"

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
	return s.repo.Update(ctx, id, po)
}

func (s *paymentOrderService) SoftDelete(ctx context.Context, id int64) error {
	return s.repo.SoftDelete(ctx, id)
}
