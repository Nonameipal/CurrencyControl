package service

import (
	"context"
	"fmt"
	"time"

	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/service/ports"
)

type gtdExtensionService struct {
	repo     ports.GTDExtensionRepository
	auditSvc ports.AuditLogService
}

func NewGTDExtensionService(repo ports.GTDExtensionRepository, auditSvc ports.AuditLogService) ports.GTDExtensionService {
	return &gtdExtensionService{
		repo:     repo,
		auditSvc: auditSvc,
	}
}

func (s *gtdExtensionService) CreateRequest(ctx context.Context, login, role string, gtdID int64, requestedDeadline time.Time, documentPath string) (domain.GTDExtensionRequest, error) {
	switch role {
	case domain.RoleOperator, domain.RoleCompliance, domain.RoleAdmin:
	default:
		return domain.GTDExtensionRequest{}, fmt.Errorf("только операционист или администратор может подавать заявку на увеличение срока ГТД")
	}

	req := domain.GTDExtensionRequest{
		GTDID:             gtdID,
		RequestedDeadline: requestedDeadline,
		DocumentPath:      documentPath,
		CreatedBy:         login,
	}

	created, err := s.repo.CreateRequest(ctx, req)
	if err != nil {
		return domain.GTDExtensionRequest{}, err
	}

	if s.auditSvc != nil {
		s.auditSvc.Log(ctx, login, role, nil, "REQUEST_GTD_EXTENSION", "gtd_extension_request", &created.ID,
			fmt.Sprintf("Запрос на увеличение срока ГТД №%d до %s", gtdID, requestedDeadline.Format("02.01.2006")), "")
	}

	return created, nil
}

func (s *gtdExtensionService) UpdateRequest(ctx context.Context, login, role string, id int64, requestedDeadline time.Time, documentPath string) (*domain.GTDExtensionRequest, error) {
	switch role {
	case domain.RoleOperator, domain.RoleAdmin:
	default:
		return nil, fmt.Errorf("редактировать заявку может только операционист или администратор")
	}

	updated, err := s.repo.UpdateRequest(ctx, id, login, role, requestedDeadline, documentPath)
	if err != nil {
		return nil, err
	}

	if s.auditSvc != nil {
		s.auditSvc.Log(ctx, login, role, nil, "UPDATE_GTD_EXTENSION", "gtd_extension_request", &updated.ID,
			fmt.Sprintf("Обновление заявки на увеличение срока ГТД №%d (новая дата: %s)", updated.GTDID, requestedDeadline.Format("02.01.2006")), "")
	}

	return updated, nil
}

func (s *gtdExtensionService) GetByID(ctx context.Context, id int64) (*domain.GTDExtensionRequest, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *gtdExtensionService) GetByGTDID(ctx context.Context, gtdID int64) ([]domain.GTDExtensionRequest, error) {
	return s.repo.GetByGTDID(ctx, gtdID)
}

func (s *gtdExtensionService) GetPendingRequests(ctx context.Context, stage string, branchID *int, page, pageSize int) ([]domain.GTDExtensionRequest, int, error) {
	return s.repo.GetPendingRequests(ctx, stage, branchID, page, pageSize)
}

func (s *gtdExtensionService) SetCurrencyControlDecision(ctx context.Context, role, login string, id int64, decision string, comment string) (*domain.GTDExtensionRequest, error) {
	switch role {
	case domain.RoleCurrencyControl, domain.RoleCurrencyController, domain.RoleAdmin:
	default:
		return nil, fmt.Errorf("только сотрудники валютного контроля могут рассматривать заявку на этом этапе")
	}

	reviewed, err := s.repo.SetCurrencyControlDecision(ctx, id, decision, comment, login)
	if err != nil {
		return nil, err
	}

	if s.auditSvc != nil {
		s.auditSvc.Log(ctx, login, role, nil, "CURRENCY_CONTROL_DECISION_GTD_EXTENSION", "gtd_extension_request", &reviewed.ID,
			fmt.Sprintf("Валютный контроль вынес решение: %s по заявке на увеличение срока ГТД №%d (комментарий: %s)", decision, reviewed.GTDID, comment), "")
	}

	return reviewed, nil
}

func (s *gtdExtensionService) SetComplianceDecision(ctx context.Context, role, login string, id int64, decision string, comment string) (*domain.GTDExtensionRequest, error) {
	switch role {
	case domain.RoleCompliance, domain.RoleAdmin:
	default:
		return nil, fmt.Errorf("только сотрудники комплаенс-контроля могут рассматривать заявку на этом этапе")
	}

	reviewed, err := s.repo.SetComplianceDecision(ctx, id, decision, comment, login)
	if err != nil {
		return nil, err
	}

	if s.auditSvc != nil {
		s.auditSvc.Log(ctx, login, role, nil, "COMPLIANCE_DECISION_GTD_EXTENSION", "gtd_extension_request", &reviewed.ID,
			fmt.Sprintf("Комплаенс-контроль вынес решение: %s по заявке на увеличение срока ГТД №%d (комментарий: %s)", decision, reviewed.GTDID, comment), "")
	}

	return reviewed, nil
}

func (s *gtdExtensionService) ReviewRequest(ctx context.Context, role, login string, id int64, decision string, comment string) (*domain.GTDExtensionRequest, error) {
	switch role {
	case domain.RoleCompliance:
		return s.SetComplianceDecision(ctx, role, login, id, decision, comment)
	case domain.RoleCurrencyControl, domain.RoleCurrencyController:
		return s.SetCurrencyControlDecision(ctx, role, login, id, decision, comment)
	case domain.RoleAdmin:
		// Администратор может рассматривать заявку на любом этапе
		req, err := s.repo.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		if req.Status == domain.ExtensionStatusPendingCompliance {
			return s.SetComplianceDecision(ctx, role, login, id, decision, comment)
		}
		return s.SetCurrencyControlDecision(ctx, role, login, id, decision, comment)
	default:
		return nil, fmt.Errorf("недостаточно прав для рассмотрения заявки")
	}
}
