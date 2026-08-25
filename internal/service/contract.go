package service

import (
	"context"

	"CurrencyControl/internal/delivery/dto"
	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/repository"
	"CurrencyControl/internal/service/ports"
)

type contractService struct {
	repo repository.ContractRepository
}

func NewContractService(repo repository.ContractRepository) ports.ContractService {
	return &contractService{repo: repo}
}

func (s *contractService) Create(ctx context.Context, login string, input domain.Contract) (domain.Contract, error) {
	return s.repo.Create(ctx, input)
}

func (s *contractService) GetByID(ctx context.Context, login string, id int64) (domain.Contract, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *contractService) GetAll(ctx context.Context, login string) ([]domain.Contract, error) {
	return s.repo.GetAll(ctx)
}

func (s *contractService) Update(ctx context.Context, login string, id int64, input domain.Contract) (domain.Contract, error) {
	input.ID = id
	return s.repo.Update(ctx, input)
}

func (s *contractService) Delete(ctx context.Context, login string, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *contractService) SearchDashboard(ctx context.Context, login string, req dto.DashboardSearchRequest) ([]dto.DashboardSearchResult, error) {
	return s.repo.SearchDashboard(ctx, req)
}
