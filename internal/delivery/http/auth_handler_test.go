package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/service/ports"
)

type mockAuthService struct {
	history []domain.AccessRequest
}

func (m *mockAuthService) Login(ctx context.Context, login, password string) (ports.LoginResult, error) {
	return ports.LoginResult{}, nil
}
func (m *mockAuthService) Refresh(ctx context.Context, refreshToken string) (ports.LoginResult, error) {
	return ports.LoginResult{}, nil
}
func (m *mockAuthService) RequestAccess(ctx context.Context, login string, branchID int64, role string) (domain.AccessRequest, error) {
	return domain.AccessRequest{}, nil
}
func (m *mockAuthService) GetRequestStatus(ctx context.Context, requestID int64) (*domain.AccessRequest, error) {
	return nil, nil
}
func (m *mockAuthService) GetPendingRequests(ctx context.Context) ([]domain.AccessRequest, error) {
	return nil, nil
}
func (m *mockAuthService) GetAccessRequestsHistory(ctx context.Context) ([]domain.AccessRequest, error) {
	return m.history, nil
}
func (m *mockAuthService) ApproveRequest(ctx context.Context, requestID int64, reviewer string) (ports.LoginResult, error) {
	return ports.LoginResult{}, nil
}
func (m *mockAuthService) RejectRequest(ctx context.Context, requestID int64, reviewer string) error {
	return nil
}
func (m *mockAuthService) ValidateSession(ctx context.Context, token string) (*domain.Session, error) {
	return nil, nil
}
func (m *mockAuthService) Logout(ctx context.Context, token string) error {
	return nil
}

func TestGetAccessRequestsHistory(t *testing.T) {
	now := time.Now()
	mockSvc := &mockAuthService{
		history: []domain.AccessRequest{
			{
				ID:         1,
				Login:      "user1",
				BranchID:   1,
				BranchName: "Головной офис",
				Role:       domain.RoleOperator,
				Status:     "approved",
				ReviewedBy: "admin",
				ReviewedAt: &now,
			},
			{
				ID:         2,
				Login:      "user2",
				BranchID:   2,
				BranchName: "Филиал Худжанд",
				Role:       domain.RoleCurrencyControl,
				Status:     "rejected",
				ReviewedBy: "compliance_user",
				ReviewedAt: &now,
			},
		},
	}

	handler := NewAuthHandler(mockSvc, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/access-requests/history", nil)
	rr := httptest.NewRecorder()

	handler.GetAccessRequestsHistory(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", rr.Code)
	}

	var resp []domain.AccessRequest
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp) != 2 {
		t.Fatalf("expected 2 items, got %d", len(resp))
	}

	if resp[0].ReviewedBy != "admin" || resp[0].Status != "approved" {
		t.Errorf("unexpected first item: %+v", resp[0])
	}
	if resp[1].ReviewedBy != "compliance_user" || resp[1].Status != "rejected" {
		t.Errorf("unexpected second item: %+v", resp[1])
	}
}
