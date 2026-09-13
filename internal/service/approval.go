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

func (s *approvalService) ReviewCurrencyControl(ctx context.Context, role, login string, entityType string, id int64, req dto.CurrencyControlDecisionRequest) (*dto.ApprovalItemResponse, error) {
	switch role {
	case domain.RoleCurrencyControl, domain.RoleCurrencyController, domain.RoleAdmin:
	default:
		return nil, errors.New("только сотрудники отдела валютного контроля могут выносить решение на данном этапе")
	}

	norm := strings.ToLower(strings.TrimSpace(req.Decision))
	switch norm {
	case "accepted":
	case "revision":
	case "rejected":
	default:
		return nil, fmt.Errorf("недопустимое решение валютного контроля: '%s' (допустимы: accepted, revision, rejected)", req.Decision)
	}

	item, err := s.repo.SetCurrencyControlDecision(ctx, entityType, id, norm, req.Comment, login)
	if err != nil {
		return nil, err
	}

	if s.auditSvc != nil {
		var bID *int64
		if item != nil && item.BranchID != nil {
			val := int64(*item.BranchID)
			bID = &val
		}
		s.auditSvc.Log(ctx, login, role, bID, "REVIEW_CURRENCY_CONTROL", entityType, &id,
			fmt.Sprintf("Валютный контроль вынес решение: %s. Комментарий: %s", norm, req.Comment), "")
	}

	return item, nil
}

func (s *approvalService) ReviewCompliance(ctx context.Context, role, login string, entityType string, id int64, req dto.ComplianceDecisionRequest) (*dto.ApprovalItemResponse, error) {
	switch role {
	case domain.RoleCompliance, domain.RoleAdmin:
	default:
		return nil, errors.New("только сотрудники отдела комплаенс-контроля могут выносить решение на данном этапе")
	}

	norm := strings.ToLower(strings.TrimSpace(req.Decision))
	switch norm {
	case "approve":
	case "reject":
		if strings.TrimSpace(req.Comment) == "" {
			return nil, errors.New("причина отказа обязательна для заполнения")
		}
	default:
		return nil, fmt.Errorf("недопустимое решение комплаенс-контроля: '%s' (допустимы: approve, reject)", req.Decision)
	}

	item, err := s.repo.SetComplianceDecision(ctx, entityType, id, norm, req.Comment, login)
	if err != nil {
		return nil, err
	}

	if s.auditSvc != nil {
		var bID *int64
		if item != nil && item.BranchID != nil {
			val := int64(*item.BranchID)
			bID = &val
		}
		s.auditSvc.Log(ctx, login, role, bID, "REVIEW_COMPLIANCE", entityType, &id,
			fmt.Sprintf("Комплаенс-контроль вынес решение: %s. Причина/комментарий: %s", norm, req.Comment), "")
	}

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
