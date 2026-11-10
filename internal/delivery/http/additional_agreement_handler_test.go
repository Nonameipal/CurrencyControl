package http_test

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	delivery "CurrencyControl/internal/delivery/http"
	"CurrencyControl/internal/domain"

	"github.com/gorilla/mux"
)

type mockAddlService struct {
	createFunc      func(ctx context.Context, ag domain.AdditionalAgreement) (domain.AdditionalAgreement, error)
	getByContractID func(ctx context.Context, contractID int64) ([]domain.AdditionalAgreement, error)
	getByIDFunc     func(ctx context.Context, id int64) (domain.AdditionalAgreement, error)
	updateFunc      func(ctx context.Context, id int64, ag domain.AdditionalAgreement) (domain.AdditionalAgreement, error)
	deleteFunc      func(ctx context.Context, id int64) error
	restoreFunc     func(ctx context.Context, id int64) error
}

func (m *mockAddlService) Create(ctx context.Context, ag domain.AdditionalAgreement) (domain.AdditionalAgreement, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, ag)
	}
	ag.ID = 201
	return ag, nil
}

func (m *mockAddlService) GetByContractID(ctx context.Context, contractID int64) ([]domain.AdditionalAgreement, error) {
	if m.getByContractID != nil {
		return m.getByContractID(ctx, contractID)
	}
	return []domain.AdditionalAgreement{{ID: 201, ContractID: contractID}}, nil
}

func (m *mockAddlService) GetByID(ctx context.Context, id int64) (domain.AdditionalAgreement, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	num := "AA-100"
	return domain.AdditionalAgreement{ID: id, AgreementNumber: &num}, nil
}

func (m *mockAddlService) Update(ctx context.Context, id int64, ag domain.AdditionalAgreement) (domain.AdditionalAgreement, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, id, ag)
	}
	return ag, nil
}

func (m *mockAddlService) SoftDelete(ctx context.Context, id int64) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func (m *mockAddlService) RestoreAdditionalAgreement(ctx context.Context, id int64) error {
	if m.restoreFunc != nil {
		return m.restoreFunc(ctx, id)
	}
	return nil
}

func TestAdditionalAgreementHandler_CardCRUD(t *testing.T) {
	addlSvc := &mockAddlService{}
	h := delivery.NewInvoiceHandler(nil, nil, addlSvc, nil)

	t.Run("success create AA with exact 4 card fields + PDF", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		_ = writer.WriteField("agreement_number", "AA-2026-001")
		_ = writer.WriteField("agreement_date", "10.11.2026")
		_ = writer.WriteField("currency", "EUR")
		_ = writer.WriteField("amount", "7500.50")

		part, _ := writer.CreateFormFile("document", "agreement.pdf")
		_, _ = part.Write([]byte("%PDF-1.4 test additional agreement content"))
		_ = writer.Close()

		var capturedAA domain.AdditionalAgreement
		addlSvc.createFunc = func(ctx context.Context, ag domain.AdditionalAgreement) (domain.AdditionalAgreement, error) {
			capturedAA = ag
			ag.ID = 201
			return ag, nil
		}

		req := httptest.NewRequest(http.MethodPost, "/api/branches/1/dashboard/companies/2/contracts/3/additional-agreements", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req = mux.SetURLVars(req, map[string]string{
			"id":          "1",
			"company_id":  "2",
			"contract_id": "3",
		})
		req = req.WithContext(context.WithValue(req.Context(), domain.CtxKeyLogin, "operator_user"))

		w := httptest.NewRecorder()
		h.CreateAdditionalAgreement(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
		}

		if capturedAA.AgreementNumber == nil || *capturedAA.AgreementNumber != "AA-2026-001" {
			t.Errorf("expected AgreementNumber 'AA-2026-001', got %v", capturedAA.AgreementNumber)
		}
		if capturedAA.Currency == nil || *capturedAA.Currency != "EUR" {
			t.Errorf("expected Currency 'EUR', got %v", capturedAA.Currency)
		}
		if capturedAA.Amount == nil || *capturedAA.Amount != 7500.50 {
			t.Errorf("expected Amount 7500.50, got %v", capturedAA.Amount)
		}
		if capturedAA.CreatedBy != "operator_user" {
			t.Errorf("expected CreatedBy 'operator_user', got '%s'", capturedAA.CreatedBy)
		}
		if capturedAA.DocumentPath == "" {
			t.Errorf("expected DocumentPath to be non-empty after PDF upload")
		}
	})

	t.Run("missing required agreement_number returns 400", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		_ = writer.WriteField("agreement_date", "10.11.2026")
		_ = writer.WriteField("currency", "USD")
		_ = writer.WriteField("amount", "5000")
		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/branches/1/dashboard/companies/2/contracts/3/additional-agreements", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req = mux.SetURLVars(req, map[string]string{
			"id":          "1",
			"company_id":  "2",
			"contract_id": "3",
		})

		w := httptest.NewRecorder()
		h.CreateAdditionalAgreement(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "agreement_number") {
			t.Errorf("expected error mentioning agreement_number, got %s", w.Body.String())
		}
	})

	t.Run("missing required currency returns 400", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		_ = writer.WriteField("agreement_number", "AA-202")
		_ = writer.WriteField("agreement_date", "10.11.2026")
		_ = writer.WriteField("amount", "5000")
		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/branches/1/dashboard/companies/2/contracts/3/additional-agreements", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req = mux.SetURLVars(req, map[string]string{
			"id":          "1",
			"company_id":  "2",
			"contract_id": "3",
		})

		w := httptest.NewRecorder()
		h.CreateAdditionalAgreement(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "currency") {
			t.Errorf("expected error mentioning currency, got %s", w.Body.String())
		}
	})

	t.Run("GetAdditionalAgreementByID success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/branches/1/dashboard/companies/2/contracts/3/additional-agreements/201", nil)
		req = mux.SetURLVars(req, map[string]string{
			"id":           "1",
			"company_id":   "2",
			"contract_id":  "3",
			"agreement_id": "201",
		})
		w := httptest.NewRecorder()
		h.GetAdditionalAgreementByID(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("UpdateAdditionalAgreement success", func(t *testing.T) {
		amt := 6000.0
		curr := "USD"
		num := "AA-100"
		d := time.Now()
		addlSvc.getByIDFunc = func(ctx context.Context, id int64) (domain.AdditionalAgreement, error) {
			return domain.AdditionalAgreement{
				ID:              id,
				ContractID:      3,
				AgreementNumber: &num,
				AgreementDate:   &d,
				Amount:          &amt,
				ForeignAmount:   &amt,
				Currency:        &curr,
				ForeignCurrency: &curr,
			}, nil
		}

		var updatedAA domain.AdditionalAgreement
		addlSvc.updateFunc = func(ctx context.Context, id int64, ag domain.AdditionalAgreement) (domain.AdditionalAgreement, error) {
			updatedAA = ag
			return ag, nil
		}

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		_ = writer.WriteField("agreement_number", "AA-100-MODIFIED")
		_ = writer.WriteField("amount", "9900")
		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPut, "/api/branches/1/dashboard/companies/2/contracts/3/additional-agreements/201", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req = mux.SetURLVars(req, map[string]string{
			"id":           "1",
			"company_id":   "2",
			"contract_id":  "3",
			"agreement_id": "201",
		})
		req = req.WithContext(context.WithValue(req.Context(), domain.CtxKeyLogin, "updater_user"))

		w := httptest.NewRecorder()
		h.UpdateAdditionalAgreement(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}
		if updatedAA.AgreementNumber == nil || *updatedAA.AgreementNumber != "AA-100-MODIFIED" {
			t.Errorf("expected updated AgreementNumber 'AA-100-MODIFIED', got %v", updatedAA.AgreementNumber)
		}
		if updatedAA.Amount == nil || *updatedAA.Amount != 9900 {
			t.Errorf("expected updated Amount 9900, got %v", updatedAA.Amount)
		}
		if updatedAA.UpdatedBy != "updater_user" {
			t.Errorf("expected UpdatedBy 'updater_user', got '%s'", updatedAA.UpdatedBy)
		}
	})
}
