package service

import (
	"context"
	"errors"

	"CurrencyControl/internal/delivery/dto"
	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/repository"
)

type ReportService interface {
	GetContractsReport(ctx context.Context, userRole string, userBranchID int64, filter repository.ReportFilter) (*dto.ContractsReportResponse, error)
}

type reportService struct {
	repo repository.ReportRepository
}

func NewReportService(repo repository.ReportRepository) ReportService {
	return &reportService{repo: repo}
}

func (s *reportService) GetContractsReport(ctx context.Context, userRole string, userBranchID int64, filter repository.ReportFilter) (*dto.ContractsReportResponse, error) {
	switch userRole {
	case domain.RoleBranchHead:
		if filter.BranchID != nil && int64(*filter.BranchID) != userBranchID {
			return nil, errors.New("начальник подразделения имеет доступ к формированию отчетности только своего подразделения")
		}
		branchInt := int(userBranchID)
		filter.BranchID = &branchInt

	case domain.RoleCurrencyControl, domain.RoleCurrencyController, domain.RoleCompliance, domain.RoleInternalAudit, domain.RoleAdmin:

	default:
		return nil, errors.New("недостаточно прав для формирования отчетности")
	}

	return s.repo.GetContractsReport(ctx, filter)
}
