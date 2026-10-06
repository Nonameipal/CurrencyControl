package http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"CurrencyControl/internal/delivery/dto"
	delivery "CurrencyControl/internal/delivery/http"
	"CurrencyControl/internal/domain"
	"github.com/gorilla/mux"
)

type mockContractServiceForSearch struct {
	lastClientID int64
	lastFilter   dto.ContractFilter
	contracts    []domain.Contract
	err          error
}

func (m *mockContractServiceForSearch) Create(ctx context.Context, login string, input domain.Contract) (domain.Contract, error) {
	return domain.Contract{}, nil
}
func (m *mockContractServiceForSearch) GetByID(ctx context.Context, login string, id int64) (domain.Contract, error) {
	return domain.Contract{}, nil
}
func (m *mockContractServiceForSearch) GetByClientID(ctx context.Context, login string, clientID int64, filter dto.ContractFilter) ([]domain.Contract, error) {
	m.lastClientID = clientID
	m.lastFilter = filter
	if m.err != nil {
		return nil, m.err
	}
	return m.contracts, nil
}
func (m *mockContractServiceForSearch) GetArchived(ctx context.Context, branchID int, page, pageSize int) ([]domain.Contract, int, error) {
	return nil, 0, nil
}
func (m *mockContractServiceForSearch) GetArchivedByClientID(ctx context.Context, clientID int64) ([]domain.Contract, error) {
	return nil, nil
}
func (m *mockContractServiceForSearch) RestoreContract(ctx context.Context, id int64) error {
	return nil
}
func (m *mockContractServiceForSearch) SearchDashboard(ctx context.Context, login string, req dto.DashboardSearchRequest) ([]dto.DashboardSearchResult, error) {
	return nil, nil
}
func (m *mockContractServiceForSearch) CheckCountry(ctx context.Context, name string) (bool, error) {
	return true, nil
}
func (m *mockContractServiceForSearch) CheckCurrency(ctx context.Context, code string) (bool, error) {
	return true, nil
}
func (m *mockContractServiceForSearch) GetExpiringContracts(ctx context.Context, branchID int) ([]dto.NotificationResponse, error) {
	return nil, nil
}
func (m *mockContractServiceForSearch) Update(ctx context.Context, id int64, c domain.Contract) (domain.Contract, error) {
	return c, nil
}
func (m *mockContractServiceForSearch) SoftDelete(ctx context.Context, id int64) error {
	return nil
}

func TestGetContractsByCompany_AmountSearch(t *testing.T) {
	t.Run("no filter returns all contracts", func(t *testing.T) {
		svc := &mockContractServiceForSearch{
			contracts: []domain.Contract{
				{ID: 1, TotalAmount: 1000},
				{ID: 2, TotalAmount: 2000},
			},
		}
		handler := delivery.NewContractHandler(svc)

		req := httptest.NewRequest(http.MethodGet, "/api/branches/1/dashboard/companies/5/contracts", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "1", "company_id": "5"})
		rec := httptest.NewRecorder()

		handler.GetContractsByCompany(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}
		if svc.lastClientID != 5 {
			t.Errorf("expected clientID 5, got %d", svc.lastClientID)
		}
		if svc.lastFilter.Amount != nil {
			t.Errorf("expected nil filter.Amount, got %v", *svc.lastFilter.Amount)
		}
	})

	t.Run("filter by amount query parameter", func(t *testing.T) {
		svc := &mockContractServiceForSearch{
			contracts: []domain.Contract{
				{ID: 1, TotalAmount: 50000},
			},
		}
		handler := delivery.NewContractHandler(svc)

		req := httptest.NewRequest(http.MethodGet, "/api/branches/1/dashboard/companies/5/contracts?amount=50000", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "1", "company_id": "5"})
		rec := httptest.NewRecorder()

		handler.GetContractsByCompany(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}
		if svc.lastFilter.Amount == nil {
			t.Fatal("expected non-nil filter.Amount")
		}
		if *svc.lastFilter.Amount != 50000 {
			t.Errorf("expected amount 50000, got %v", *svc.lastFilter.Amount)
		}
	})

	t.Run("filter by query with spaces and comma", func(t *testing.T) {
		svc := &mockContractServiceForSearch{
			contracts: []domain.Contract{
				{ID: 2, TotalAmount: 125000.50},
			},
		}
		handler := delivery.NewContractHandler(svc)

		req := httptest.NewRequest(http.MethodGet, "/api/branches/1/dashboard/companies/5/contracts?amount=125%20000,50", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "1", "company_id": "5"})
		rec := httptest.NewRecorder()

		handler.GetContractsByCompany(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}
		if svc.lastFilter.Amount == nil {
			t.Fatal("expected non-nil filter.Amount")
		}
		if *svc.lastFilter.Amount != 125000.50 {
			t.Errorf("expected amount 125000.50, got %v", *svc.lastFilter.Amount)
		}
	})

	t.Run("invalid amount format returns 400 Bad Request", func(t *testing.T) {
		svc := &mockContractServiceForSearch{}
		handler := delivery.NewContractHandler(svc)

		req := httptest.NewRequest(http.MethodGet, "/api/branches/1/dashboard/companies/5/contracts?amount=abc", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "1", "company_id": "5"})
		rec := httptest.NewRecorder()

		handler.GetContractsByCompany(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", rec.Code)
		}
	})
}
