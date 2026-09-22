package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"CurrencyControl/internal/delivery/dto"
	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/service"
)

type mockApprovalRepo struct {
	lastDecision string
	lastComment  string
	lastReviewer string
	lastStatus   string
	itemToReturn *dto.ApprovalItemResponse
	returnError  error
	pendingTotal int
	pendingItems []dto.ApprovalItemResponse
	resetCalled  bool
}

func (m *mockApprovalRepo) SetCurrencyControlDecision(ctx context.Context, entityType string, id int64, decision, comment, reviewer string) (*dto.ApprovalItemResponse, error) {
	if m.returnError != nil {
		return nil, m.returnError
	}
	m.lastDecision = decision
	m.lastComment = comment
	m.lastReviewer = reviewer
	var newStatus string
	switch decision {
	case "accepted", "accept":
		newStatus = domain.ApprovalStatusPendingCompliance
	case "revision":
		newStatus = domain.ApprovalStatusRevisionRequired
	case "rejected", "reject":
		newStatus = domain.ApprovalStatusRejectedCurrencyControl
	}
	m.lastStatus = newStatus
	return &dto.ApprovalItemResponse{
		EntityType:                entityType,
		EntityID:                  id,
		ApprovalStatus:            newStatus,
		CurrencyControlDecision:   decision,
		CurrencyControlComment:    comment,
		CurrencyControlReviewedBy: reviewer,
	}, nil
}

func (m *mockApprovalRepo) SetComplianceDecision(ctx context.Context, entityType string, id int64, decision, comment, reviewer string) (*dto.ApprovalItemResponse, error) {
	if m.returnError != nil {
		return nil, m.returnError
	}
	m.lastDecision = decision
	m.lastComment = comment
	m.lastReviewer = reviewer
	var newStatus string
	switch decision {
	case "approve":
		newStatus = domain.ApprovalStatusApproved
	case "reject":
		newStatus = domain.ApprovalStatusRejectedCompliance
	}
	m.lastStatus = newStatus
	return &dto.ApprovalItemResponse{
		EntityType:           entityType,
		EntityID:             id,
		ApprovalStatus:       newStatus,
		ComplianceDecision:   decision,
		ComplianceComment:    comment,
		ComplianceReviewedBy: reviewer,
		RejectionReason:      comment,
	}, nil
}

func (m *mockApprovalRepo) GetPendingApprovals(ctx context.Context, filter dto.PendingApprovalsFilter) ([]dto.ApprovalItemResponse, int, error) {
	if m.returnError != nil {
		return nil, 0, m.returnError
	}
	return m.pendingItems, m.pendingTotal, nil
}

func (m *mockApprovalRepo) GetApprovalDetail(ctx context.Context, entityType string, id int64) (*dto.ApprovalItemResponse, error) {
	if m.returnError != nil {
		return nil, m.returnError
	}
	if m.itemToReturn != nil {
		return m.itemToReturn, nil
	}
	return &dto.ApprovalItemResponse{
		EntityType:     entityType,
		EntityID:       id,
		ApprovalStatus: domain.ApprovalStatusPendingCurrencyControl,
	}, nil
}

func (m *mockApprovalRepo) ResetToPendingCurrencyControl(ctx context.Context, entityType string, id int64) error {
	m.resetCalled = true
	return m.returnError
}

func TestApprovalService_CurrencyControl(t *testing.T) {
	ctx := context.Background()

	t.Run("Accept decision moves to pending_compliance", func(t *testing.T) {
		repo := &mockApprovalRepo{}
		svc := service.NewApprovalService(repo, nil)

		res, err := svc.ReviewCurrencyControl(ctx, domain.RoleCurrencyControl, "cc_officer", domain.EntityTypeContract, 10, dto.CurrencyControlDecisionRequest{
			Decision: "accepted",
			Comment:  "Предмет контракта проверен и соответствует требованиям",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.ApprovalStatus != domain.ApprovalStatusPendingCompliance {
			t.Errorf("expected approval status '%s', got '%s'", domain.ApprovalStatusPendingCompliance, res.ApprovalStatus)
		}
		if repo.lastDecision != "accepted" {
			t.Errorf("expected normalized decision 'accepted', got '%s'", repo.lastDecision)
		}
		if repo.lastReviewer != "cc_officer" {
			t.Errorf("expected reviewer 'cc_officer', got '%s'", repo.lastReviewer)
		}
	})

	t.Run("Revision decision moves to revision_required", func(t *testing.T) {
		repo := &mockApprovalRepo{}
		svc := service.NewApprovalService(repo, nil)

		res, err := svc.ReviewCurrencyControl(ctx, domain.RoleCurrencyController, "cc_head", domain.EntityTypeInvoice, 25, dto.CurrencyControlDecisionRequest{
			Decision: "revision",
			Comment:  "Уточните назначение платежа и код ТН ВЭД",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.ApprovalStatus != domain.ApprovalStatusRevisionRequired {
			t.Errorf("expected approval status '%s', got '%s'", domain.ApprovalStatusRevisionRequired, res.ApprovalStatus)
		}
		if repo.lastDecision != "revision" {
			t.Errorf("expected normalized decision 'revision', got '%s'", repo.lastDecision)
		}
	})

	t.Run("Reject decision moves to rejected_currency_control", func(t *testing.T) {
		repo := &mockApprovalRepo{}
		svc := service.NewApprovalService(repo, nil)

		res, err := svc.ReviewCurrencyControl(ctx, domain.RoleCurrencyControl, "cc_officer", domain.EntityTypeGTD, 42, dto.CurrencyControlDecisionRequest{
			Decision: "rejected",
			Comment:  "Сумма в ГТД не соответствует инвойсу",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.ApprovalStatus != domain.ApprovalStatusRejectedCurrencyControl {
			t.Errorf("expected approval status '%s', got '%s'", domain.ApprovalStatusRejectedCurrencyControl, res.ApprovalStatus)
		}
		if repo.lastDecision != "rejected" {
			t.Errorf("expected normalized decision 'rejected', got '%s'", repo.lastDecision)
		}
	})

	t.Run("Invalid decision rejected", func(t *testing.T) {
		repo := &mockApprovalRepo{}
		svc := service.NewApprovalService(repo, nil)

		_, err := svc.ReviewCurrencyControl(ctx, domain.RoleCurrencyControl, "cc_officer", domain.EntityTypeContract, 10, dto.CurrencyControlDecisionRequest{
			Decision: "unknown_decision",
		})
		if err == nil {
			t.Fatal("expected error on invalid decision, got nil")
		}
		if !strings.Contains(err.Error(), "недопустимое решение") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("Unauthorized role blocked", func(t *testing.T) {
		repo := &mockApprovalRepo{}
		svc := service.NewApprovalService(repo, nil)

		_, err := svc.ReviewCurrencyControl(ctx, domain.RoleOperator, "operator1", domain.EntityTypeContract, 10, dto.CurrencyControlDecisionRequest{
			Decision: "accept",
		})
		if err == nil {
			t.Fatal("expected error for unauthorized role, got nil")
		}
		if !strings.Contains(err.Error(), "только сотрудники отдела валютного контроля") {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}

func TestApprovalService_Compliance(t *testing.T) {
	ctx := context.Background()

	t.Run("Approve decision moves to approved", func(t *testing.T) {
		repo := &mockApprovalRepo{}
		svc := service.NewApprovalService(repo, nil)

		res, err := svc.ReviewCompliance(ctx, domain.RoleCompliance, "compliance_officer", domain.EntityTypeContract, 10, dto.ComplianceDecisionRequest{
			Decision: "approve",
			Comment:  "Согласовано комплаенс-контролем",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.ApprovalStatus != domain.ApprovalStatusApproved {
			t.Errorf("expected approval status '%s', got '%s'", domain.ApprovalStatusApproved, res.ApprovalStatus)
		}
		if repo.lastDecision != "approve" {
			t.Errorf("expected normalized decision 'approve', got '%s'", repo.lastDecision)
		}
	})

	t.Run("Reject decision with comment sets rejected_compliance", func(t *testing.T) {
		repo := &mockApprovalRepo{}
		svc := service.NewApprovalService(repo, nil)

		res, err := svc.ReviewCompliance(ctx, domain.RoleCompliance, "compliance_officer", domain.EntityTypeContract, 10, dto.ComplianceDecisionRequest{
			Decision: "reject",
			Comment:  "Контрагент включен в санкционный список",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.ApprovalStatus != domain.ApprovalStatusRejectedCompliance {
			t.Errorf("expected approval status '%s', got '%s'", domain.ApprovalStatusRejectedCompliance, res.ApprovalStatus)
		}
		if repo.lastDecision != "reject" {
			t.Errorf("expected normalized decision 'reject', got '%s'", repo.lastDecision)
		}
		if res.RejectionReason != "Контрагент включен в санкционный список" {
			t.Errorf("expected rejection reason, got '%s'", res.RejectionReason)
		}
	})

	t.Run("Reject decision WITHOUT comment MUST be blocked with error", func(t *testing.T) {
		repo := &mockApprovalRepo{}
		svc := service.NewApprovalService(repo, nil)

		_, err := svc.ReviewCompliance(ctx, domain.RoleCompliance, "compliance_officer", domain.EntityTypeContract, 10, dto.ComplianceDecisionRequest{
			Decision: "reject",
			Comment:  "   ", // empty / whitespace only
		})
		if err == nil {
			t.Fatal("expected rejection without reason to fail, got nil")
		}
		if !strings.Contains(err.Error(), "причина отказа обязательна для заполнения") {
			t.Errorf("expected mandatory rejection reason error, got: %v", err)
		}
	})

	t.Run("Unauthorized role blocked", func(t *testing.T) {
		repo := &mockApprovalRepo{}
		svc := service.NewApprovalService(repo, nil)

		_, err := svc.ReviewCompliance(ctx, domain.RoleCurrencyControl, "cc_user", domain.EntityTypeContract, 10, dto.ComplianceDecisionRequest{
			Decision: "approve",
		})
		if err == nil {
			t.Fatal("expected error for unauthorized role, got nil")
		}
		if !strings.Contains(err.Error(), "только сотрудники отдела комплаенс-контроля") {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}

func TestApprovalService_GetPendingApprovals(t *testing.T) {
	ctx := context.Background()

	t.Run("BranchHead filter automatically restricted to branch", func(t *testing.T) {
		repo := &mockApprovalRepo{
			pendingTotal: 1,
			pendingItems: []dto.ApprovalItemResponse{
				{
					EntityType:     "contract",
					EntityID:       10,
					ApprovalStatus: domain.ApprovalStatusPendingCurrencyControl,
				},
			},
		}
		svc := service.NewApprovalService(repo, nil)

		filter := dto.PendingApprovalsFilter{
			Page:     1,
			PageSize: 10,
		}
		res, err := svc.GetPendingApprovals(ctx, domain.RoleBranchHead, 5, filter)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Total != 1 {
			t.Errorf("expected total 1, got %d", res.Total)
		}
	})

	t.Run("Repo error propagated", func(t *testing.T) {
		repo := &mockApprovalRepo{
			returnError: errors.New("db error"),
		}
		svc := service.NewApprovalService(repo, nil)

		_, err := svc.GetPendingApprovals(ctx, domain.RoleAdmin, 0, dto.PendingApprovalsFilter{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
