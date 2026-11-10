package http_test

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	delivery "CurrencyControl/internal/delivery/http"
	"CurrencyControl/internal/domain"

	"github.com/gorilla/mux"
)

type mockGTDService struct {
	createFunc func(ctx context.Context, g domain.GTD) (domain.GTD, error)
	getByIDFunc func(ctx context.Context, id int64) (*domain.GTD, error)
	getListByInvoiceFunc func(ctx context.Context, invoiceID int64) ([]domain.GTD, error)
	getByContractFunc func(ctx context.Context, contractID int64) ([]domain.GTD, error)
	updateFunc func(ctx context.Context, id int64, g domain.GTD) (domain.GTD, error)
	deleteFunc func(ctx context.Context, id int64) error
}

func (m *mockGTDService) Create(ctx context.Context, g domain.GTD) (domain.GTD, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, g)
	}
	g.ID = 101
	return g, nil
}
func (m *mockGTDService) GetByID(ctx context.Context, id int64) (*domain.GTD, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return &domain.GTD{ID: id, GTDNumber: "GTD-001"}, nil
}
func (m *mockGTDService) GetByInvoiceID(ctx context.Context, invoiceID int64) (*domain.GTD, error) {
	return &domain.GTD{ID: 1, InvoiceID: invoiceID}, nil
}
func (m *mockGTDService) GetListByInvoiceID(ctx context.Context, invoiceID int64) ([]domain.GTD, error) {
	if m.getListByInvoiceFunc != nil {
		return m.getListByInvoiceFunc(ctx, invoiceID)
	}
	return []domain.GTD{{ID: 1, InvoiceID: invoiceID}}, nil
}
func (m *mockGTDService) GetByContractID(ctx context.Context, contractID int64) ([]domain.GTD, error) {
	if m.getByContractFunc != nil {
		return m.getByContractFunc(ctx, contractID)
	}
	return []domain.GTD{{ID: 1, ContractID: contractID}}, nil
}
func (m *mockGTDService) Update(ctx context.Context, id int64, g domain.GTD) (domain.GTD, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, id, g)
	}
	return g, nil
}
func (m *mockGTDService) SoftDelete(ctx context.Context, id int64) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

type mockActService struct {
	createFunc func(ctx context.Context, act domain.Act) (domain.Act, error)
	getByIDFunc func(ctx context.Context, id int64) (*domain.Act, error)
	getByInvoiceFunc func(ctx context.Context, invoiceID int64) ([]domain.Act, error)
	getByContractFunc func(ctx context.Context, contractID int64) ([]domain.Act, error)
	updateFunc func(ctx context.Context, id int64, act domain.Act) (domain.Act, error)
	deleteFunc func(ctx context.Context, id int64) error
}

func (m *mockActService) Create(ctx context.Context, act domain.Act) (domain.Act, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, act)
	}
	act.ID = 201
	return act, nil
}
func (m *mockActService) GetByID(ctx context.Context, id int64) (*domain.Act, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return &domain.Act{ID: id, ActNumber: "ACT-001"}, nil
}
func (m *mockActService) GetByInvoiceID(ctx context.Context, invoiceID int64) ([]domain.Act, error) {
	if m.getByInvoiceFunc != nil {
		return m.getByInvoiceFunc(ctx, invoiceID)
	}
	return []domain.Act{{ID: 1, InvoiceID: invoiceID}}, nil
}
func (m *mockActService) GetByContractID(ctx context.Context, contractID int64) ([]domain.Act, error) {
	if m.getByContractFunc != nil {
		return m.getByContractFunc(ctx, contractID)
	}
	return []domain.Act{{ID: 1, ContractID: contractID}}, nil
}
func (m *mockActService) Update(ctx context.Context, id int64, act domain.Act) (domain.Act, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, id, act)
	}
	return act, nil
}
func (m *mockActService) SoftDelete(ctx context.Context, id int64) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func TestGTDHandler_CreateAndCRUD(t *testing.T) {
	mockSvc := &mockGTDService{}
	handler := delivery.NewGTDHandler(mockSvc)

	sendRequest := func(fields map[string]string) *httptest.ResponseRecorder {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		for k, v := range fields {
			_ = writer.WriteField(k, v)
		}
		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/branches/1/dashboard/companies/1/contracts/1/invoices/1/gtd", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		ctx := context.WithValue(req.Context(), delivery.LoginContextKey, "operator_user")
		ctx = context.WithValue(ctx, delivery.RoleContextKey, domain.RoleOperator)
		ctx = context.WithValue(ctx, delivery.BranchIDContextKey, 1)
		req = req.WithContext(ctx)

		req = mux.SetURLVars(req, map[string]string{
			"id":          "1",
			"company_id":  "1",
			"contract_id": "1",
			"invoice_id":  "1",
		})

		rr := httptest.NewRecorder()
		handler.CreateGTD(rr, req)
		return rr
	}

	t.Run("success create GTD with exact 3 input fields", func(t *testing.T) {
		rr := sendRequest(map[string]string{
			"gtd_number": "GTD-998877",
			"gtd_date":   "2026-06-15",
			"gtd_amount": "5000.00",
		})
		if rr.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("missing required gtd_number", func(t *testing.T) {
		rr := sendRequest(map[string]string{
			"gtd_date":   "2026-06-15",
			"gtd_amount": "5000.00",
		})
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", rr.Code)
		}
	})

	t.Run("GetGTDByID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/branches/1/dashboard/companies/1/contracts/1/gtd/101", nil)
		req = mux.SetURLVars(req, map[string]string{
			"id":          "1",
			"company_id":  "1",
			"contract_id": "1",
			"gtd_id":      "101",
		})
		rr := httptest.NewRecorder()
		handler.GetGTDByID(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
	})
}

func TestActHandler_CreateAndCRUD(t *testing.T) {
	mockSvc := &mockActService{}
	handler := delivery.NewActHandler(mockSvc)

	sendRequest := func(fields map[string]string) *httptest.ResponseRecorder {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		for k, v := range fields {
			_ = writer.WriteField(k, v)
		}
		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/branches/1/dashboard/companies/1/contracts/1/invoices/1/acts", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		ctx := context.WithValue(req.Context(), delivery.LoginContextKey, "operator_user")
		ctx = context.WithValue(ctx, delivery.RoleContextKey, domain.RoleOperator)
		ctx = context.WithValue(ctx, delivery.BranchIDContextKey, 1)
		req = req.WithContext(ctx)

		req = mux.SetURLVars(req, map[string]string{
			"id":          "1",
			"company_id":  "1",
			"contract_id": "1",
			"invoice_id":  "1",
		})

		rr := httptest.NewRecorder()
		handler.CreateAct(rr, req)
		return rr
	}

	t.Run("success create Act with exact 3 input fields", func(t *testing.T) {
		rr := sendRequest(map[string]string{
			"act_number": "ACT-12345",
			"act_date":   "2026-06-20",
			"act_amount": "3500.00",
		})
		if rr.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("missing required act_number", func(t *testing.T) {
		rr := sendRequest(map[string]string{
			"act_date":   "2026-06-20",
			"act_amount": "3500.00",
		})
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", rr.Code)
		}
	})

	t.Run("GetActByID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/branches/1/dashboard/companies/1/contracts/1/acts/201", nil)
		req = mux.SetURLVars(req, map[string]string{
			"id":          "1",
			"company_id":  "1",
			"contract_id": "1",
			"act_id":      "201",
		})
		rr := httptest.NewRecorder()
		handler.GetActByID(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
	})
}
