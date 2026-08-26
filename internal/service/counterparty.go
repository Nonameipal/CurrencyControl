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
