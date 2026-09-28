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
	if err := input.ValidateDates(); err != nil {
		return domain.Contract{}, err
	}
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
	if err := c.ValidateDates(); err != nil {
		return domain.Contract{}, err
	}
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return domain.Contract{}, err
	}
	switch existing.ApprovalStatus {
	case domain.ApprovalStatusPendingCurrencyControl:
		return domain.Contract{}, fmt.Errorf("документ на рассмотрении валютного контроля  редактирование запрещено")
	case domain.ApprovalStatusPendingCompliance:
		return domain.Contract{}, fmt.Errorf("документ на рассмотрении комплаенса  редактирование запрещено")
	case domain.ApprovalStatusRevisionRequired:
		if c.UpdatedBy != existing.CreatedBy {
			return domain.Contract{}, fmt.Errorf("редактировать документ на доработке может только его создатель")
		}
		updated, err := s.repo.Update(ctx, id, c)
		if err != nil {
			return domain.Contract{}, err
		}
		_ = s.repo.ResetApprovalStatus(ctx, id)
		return updated, nil
	case domain.ApprovalStatusApproved:
	case domain.ApprovalStatusRejectedCurrencyControl, domain.ApprovalStatusRejectedCompliance:
		return domain.Contract{}, fmt.Errorf("документ отклонён и перемещён в корзину — редактирование невозможно")
	}
	return s.repo.Update(ctx, id, c)
}

func (s *contractService) SoftDelete(ctx context.Context, id int64) error {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	switch existing.ApprovalStatus {
	case domain.ApprovalStatusPendingCurrencyControl:
		return fmt.Errorf("документ на рассмотрении валютного контроля удаление запрещено")
	case domain.ApprovalStatusPendingCompliance:
		return fmt.Errorf("документ на рассмотрении комплаенса удаление запрещено")
	case domain.ApprovalStatusRevisionRequired:
		return fmt.Errorf("нельзя удалить документ, отправленный на доработку")
	case domain.ApprovalStatusRejectedCurrencyControl, domain.ApprovalStatusRejectedCompliance:
		return fmt.Errorf("документ уже в корзине")
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
