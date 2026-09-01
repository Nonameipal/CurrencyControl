package service

import (
	"context"

	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/service/ports"
)

type counterpartyService struct {
	repo ports.CounterpartyRepository
}

func NewCounterpartyService(repo ports.CounterpartyRepository) ports.CounterpartyService {
	return &counterpartyService{repo: repo}
}

func (s *counterpartyService) Create(ctx context.Context, login string, input domain.Counterparty) (domain.Counterparty, error) {
	return s.repo.Create(ctx, input)
}

func (s *counterpartyService) CheckExistsInBranch(ctx context.Context, branchID int, name string) (bool, error) {
	return s.repo.CheckExistsInBranch(ctx, branchID, name)
}

func (s *counterpartyService) GetByID(ctx context.Context, id int64) (domain.Counterparty, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *counterpartyService) Update(ctx context.Context, id int64, input domain.Counterparty) (domain.Counterparty, error) {
	return s.repo.Update(ctx, id, input)
}

func (s *counterpartyService) SoftDelete(ctx context.Context, id int64) error {
	return s.repo.SoftDelete(ctx, id)
}
