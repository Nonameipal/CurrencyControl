package service

import (
	"context"
	"fmt"

	"CurrencyControl/internal/delivery/dto"
	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/service/ports"
)

type contractService struct {
	repo ports.ContractRepository
}

func NewContractService(repo ports.ContractRepository) ports.ContractService {
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
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return domain.Contract{}, err
	}
	if existing.ApprovalStatus != domain.ApprovalStatusApproved {
		return domain.Contract{}, fmt.Errorf("нельзя редактировать контракт, находящийся на стадии согласования (текущий статус: %s). Редактирование возможно только после подтверждения", existing.ApprovalStatus)
	}
	return s.repo.Update(ctx, id, c)
}

func (s *contractService) SoftDelete(ctx context.Context, id int64) error {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing.ApprovalStatus != domain.ApprovalStatusApproved {
		return fmt.Errorf("нельзя удалить контракт, находящийся на стадии согласования (текущий статус: %s). Удаление возможно только после подтверждения", existing.ApprovalStatus)
	}
	return s.repo.SoftDelete(ctx, id)
}

func (s *contractService) GetArchived(ctx context.Context, branchID int, page, pageSize int) ([]domain.Contract, int, error) {
	return s.repo.GetArchived(ctx, branchID, page, pageSize)
}

func (s *contractService) GetArchivedByClientID(ctx context.Context, clientID int64) ([]domain.Contract, error) {
	return s.repo.GetArchivedByClientID(ctx, clientID)
}

func (s *contractService) RestoreContract(ctx context.Context, id int64) error {
	return s.repo.RestoreContract(ctx, id)
}
