package http

// import (
// 	"context"
// 	"encoding/json"
// 	"net/http"
// 	"net/http/httptest"
// 	"testing"

// 	"CurrencyControl/internal/delivery/dto"
// 	"github.com/gorilla/mux"
// )

// type mockDashboardContractService struct {
// 	mockContractServiceForTest
// 	lastSearchReq dto.DashboardSearchRequest
// }

// func (m *mockDashboardContractService) SearchDashboard(ctx context.Context, login string, req dto.DashboardSearchRequest) ([]dto.DashboardSearchResult, error) {
// 	m.lastSearchReq = req
// 	return []dto.DashboardSearchResult{
// 		{
// 			ID:     1,
// 			Number: "№ 1",
// 			Name:   "Альфа",
// 			INN:    "123456789",
// 		},
// 	}, nil
// }

// func TestDashboardSearch_UnifiedInput(t *testing.T) {
// 	t.Run("search by name with 1 field and name button", func(t *testing.T) {
// 		svc := &mockDashboardContractService{}
// 		handler := NewContractHandler(svc)

// 		req := httptest.NewRequest(http.MethodGet, "/api/branches/1/dashboard?query=Альфа&search_type=name", nil)
// 		req = mux.SetURLVars(req, map[string]string{"id": "1"})
// 		rec := httptest.NewRecorder()

// 		handler.Dashboard(rec, req)

// 		if rec.Code != http.StatusOK {
// 			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
// 		}
// 		if svc.lastSearchReq.Query != "Альфа" {
// 			t.Errorf("expected Query 'Альфа', got '%s'", svc.lastSearchReq.Query)
// 		}
// 		if svc.lastSearchReq.SearchType != "name" {
// 			t.Errorf("expected SearchType 'name', got '%s'", svc.lastSearchReq.SearchType)
// 		}
// 		if svc.lastSearchReq.BranchID != 1 {
// 			t.Errorf("expected BranchID 1, got %d", svc.lastSearchReq.BranchID)
// 		}
// 	})

// 	t.Run("search by inn with 1 field and inn button", func(t *testing.T) {
// 		svc := &mockDashboardContractService{}
// 		handler := NewContractHandler(svc)

// 		req := httptest.NewRequest(http.MethodGet, "/api/branches/2/dashboard?query=010023456&search_type=inn", nil)
// 		req = mux.SetURLVars(req, map[string]string{"id": "2"})
// 		rec := httptest.NewRecorder()

// 		handler.Dashboard(rec, req)

// 		if rec.Code != http.StatusOK {
// 			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
// 		}
// 		if svc.lastSearchReq.Query != "010023456" {
// 			t.Errorf("expected Query '010023456', got '%s'", svc.lastSearchReq.Query)
// 		}
// 		if svc.lastSearchReq.SearchType != "inn" {
// 			t.Errorf("expected SearchType 'inn', got '%s'", svc.lastSearchReq.SearchType)
// 		}
// 	})

// 	t.Run("search by amount with 1 field and amount button", func(t *testing.T) {
// 		svc := &mockDashboardContractService{}
// 		handler := NewContractHandler(svc)

// 		req := httptest.NewRequest(http.MethodGet, "/api/branches/1/dashboard?query=50000&search_type=amount", nil)
// 		req = mux.SetURLVars(req, map[string]string{"id": "1"})
// 		rec := httptest.NewRecorder()

// 		handler.Dashboard(rec, req)

// 		if rec.Code != http.StatusOK {
// 			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
// 		}
// 		if svc.lastSearchReq.Query != "50000" {
// 			t.Errorf("expected Query '50000', got '%s'", svc.lastSearchReq.Query)
// 		}
// 		if svc.lastSearchReq.SearchType != "amount" {
// 			t.Errorf("expected SearchType 'amount', got '%s'", svc.lastSearchReq.SearchType)
// 		}
// 	})

// 	t.Run("search with aliases q and type", func(t *testing.T) {
// 		svc := &mockDashboardContractService{}
// 		handler := NewContractHandler(svc)

// 		req := httptest.NewRequest(http.MethodGet, "/api/branches/3/dashboard?q=Beta&type=company_name", nil)
// 		req = mux.SetURLVars(req, map[string]string{"id": "3"})
// 		rec := httptest.NewRecorder()

// 		handler.Dashboard(rec, req)

// 		if rec.Code != http.StatusOK {
// 			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
// 		}
// 		if svc.lastSearchReq.Query != "Beta" {
// 			t.Errorf("expected Query 'Beta', got '%s'", svc.lastSearchReq.Query)
// 		}
// 		if svc.lastSearchReq.SearchType != "company_name" {
// 			t.Errorf("expected SearchType 'company_name', got '%s'", svc.lastSearchReq.SearchType)
// 		}
// 	})

// 	t.Run("backwards compatibility with separate fields", func(t *testing.T) {
// 		svc := &mockDashboardContractService{}
// 		handler := NewContractHandler(svc)

// 		req := httptest.NewRequest(http.MethodGet, "/api/branches/1/dashboard?company_name=Gamma&inn=987654&amount=75000.50", nil)
// 		req = mux.SetURLVars(req, map[string]string{"id": "1"})
// 		rec := httptest.NewRecorder()

// 		handler.Dashboard(rec, req)

// 		if rec.Code != http.StatusOK {
// 			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
// 		}
// 		if svc.lastSearchReq.CompanyName != "Gamma" {
// 			t.Errorf("expected CompanyName 'Gamma', got '%s'", svc.lastSearchReq.CompanyName)
// 		}
// 		if svc.lastSearchReq.INN != "987654" {
// 			t.Errorf("expected INN '987654', got '%s'", svc.lastSearchReq.INN)
// 		}
// 		if svc.lastSearchReq.Amount != 75000.50 {
// 			t.Errorf("expected Amount 75000.50, got %v", svc.lastSearchReq.Amount)
// 		}

// 		var res []dto.DashboardSearchResult
// 		if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
// 			t.Fatalf("failed to decode response: %v", err)
// 		}
// 		if len(res) != 1 || res[0].Name != "Альфа" {
// 			t.Errorf("unexpected response body: %s", rec.Body.String())
// 		}
// 	})
// }
