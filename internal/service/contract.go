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

func (s *contractService) GetByClientID(ctx context.Context, login string, clientID int64) ([]domain.Contract, error) {
	return s.repo.GetByClientID(ctx, clientID)
}

func (s *contractService) SearchDashboard(ctx context.Context, login string, req dto.DashboardSearchRequest) ([]dto.DashboardSearchResult, error) {
	return s.repo.SearchDashboard(ctx, req)
}

func (s *contractService) CheckCountry(ctx context.Context, name string) (bool, error) {
	return s.repo.CheckCountry(ctx, name)
}

func (s *contractService) CheckCurrency(ctx context.Context, code string) (bool, error) {
	return s.repo.CheckCurrency(ctx, code)
}

func (s *contractService) GetExpiringContracts(ctx context.Context, branchID int) ([]dto.NotificationResponse, error) {
	return s.repo.GetExpiringContracts(ctx, branchID)
}

func (s *contractService) Update(ctx context.Context, id int64, c domain.Contract) (domain.Contract, error) {
	return s.repo.Update(ctx, id, c)
}