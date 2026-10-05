package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"

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
func (m *mockAuthService) RequestAccess(ctx context.Context, login, lastName, firstName, email string, branchID int64, role string) (domain.AccessRequest, error) {
	if role == domain.RoleAdmin {
		return domain.AccessRequest{}, fmt.Errorf("роль 'admin' нельзя запросить через интерфейс")
	}
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
func (m *mockAuthService) GetAllUsers(ctx context.Context) ([]domain.User, error) {
	return []domain.User{
		{ID: 1, Login: "operator1", Role: domain.RoleOperator, BranchID: 1, BranchName: "Головной офис"},
	}, nil
}
func (m *mockAuthService) CreateUser(ctx context.Context, req domain.CreateUserRequest) (domain.User, error) {
	if req.Role == domain.RoleAdmin {
		return domain.User{}, fmt.Errorf("роль 'admin' нельзя назначить через интерфейс")
	}
	if req.Login == "duplicate" {
		return domain.User{}, fmt.Errorf("пользователь с логином '%s' уже существует", req.Login)
	}
	return domain.User{ID: 2, Login: req.Login, Role: req.Role, BranchID: req.BranchID}, nil
}
func (m *mockAuthService) UpdateUser(ctx context.Context, id int64, req domain.UpdateUserRequest) (domain.User, error) {
	if req.Role == domain.RoleAdmin {
		return domain.User{}, fmt.Errorf("роль 'admin' нельзя назначить через интерфейс")
	}
	return domain.User{ID: id, Login: "user1", Role: req.Role, BranchID: req.BranchID}, nil
}
func (m *mockAuthService) DeleteUser(ctx context.Context, id int64) error {
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

func TestUserManagement(t *testing.T) {
	mockSvc := &mockAuthService{}
	handler := NewAuthHandler(mockSvc, nil)

	// 1. GetUsers
	reqGet := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	rrGet := httptest.NewRecorder()
	handler.GetUsers(rrGet, reqGet)
	if rrGet.Code != http.StatusOK {
		t.Errorf("GetUsers: expected 200, got %d", rrGet.Code)
	}

	// 2. CreateUser
	bodyCreate := `{"login":"testuser","role":"operator","branch_id":1}`
	reqPost := httptest.NewRequest(http.MethodPost, "/api/users", strings.NewReader(bodyCreate))
	rrPost := httptest.NewRecorder()
	handler.CreateUser(rrPost, reqPost)
	if rrPost.Code != http.StatusCreated {
		t.Errorf("CreateUser: expected 201, got %d", rrPost.Code)
	}

	// 3. UpdateUser
	bodyUpdate := `{"role":"branch_head","branch_id":2}`
	reqPut := httptest.NewRequest(http.MethodPut, "/api/users/1", strings.NewReader(bodyUpdate))
	reqPut = mux.SetURLVars(reqPut, map[string]string{"id": "1"})
	rrPut := httptest.NewRecorder()
	handler.UpdateUser(rrPut, reqPut)
	if rrPut.Code != http.StatusOK {
		t.Errorf("UpdateUser: expected 200, got %d", rrPut.Code)
	}

	// 4. DeleteUser
	reqDel := httptest.NewRequest(http.MethodDelete, "/api/users/1", nil)
	reqDel = mux.SetURLVars(reqDel, map[string]string{"id": "1"})
	rrDel := httptest.NewRecorder()
	handler.DeleteUser(rrDel, reqDel)
	if rrDel.Code != http.StatusOK {
		t.Errorf("DeleteUser: expected 200, got %d", rrDel.Code)
	}
}

func TestAdminRoleForbidden(t *testing.T) {
	mockSvc := &mockAuthService{}
	handler := NewAuthHandler(mockSvc, nil)

	// 1. CreateUser с ролью admin должен возвращать 400
	bodyCreate := `{"login":"superman","role":"admin","branch_id":1}`
	reqPost := httptest.NewRequest(http.MethodPost, "/api/users", strings.NewReader(bodyCreate))
	rrPost := httptest.NewRecorder()
	handler.CreateUser(rrPost, reqPost)
	if rrPost.Code != http.StatusBadRequest {
		t.Errorf("CreateUser with admin role: expected 400, got %d", rrPost.Code)
	}

	// 2. UpdateUser с ролью admin должен возвращать 400
	bodyUpdate := `{"role":"admin","branch_id":1}`
	reqPut := httptest.NewRequest(http.MethodPut, "/api/users/1", strings.NewReader(bodyUpdate))
	reqPut = mux.SetURLVars(reqPut, map[string]string{"id": "1"})
	rrPut := httptest.NewRecorder()
	handler.UpdateUser(rrPut, reqPut)
	if rrPut.Code != http.StatusBadRequest {
		t.Errorf("UpdateUser with admin role: expected 400, got %d", rrPut.Code)
	}
}

func TestCreateUserDuplicate(t *testing.T) {
	mockSvc := &mockAuthService{}
	handler := NewAuthHandler(mockSvc, nil)

	bodyCreate := `{"login":"duplicate","role":"operator","branch_id":1}`
	reqPost := httptest.NewRequest(http.MethodPost, "/api/users", strings.NewReader(bodyCreate))
	rrPost := httptest.NewRecorder()
	handler.CreateUser(rrPost, reqPost)
	if rrPost.Code != http.StatusBadRequest {
		t.Errorf("CreateUser with duplicate login: expected 400, got %d", rrPost.Code)
	}
}
