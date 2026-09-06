package service

import (
	"context"

	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/repository"
)

func NewGTDService(repo repository.GTDRepository) GTDService {
	return &gtdService{repo: repo}
}

type gtdService struct{ repo repository.GTDRepository }

type GTDService interface {
	Create(ctx context.Context, g domain.GTD) (domain.GTD, error)
	GetByInvoiceID(ctx context.Context, invoiceID int64) (*domain.GTD, error)
	Update(ctx context.Context, id int64, g domain.GTD) (domain.GTD, error)
	SoftDelete(ctx context.Context, id int64) error
}


func (s *gtdService) Create(ctx context.Context, g domain.GTD) (domain.GTD, error) {
	return s.repo.Create(ctx, g)
}
func (s *gtdService) GetByInvoiceID(ctx context.Context, invoiceID int64) (*domain.GTD, error) {
	return s.repo.GetByInvoiceID(ctx, invoiceID)
}
func (s *gtdService) Update(ctx context.Context, id int64, g domain.GTD) (domain.GTD, error) {
	return s.repo.Update(ctx, id, g)
}
func (s *gtdService) SoftDelete(ctx context.Context, id int64) error {
	return s.repo.SoftDelete(ctx, id)
}

