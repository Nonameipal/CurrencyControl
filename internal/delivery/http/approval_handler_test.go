package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"CurrencyControl/internal/delivery/dto"
	delivery "CurrencyControl/internal/delivery/http"
	"CurrencyControl/internal/domain"

	"github.com/gorilla/mux"
)

type mockApprovalService struct {
	lastCCDecision    dto.CurrencyControlDecisionRequest
	lastCompDecision  dto.ComplianceDecisionRequest
	ccReviewReturn    *dto.ApprovalItemResponse
	compReviewReturn  *dto.ApprovalItemResponse
	pendingResp       *dto.PendingApprovalsResponse
	detailResp        *dto.ApprovalItemResponse
	ccReviewErr       error
	compReviewErr     error
	pendingErr        error
	detailErr         error
}

func (m *mockApprovalService) ReviewCurrencyControl(ctx context.Context, role, login string, entityType string, id int64, req dto.CurrencyControlDecisionRequest) (*dto.ApprovalItemResponse, error) {
	if m.ccReviewErr != nil {
		return nil, m.ccReviewErr
	}
	m.lastCCDecision = req
	if m.ccReviewReturn != nil {
		return m.ccReviewReturn, nil
	}
	return &dto.ApprovalItemResponse{
		EntityType:     entityType,
		EntityID:       id,
		ApprovalStatus: domain.ApprovalStatusPendingCompliance,
	}, nil
}

func (m *mockApprovalService) ReviewCompliance(ctx context.Context, role, login string, entityType string, id int64, req dto.ComplianceDecisionRequest) (*dto.ApprovalItemResponse, error) {
	if m.compReviewErr != nil {
		return nil, m.compReviewErr
	}
	m.lastCompDecision = req
	if m.compReviewReturn != nil {
		return m.compReviewReturn, nil
	}
	return &dto.ApprovalItemResponse{
		EntityType:     entityType,
		EntityID:       id,
		ApprovalStatus: domain.ApprovalStatusApproved,
	}, nil
}

func (m *mockApprovalService) GetPendingApprovals(ctx context.Context, role string, userBranchID int64, filter dto.PendingApprovalsFilter) (*dto.PendingApprovalsResponse, error) {
	if m.pendingErr != nil {
		return nil, m.pendingErr
	}
	if m.pendingResp != nil {
		return m.pendingResp, nil
	}
	return &dto.PendingApprovalsResponse{
		Items:    []dto.ApprovalItemResponse{},
		Total:    0,
		Page:     1,
		PageSize: 20,
	}, nil
}

func (m *mockApprovalService) GetApprovalDetail(ctx context.Context, entityType string, id int64) (*dto.ApprovalItemResponse, error) {
	if m.detailErr != nil {
		return nil, m.detailErr
	}
	if m.detailResp != nil {
		return m.detailResp, nil
	}
	return &dto.ApprovalItemResponse{
		EntityType:     entityType,
		EntityID:       id,
		ApprovalStatus: domain.ApprovalStatusPendingCurrencyControl,
	}, nil
}

func TestApprovalHandler_ReviewCurrencyControl(t *testing.T) {
	mockSvc := &mockApprovalService{}
	handler := delivery.NewApprovalHandler(mockSvc)

	r := mux.NewRouter()
	r.HandleFunc("/api/approvals/{entity_type}/{id:[0-9]+}/currency-control", handler.ReviewCurrencyControl).Methods(http.MethodPost)

	t.Run("CurrencyControl officer can accept", func(t *testing.T) {
		body, _ := json.Marshal(dto.CurrencyControlDecisionRequest{
			Decision: "accepted",
			Comment:  "All good",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/approvals/contract/1/currency-control", bytes.NewReader(body))
		ctx := context.WithValue(req.Context(), delivery.RoleContextKey, domain.RoleCurrencyControl)
		ctx = context.WithValue(ctx, delivery.LoginContextKey, "cc_user")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("Operator cannot review at currency control stage", func(t *testing.T) {
		body, _ := json.Marshal(dto.CurrencyControlDecisionRequest{
			Decision: "accepted",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/approvals/contract/1/currency-control", bytes.NewReader(body))
		ctx := context.WithValue(req.Context(), delivery.RoleContextKey, domain.RoleOperator)
		ctx = context.WithValue(ctx, delivery.LoginContextKey, "operator_user")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", rec.Code)
		}
	})
}

func TestApprovalHandler_ReviewCompliance(t *testing.T) {
	mockSvc := &mockApprovalService{}
	handler := delivery.NewApprovalHandler(mockSvc)

	r := mux.NewRouter()
	r.HandleFunc("/api/approvals/{entity_type}/{id:[0-9]+}/compliance", handler.ReviewCompliance).Methods(http.MethodPost)

	t.Run("Compliance officer can approve", func(t *testing.T) {
		body, _ := json.Marshal(dto.ComplianceDecisionRequest{
			Decision: "approve",
			Comment:  "Approved",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/approvals/contract/1/compliance", bytes.NewReader(body))
		ctx := context.WithValue(req.Context(), delivery.RoleContextKey, domain.RoleCompliance)
		ctx = context.WithValue(ctx, delivery.LoginContextKey, "comp_user")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("Compliance reject without comment returns 400 Bad Request", func(t *testing.T) {
		mockSvc.compReviewErr = errors.New("причина отказа обязательна для заполнения")

		body, _ := json.Marshal(dto.ComplianceDecisionRequest{
			Decision: "reject",
			Comment:  "",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/approvals/contract/1/compliance", bytes.NewReader(body))
		ctx := context.WithValue(req.Context(), delivery.RoleContextKey, domain.RoleCompliance)
		ctx = context.WithValue(ctx, delivery.LoginContextKey, "comp_user")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request on missing rejection comment, got %d", rec.Code)
		}
		mockSvc.compReviewErr = nil
	})

	t.Run("CurrencyControl cannot review compliance stage", func(t *testing.T) {
		body, _ := json.Marshal(dto.ComplianceDecisionRequest{
			Decision: "approve",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/approvals/contract/1/compliance", bytes.NewReader(body))
		ctx := context.WithValue(req.Context(), delivery.RoleContextKey, domain.RoleCurrencyControl)
		ctx = context.WithValue(ctx, delivery.LoginContextKey, "cc_user")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden, got %d", rec.Code)
		}
	})
}

func TestApprovalHandler_GetPendingAndDetail(t *testing.T) {
	mockSvc := &mockApprovalService{}
	handler := delivery.NewApprovalHandler(mockSvc)

	r := mux.NewRouter()
	r.HandleFunc("/api/approvals/pending", handler.GetPendingApprovals).Methods(http.MethodGet)
	r.HandleFunc("/api/approvals/{entity_type}/{id:[0-9]+}", handler.GetApprovalDetail).Methods(http.MethodGet)

	t.Run("GetPendingApprovals returns 200", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/approvals/pending?stage=currency_control", nil)
		ctx := context.WithValue(req.Context(), delivery.RoleContextKey, domain.RoleCurrencyControl)
		ctx = context.WithValue(ctx, delivery.LoginContextKey, "cc_user")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("GetApprovalDetail returns 200", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/approvals/contract/123", nil)
		ctx := context.WithValue(req.Context(), delivery.LoginContextKey, "any_user")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})
}
