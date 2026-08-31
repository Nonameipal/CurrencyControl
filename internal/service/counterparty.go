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

func (s *counterpartyService) Update(ctx context.Context, id int64, input domain.Counterparty) (domain.Counterparty, error) {
	return s.repo.Update(ctx, id, input)
}
