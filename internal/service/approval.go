package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"CurrencyControl/internal/delivery/dto"
	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/service/ports"
)

type approvalService struct {
	repo     ports.ApprovalRepository
	auditSvc ports.AuditLogService
}

func NewApprovalService(repo ports.ApprovalRepository, auditSvc ports.AuditLogService) ports.ApprovalService {
	return &approvalService{
		repo:     repo,
		auditSvc: auditSvc,
	}
}

func checkApprovalEntityType(entityType string) error {
	switch strings.ToLower(strings.TrimSpace(entityType)) {
	case domain.EntityTypeGTD, domain.EntityTypePaymentOrder:
		return nil
	default:
		return fmt.Errorf("санкции применяются только к платёжным поручениям и ГТД")
	}
}

func validateDecision(decision, comment, positiveDecision, stageName string) (string, error) {
	norm := strings.ToLower(strings.TrimSpace(decision))
	switch norm {
	case positiveDecision:
	case "revision":
		if strings.TrimSpace(comment) == "" {
			return "", errors.New("причина отправки на доработку обязательна для заполнения")
		}
	case "rejected", "reject":
		if strings.TrimSpace(comment) == "" {
			return "", errors.New("причина отклонения обязательна для заполнения")
		}
	default:
		rejectWord := "rejected"
		if positiveDecision == "approve" {
			rejectWord = "reject"
		}
		return "", fmt.Errorf("недопустимое решение %s: '%s' (допустимы: %s, revision, %s)",
			stageName, decision, positiveDecision, rejectWord)
	}
	return norm, nil
}

func (s *approvalService) logAudit(ctx context.Context, login, role string, item *dto.ApprovalItemResponse, action, entityType string, id int64, details string) {
	if s.auditSvc == nil {
		return
	}
	var bID *int64
	if item != nil && item.BranchID != nil {
		val := int64(*item.BranchID)
		bID = &val
	}
	s.auditSvc.Log(ctx, login, role, bID, action, entityType, &id, details, "")
}

func (s *approvalService) ReviewCurrencyControl(ctx context.Context, role, login string, entityType string, id int64, req dto.CurrencyControlDecisionRequest) (*dto.ApprovalItemResponse, error) {
	switch role {
	case domain.RoleCurrencyControl, domain.RoleCurrencyController, domain.RoleAdmin:
	default:
		return nil, errors.New("только сотрудники отдела валютного контроля могут выносить решение на данном этапе")
	}

	if err := checkApprovalEntityType(entityType); err != nil {
		return nil, err
	}
	norm, err := validateDecision(req.Decision, req.Comment, "accepted", "валютного контроля")
	if err != nil {
		return nil, err
	}
	item, err := s.repo.SetCurrencyControlDecision(ctx, entityType, id, norm, req.Comment, login)
	if err != nil {
		return nil, err
	}
	s.logAudit(ctx, login, role, item, "REVIEW_CURRENCY_CONTROL", entityType, id,
		fmt.Sprintf("Валютный контроль вынес решение: %s. Комментарий: %s", norm, req.Comment))

	return item, nil
}

func (s *approvalService) ReviewCompliance(ctx context.Context, role, login string, entityType string, id int64, req dto.ComplianceDecisionRequest) (*dto.ApprovalItemResponse, error) {
	switch role {
	case domain.RoleCompliance, domain.RoleAdmin:
	default:
		return nil, errors.New("только сотрудники отдела комплаенс могут выносить решение на данном этапе")
	}
	if err := checkApprovalEntityType(entityType); err != nil {
		return nil, err
	}
	norm, err := validateDecision(req.Decision, req.Comment, "approve", "комплаенс")
	if err != nil {
		return nil, err
	}

	item, err := s.repo.SetComplianceDecision(ctx, entityType, id, norm, req.Comment, login)
	if err != nil {
		return nil, err
	}
	s.logAudit(ctx, login, role, item, "REVIEW_COMPLIANCE", entityType, id,
		fmt.Sprintf("Комплаенс вынес решение: %s. Причина/комментарий: %s", norm, req.Comment))

	return item, nil
}

func (s *approvalService) GetPendingApprovals(ctx context.Context, role string, userBranchID int64, filter dto.PendingApprovalsFilter) (*dto.PendingApprovalsResponse, error) {
	if role == domain.RoleBranchHead && userBranchID > 0 {
		bID := int(userBranchID)
		filter.BranchID = &bID
	}

	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 || filter.PageSize > 100 {
		filter.PageSize = 20
	}

	items, total, err := s.repo.GetPendingApprovals(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &dto.PendingApprovalsResponse{
		Items:    items,
		Total:    total,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}, nil
}

func (s *approvalService) GetApprovalDetail(ctx context.Context, entityType string, id int64) (*dto.ApprovalItemResponse, error) {
	return s.repo.GetApprovalDetail(ctx, entityType, id)
}

func (s *approvalService) GetAcceptedApprovals(ctx context.Context, role string, filter dto.AcceptedApprovalsFilter) (*dto.AcceptedApprovalsResponse, error) {
	if role != domain.RoleAdmin {
		return nil, errors.New("доступ разрешён только администраторам")
	}

	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 || filter.PageSize > 100 {
		filter.PageSize = 20
	}

	items, total, err := s.repo.GetAcceptedApprovals(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &dto.AcceptedApprovalsResponse{
		Items:    items,
		Total:    total,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}, nil
}
