package http_test

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	delivery "CurrencyControl/internal/delivery/http"
	"CurrencyControl/internal/domain"

	"github.com/gorilla/mux"
)

type mockPaymentOrderService struct {
	createFunc       func(ctx context.Context, po domain.PaymentOrder) (domain.PaymentOrder, error)
	getByIDFunc      func(ctx context.Context, id int64) (*domain.PaymentOrder, error)
	getByInvoiceFunc func(ctx context.Context, invoiceID int64) ([]domain.PaymentOrder, error)
	updateFunc       func(ctx context.Context, id int64, po domain.PaymentOrder) (*domain.PaymentOrder, error)
	deleteFunc       func(ctx context.Context, id int64) error
}

func (m *mockPaymentOrderService) Create(ctx context.Context, po domain.PaymentOrder) (domain.PaymentOrder, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, po)
	}
	po.ID = 100
	return po, nil
}
func (m *mockPaymentOrderService) GetByID(ctx context.Context, id int64) (*domain.PaymentOrder, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return &domain.PaymentOrder{ID: id}, nil
}
func (m *mockPaymentOrderService) GetByInvoiceID(ctx context.Context, invoiceID int64) ([]domain.PaymentOrder, error) {
	if m.getByInvoiceFunc != nil {
		return m.getByInvoiceFunc(ctx, invoiceID)
	}
	return []domain.PaymentOrder{{ID: 1, InvoiceID: invoiceID}}, nil
}
func (m *mockPaymentOrderService) GetByContractID(ctx context.Context, contractID int64) ([]domain.PaymentOrder, error) {
	return []domain.PaymentOrder{}, nil
}
func (m *mockPaymentOrderService) GetByAdditionalAgreementID(ctx context.Context, agreementID int64) ([]domain.PaymentOrder, error) {
	return []domain.PaymentOrder{}, nil
}
func (m *mockPaymentOrderService) Update(ctx context.Context, id int64, po domain.PaymentOrder) (*domain.PaymentOrder, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, id, po)
	}
	return &po, nil
}
func (m *mockPaymentOrderService) SoftDelete(ctx context.Context, id int64) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func TestPaymentOrderHandler_Validation(t *testing.T) {
	mockSvc := &mockPaymentOrderService{}
	handler := delivery.NewPaymentOrderHandler(mockSvc)

	sendRequest := func(fields map[string]string) *httptest.ResponseRecorder {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		for k, v := range fields {
			_ = writer.WriteField(k, v)
		}
		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/branches/1/dashboard/companies/1/contracts/1/invoices/1/payment-orders", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		ctx := context.WithValue(req.Context(), delivery.LoginContextKey, "test_user")
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
		handler.CreatePaymentOrder(rr, req)
		return rr
	}

	validFields := func() map[string]string {
		return map[string]string{
			"operation_date":       "2026-05-10",
			"payment_order_number": "PO-999",
			"amount":               "1500.50",
			"currency":             "USD",
			"payer":                "ООО Рога и Копыта",
			"receiver_name":        "Supplier Corp",
			"receiver_bank":        "Bank of America",
			"payment_purpose":      "Оплата по инвойсу № 1",
			"receiver_country":     "США",
			"value_date":           "2026-05-12",
			"sender_name":          "ООО Отправитель",
			"sender_bank":          "Ориёнбанк",
			"sender_country":       "Таджикистан",
		}
	}

	t.Run("success valid fields", func(t *testing.T) {
		f := validFields()
		rr := sendRequest(f)
		if rr.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("missing sender fields", func(t *testing.T) {
		for _, field := range []string{"sender_name", "sender_bank", "sender_country"} {
			f := validFields()
			delete(f, field)
			rr := sendRequest(f)
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("expected 400 for missing %s, got %d", field, rr.Code)
			}
		}
	})

	t.Run("missing operation_date", func(t *testing.T) {
		f := validFields()
		delete(f, "operation_date")
		rr := sendRequest(f)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("missing payment_order_number", func(t *testing.T) {
		f := validFields()
		delete(f, "payment_order_number")
		rr := sendRequest(f)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("invalid or missing amount", func(t *testing.T) {
		f := validFields()
		f["amount"] = "0"
		rr := sendRequest(f)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rr.Code)
		}

		f["amount"] = "-100"
		rr = sendRequest(f)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("missing currency", func(t *testing.T) {
		f := validFields()
		delete(f, "currency")
		rr := sendRequest(f)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("service returns error on limit exceeded or currency mismatch", func(t *testing.T) {
		mockSvc.createFunc = func(ctx context.Context, po domain.PaymentOrder) (domain.PaymentOrder, error) {
			return domain.PaymentOrder{}, fmt.Errorf("сумма платежного поручения (1500.50 USD) превышает доступный остаток по оплате инвойса (500.00 USD)")
		}
		defer func() { mockSvc.createFunc = nil }()

		f := validFields()
		rr := sendRequest(f)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
		}
	})
}

func TestGetPaymentOrdersByContractAndAgreement(t *testing.T) {
	mockSvc := &mockPaymentOrderService{}
	handler := delivery.NewPaymentOrderHandler(mockSvc)

	t.Run("GetPaymentOrdersByContract success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/branches/1/dashboard/companies/2/contracts/3/payment-orders", nil)
		req = mux.SetURLVars(req, map[string]string{
			"id":          "1",
			"company_id":  "2",
			"contract_id": "3",
		})
		rr := httptest.NewRecorder()
		handler.GetPaymentOrdersByContract(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("GetPaymentOrdersByAdditionalAgreement success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/branches/1/dashboard/companies/2/contracts/3/additional-agreements/4/payment-orders", nil)
		req = mux.SetURLVars(req, map[string]string{
			"id":           "1",
			"company_id":   "2",
			"contract_id":  "3",
			"agreement_id": "4",
		})
		rr := httptest.NewRecorder()
		handler.GetPaymentOrdersByAdditionalAgreement(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("GetAdditionalAgreementInvoicePaymentOrders success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/branches/1/dashboard/companies/2/contracts/3/additional-agreements/4/invoices/5/payment-orders", nil)
		req = mux.SetURLVars(req, map[string]string{
			"id":           "1",
			"company_id":   "2",
			"contract_id":  "3",
			"agreement_id": "4",
			"invoice_id":   "5",
		})
		rr := httptest.NewRecorder()
		handler.GetAdditionalAgreementInvoicePaymentOrders(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
	})
}

