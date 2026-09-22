package http

// import (
// 	"bytes"
// 	"context"
// 	"encoding/json"
// 	"io"
// 	"mime/multipart"
// 	"net/http"
// 	"net/http/httptest"
// 	"testing"
// 	"time"

// 	"CurrencyControl/internal/delivery/dto"
// 	"CurrencyControl/internal/domain"

// 	"github.com/gorilla/mux"
// )

// func TestValidateContractDates(t *testing.T) {
// 	contractDate := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
// 	deliveryDate := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
// 	endDate := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)

// 	tests := []struct {
// 		name         string
// 		cDate        time.Time
// 		dDate        time.Time
// 		eDate        time.Time
// 		wantErr      bool
// 		errSubstring string
// 	}{
// 		{
// 			name:    "valid dates: contract <= delivery <= end",
// 			cDate:   contractDate,
// 			dDate:   deliveryDate,
// 			eDate:   endDate,
// 			wantErr: false,
// 		},
// 		{
// 			name:    "valid dates: all same day",
// 			cDate:   contractDate,
// 			dDate:   contractDate,
// 			eDate:   contractDate,
// 			wantErr: false,
// 		},
// 		{
// 			name:         "invalid: delivery before contract",
// 			cDate:        contractDate,
// 			dDate:        time.Date(2026, 5, 9, 0, 0, 0, 0, time.UTC),
// 			eDate:        endDate,
// 			wantErr:      true,
// 			errSubstring: "срок поставки товара не может быть раньше даты контракта",
// 		},
// 		{
// 			name:         "invalid: end date before contract",
// 			cDate:        contractDate,
// 			dDate:        deliveryDate,
// 			eDate:        time.Date(2026, 5, 5, 0, 0, 0, 0, time.UTC),
// 			wantErr:      true,
// 			errSubstring: "дата окончания контракта не может быть раньше даты контракта",
// 		},
// 		{
// 			name:         "invalid: delivery after end date",
// 			cDate:        contractDate,
// 			dDate:        time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
// 			eDate:        endDate,
// 			wantErr:      true,
// 			errSubstring: "срок поставки товара не может быть позже даты окончания контракта",
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			err := validateContractDates(tt.cDate, tt.dDate, tt.eDate)
// 			if (err != nil) != tt.wantErr {
// 				t.Fatalf("validateContractDates() error = %v, wantErr %v", err, tt.wantErr)
// 			}
// 			if tt.wantErr && tt.errSubstring != "" {
// 				if err == nil || !containsSubstring(err.Error(), tt.errSubstring) {
// 					t.Errorf("validateContractDates() error = %v, expected substring %q", err, tt.errSubstring)
// 				}
// 			}
// 		})
// 	}
// }

// type mockContractServiceForTest struct {
// 	createdContract  domain.Contract
// 	createErr        error
// 	existingContract domain.Contract
// 	getErr           error
// 	updatedContract  domain.Contract
// 	updateErr        error
// }

// func (m *mockContractServiceForTest) Create(ctx context.Context, login string, input domain.Contract) (domain.Contract, error) {
// 	if m.createErr != nil {
// 		return domain.Contract{}, m.createErr
// 	}
// 	m.createdContract = input
// 	input.ID = 101
// 	return input, nil
// }

// func (m *mockContractServiceForTest) GetByID(ctx context.Context, login string, id int64) (domain.Contract, error) {
// 	if m.getErr != nil {
// 		return domain.Contract{}, m.getErr
// 	}
// 	return m.existingContract, nil
// }

// func (m *mockContractServiceForTest) GetByClientID(ctx context.Context, login string, clientID int64) ([]domain.Contract, error) {
// 	return nil, nil
// }
// func (m *mockContractServiceForTest) GetArchived(ctx context.Context, branchID int, page, pageSize int) ([]domain.Contract, int, error) {
// 	return nil, 0, nil
// }
// func (m *mockContractServiceForTest) GetArchivedByClientID(ctx context.Context, clientID int64) ([]domain.Contract, error) {
// 	return nil, nil
// }
// func (m *mockContractServiceForTest) RestoreContract(ctx context.Context, id int64) error {
// 	return nil
// }
// func (m *mockContractServiceForTest) SearchDashboard(ctx context.Context, login string, req dto.DashboardSearchRequest) ([]dto.DashboardSearchResult, error) {
// 	return nil, nil
// }
// func (m *mockContractServiceForTest) CheckCountry(ctx context.Context, name string) (bool, error) {
// 	return true, nil
// }
// func (m *mockContractServiceForTest) CheckCurrency(ctx context.Context, code string) (bool, error) {
// 	return true, nil
// }
// func (m *mockContractServiceForTest) GetExpiringContracts(ctx context.Context, branchID int) ([]dto.NotificationResponse, error) {
// 	return nil, nil
// }
// func (m *mockContractServiceForTest) Update(ctx context.Context, id int64, c domain.Contract) (domain.Contract, error) {
// 	if m.updateErr != nil {
// 		return domain.Contract{}, m.updateErr
// 	}
// 	m.updatedContract = c
// 	return c, nil
// }
// func (m *mockContractServiceForTest) SoftDelete(ctx context.Context, id int64) error {
// 	return nil
// }

// func createMultipartContractRequest(t *testing.T, formFields map[string]string) *http.Request {
// 	body := &bytes.Buffer{}
// 	writer := multipart.NewWriter(body)

// 	for k, v := range formFields {
// 		_ = writer.WriteField(k, v)
// 	}

// 	part, err := writer.CreateFormFile("document", "test.pdf")
// 	if err != nil {
// 		t.Fatalf("failed to create form file: %v", err)
// 	}
// 	_, _ = io.WriteString(part, "fake pdf content")
// 	_ = writer.Close()

// 	req := httptest.NewRequest(http.MethodPost, "/api/branches/1/dashboard/companies/1/contracts", body)
// 	req.Header.Set("Content-Type", writer.FormDataContentType())
// 	req = mux.SetURLVars(req, map[string]string{
// 		"id":         "1",
// 		"company_id": "1",
// 	})
// 	return req
// }

// func TestContractHandler_Create_DateValidation(t *testing.T) {
// 	t.Run("delivery date before contract date should return 400", func(t *testing.T) {
// 		svc := &mockContractServiceForTest{}
// 		handler := NewContractHandler(svc)

// 		fields := map[string]string{
// 			"contract_number":   "CN-001",
// 			"contract_date":     "2026-05-10",
// 			"delivery_date":     "2026-05-01", // earlier than contract date!
// 			"contract_end_date": "2026-07-01",
// 			"return_days":       "30",
// 			"total_amount":      "1000",
// 			"contract_currency": "USD",
// 			"subject":           "Test contract",
// 			"receiver_name":     "Receiver Corp",
// 			"receiver_bank":     "Bank Corp",
// 			"receiver_country":  "Таджикистан",
// 		}

// 		req := createMultipartContractRequest(t, fields)
// 		rec := httptest.NewRecorder()

// 		handler.Create(rec, req)

// 		if rec.Code != http.StatusBadRequest {
// 			t.Fatalf("expected 400 Bad Request, got %d: %s", rec.Code, rec.Body.String())
// 		}

// 		var resp map[string]string
// 		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
// 		if !containsSubstring(resp["error"], "срок поставки товара не может быть раньше даты контракта") {
// 			t.Errorf("unexpected error message: %s", resp["error"])
// 		}
// 	})

// 	t.Run("contract end date before delivery date should return 400", func(t *testing.T) {
// 		svc := &mockContractServiceForTest{}
// 		handler := NewContractHandler(svc)

// 		fields := map[string]string{
// 			"contract_number":   "CN-002",
// 			"contract_date":     "2026-05-10",
// 			"delivery_date":     "2026-06-15",
// 			"contract_end_date": "2026-06-01", // earlier than delivery date!
// 			"return_days":       "30",
// 			"total_amount":      "1000",
// 			"contract_currency": "USD",
// 			"subject":           "Test contract",
// 			"receiver_name":     "Receiver Corp",
// 			"receiver_bank":     "Bank Corp",
// 			"receiver_country":  "Таджикистан",
// 		}

// 		req := createMultipartContractRequest(t, fields)
// 		rec := httptest.NewRecorder()

// 		handler.Create(rec, req)

// 		if rec.Code != http.StatusBadRequest {
// 			t.Fatalf("expected 400 Bad Request, got %d: %s", rec.Code, rec.Body.String())
// 		}

// 		var resp map[string]string
// 		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
// 		if !containsSubstring(resp["error"], "срок поставки товара не может быть позже даты окончания контракта") {
// 			t.Errorf("unexpected error message: %s", resp["error"])
// 		}
// 	})

// 	t.Run("valid dates and return_days creates contract successfully", func(t *testing.T) {
// 		svc := &mockContractServiceForTest{}
// 		handler := NewContractHandler(svc)

// 		fields := map[string]string{
// 			"contract_number":   "CN-003",
// 			"contract_date":     "2026-05-10",
// 			"delivery_date":     "2026-06-15",
// 			"contract_end_date": "2026-07-20",
// 			"return_days":       "45",
// 			"total_amount":      "5000",
// 			"contract_currency": "USD",
// 			"subject":           "Test valid contract",
// 			"receiver_name":     "Receiver Corp",
// 			"receiver_bank":     "Bank Corp",
// 			"receiver_country":  "Таджикистан",
// 		}

// 		req := createMultipartContractRequest(t, fields)
// 		rec := httptest.NewRecorder()

// 		handler.Create(rec, req)

// 		if rec.Code != http.StatusCreated {
// 			t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
// 		}

// 		if svc.createdContract.ReturnDays == nil || *svc.createdContract.ReturnDays != 45 {
// 			t.Errorf("expected ReturnDays to be 45, got %v", svc.createdContract.ReturnDays)
// 		}
// 	})
// }

// func TestContractHandler_Update_DateValidation(t *testing.T) {
// 	existingCDate := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
// 	existingDDate := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
// 	existingEDate := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
// 	rDays := 30

// 	baseContract := domain.Contract{
// 		ID:              1,
// 		ContractNumber:  "CN-UPD",
// 		ContractDate:    existingCDate,
// 		DeliveryDate:    existingDDate,
// 		ContractEndDate: &existingEDate,
// 		ReturnDays:      &rDays,
// 	}

// 	t.Run("update delivery date earlier than contract date returns 400", func(t *testing.T) {
// 		svc := &mockContractServiceForTest{existingContract: baseContract}
// 		handler := NewContractHandler(svc)

// 		fields := map[string]string{
// 			"delivery_date": "2026-05-01", // earlier than 2026-05-10
// 		}
// 		req := createMultipartContractRequest(t, fields)
// 		req = mux.SetURLVars(req, map[string]string{
// 			"id":          "1",
// 			"company_id":  "1",
// 			"contract_id": "1",
// 		})
// 		rec := httptest.NewRecorder()

// 		handler.Update(rec, req)

// 		if rec.Code != http.StatusBadRequest {
// 			t.Fatalf("expected 400 Bad Request, got %d: %s", rec.Code, rec.Body.String())
// 		}
// 		var resp map[string]string
// 		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
// 		if !containsSubstring(resp["error"], "срок поставки товара не может быть раньше даты контракта") {
// 			t.Errorf("unexpected error message: %s", resp["error"])
// 		}
// 	})

// 	t.Run("update return_days sets ReturnDays successfully", func(t *testing.T) {
// 		svc := &mockContractServiceForTest{existingContract: baseContract}
// 		handler := NewContractHandler(svc)

// 		fields := map[string]string{
// 			"return_days": "60",
// 		}
// 		req := createMultipartContractRequest(t, fields)
// 		req = mux.SetURLVars(req, map[string]string{
// 			"id":          "1",
// 			"company_id":  "1",
// 			"contract_id": "1",
// 		})
// 		rec := httptest.NewRecorder()

// 		handler.Update(rec, req)

// 		if rec.Code != http.StatusOK {
// 			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
// 		}
// 		if svc.updatedContract.ReturnDays == nil || *svc.updatedContract.ReturnDays != 60 {
// 			t.Errorf("expected updated ReturnDays 60, got %v", svc.updatedContract.ReturnDays)
// 		}
// 	})
// }

// func TestContractHandler_Create_ReturnDaysMandatory(t *testing.T) {
// 	t.Run("missing return_days returns 400", func(t *testing.T) {
// 		svc := &mockContractServiceForTest{}
// 		handler := NewContractHandler(svc)

// 		fields := map[string]string{
// 			"contract_number":   "CN-100",
// 			"contract_date":     "2026-05-10",
// 			"delivery_date":     "2026-06-15",
// 			"contract_end_date": "2026-07-20",
// 			"total_amount":      "1000",
// 			"contract_currency": "USD",
// 			"subject":           "Test contract",
// 			"receiver_name":     "Receiver Corp",
// 			"receiver_bank":     "Bank Corp",
// 			"receiver_country":  "Таджикистан",
// 		}

// 		req := createMultipartContractRequest(t, fields)
// 		rec := httptest.NewRecorder()

// 		handler.Create(rec, req)

// 		if rec.Code != http.StatusBadRequest {
// 			t.Fatalf("expected 400 Bad Request, got %d: %s", rec.Code, rec.Body.String())
// 		}
// 		var resp map[string]string
// 		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
// 		if !containsSubstring(resp["error"], "Поле return_days обязательно") {
// 			t.Errorf("expected 'Поле return_days обязательно', got: %s", resp["error"])
// 		}
// 	})

// 	t.Run("zero return_days returns 400", func(t *testing.T) {
// 		svc := &mockContractServiceForTest{}
// 		handler := NewContractHandler(svc)

// 		fields := map[string]string{
// 			"contract_number":   "CN-101",
// 			"contract_date":     "2026-05-10",
// 			"delivery_date":     "2026-06-15",
// 			"contract_end_date": "2026-07-20",
// 			"return_days":       "0",
// 			"total_amount":      "1000",
// 			"contract_currency": "USD",
// 			"subject":           "Test contract",
// 			"receiver_name":     "Receiver Corp",
// 			"receiver_bank":     "Bank Corp",
// 			"receiver_country":  "Таджикистан",
// 		}

// 		req := createMultipartContractRequest(t, fields)
// 		rec := httptest.NewRecorder()

// 		handler.Create(rec, req)

// 		if rec.Code != http.StatusBadRequest {
// 			t.Fatalf("expected 400 Bad Request, got %d: %s", rec.Code, rec.Body.String())
// 		}
// 	})
// }

// type mockAddlServiceForTest struct {
// 	created   domain.AdditionalAgreement
// 	createErr error
// 	existing  domain.AdditionalAgreement
// 	getErr    error
// 	updated   domain.AdditionalAgreement
// 	updateErr error
// }

// func (m *mockAddlServiceForTest) Create(ctx context.Context, ag domain.AdditionalAgreement) (domain.AdditionalAgreement, error) {
// 	if m.createErr != nil {
// 		return domain.AdditionalAgreement{}, m.createErr
// 	}
// 	m.created = ag
// 	ag.ID = 201
// 	return ag, nil
// }
// func (m *mockAddlServiceForTest) GetByID(ctx context.Context, id int64) (domain.AdditionalAgreement, error) {
// 	if m.getErr != nil {
// 		return domain.AdditionalAgreement{}, m.getErr
// 	}
// 	return m.existing, nil
// }
// func (m *mockAddlServiceForTest) Update(ctx context.Context, id int64, ag domain.AdditionalAgreement) (domain.AdditionalAgreement, error) {
// 	if m.updateErr != nil {
// 		return domain.AdditionalAgreement{}, m.updateErr
// 	}
// 	m.updated = ag
// 	return ag, nil
// }
// func (m *mockAddlServiceForTest) GetByContractID(ctx context.Context, contractID int64) ([]domain.AdditionalAgreement, error) {
// 	return nil, nil
// }
// func (m *mockAddlServiceForTest) SoftDelete(ctx context.Context, id int64) error {
// 	return nil
// }
// func (m *mockAddlServiceForTest) RestoreAdditionalAgreement(ctx context.Context, id int64) error {
// 	return nil
// }

// // func createMultipartAARequest(t *testing.T, formFields map[string]string) *http.Request {
// // 	body := &bytes.Buffer{}
// // 	writer := multipart.NewWriter(body)

// // 	for k, v := range formFields {
// // 		_ = writer.WriteField(k, v)
// // 	}
// // 	_ = writer.Close()

// // 	req := httptest.NewRequest(http.MethodPost, "/api/branches/1/dashboard/companies/1/contracts/1/additional-agreements", body)
// // 	req.Header.Set("Content-Type", writer.FormDataContentType())
// // 	req = mux.SetURLVars(req, map[string]string{
// // 		"id":          "1",
// // 		"company_id":  "1",
// // 		"contract_id": "1",
// // 	})
// // 	return req
// // }

// func TestAdditionalAgreement_DateValidation(t *testing.T) {
// 	t.Run("delivery date before agreement date returns 400", func(t *testing.T) {
// 		addlMock := &mockAddlServiceForTest{}
// 		handler := NewInvoiceHandler(nil, nil, addlMock, nil)

// 		fields := map[string]string{
// 			"agreement_date":     "2026-05-15",
// 			"delivery_date":      "2026-05-10", // before agreement date!
// 			"agreement_end_date": "2026-06-30",
// 		}
// 		req := createMultipartAARequest(t, fields)
// 		rec := httptest.NewRecorder()

// 		handler.CreateAdditionalAgreement(rec, req)

// 		if rec.Code != http.StatusBadRequest {
// 			t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
// 		}
// 		var resp map[string]string
// 		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
// 		if !containsSubstring(resp["error"], "срок поставки товара не может быть раньше даты доп. соглашения") {
// 			t.Errorf("unexpected error: %s", resp["error"])
// 		}
// 	})

// 	t.Run("delivery date after agreement end date returns 400", func(t *testing.T) {
// 		addlMock := &mockAddlServiceForTest{}
// 		handler := NewInvoiceHandler(nil, nil, addlMock, nil)

// 		fields := map[string]string{
// 			"agreement_date":     "2026-05-15",
// 			"delivery_date":      "2026-07-10",
// 			"agreement_end_date": "2026-06-30", // before delivery date!
// 		}
// 		req := createMultipartAARequest(t, fields)
// 		rec := httptest.NewRecorder()

// 		handler.CreateAdditionalAgreement(rec, req)

// 		if rec.Code != http.StatusBadRequest {
// 			t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
// 		}
// 		var resp map[string]string
// 		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
// 		if !containsSubstring(resp["error"], "срок поставки товара не может быть позже даты окончания доп. соглашения") {
// 			t.Errorf("unexpected error: %s", resp["error"])
// 		}
// 	})

// 	t.Run("valid dates without return_days succeeds (optional for AA)", func(t *testing.T) {
// 		addlMock := &mockAddlServiceForTest{}
// 		handler := NewInvoiceHandler(nil, nil, addlMock, nil)

// 		fields := map[string]string{
// 			"agreement_date":     "2026-05-15",
// 			"delivery_date":      "2026-06-15",
// 			"agreement_end_date": "2026-07-15",
// 		}
// 		req := createMultipartAARequest(t, fields)
// 		rec := httptest.NewRecorder()

// 		handler.CreateAdditionalAgreement(rec, req)

// 		if rec.Code != http.StatusCreated {
// 			t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
// 		}
// 		if addlMock.created.ReturnDays != nil {
// 			t.Errorf("expected nil ReturnDays, got %v", addlMock.created.ReturnDays)
// 		}
// 	})

// 	t.Run("valid dates with return_days sets ReturnDays", func(t *testing.T) {
// 		addlMock := &mockAddlServiceForTest{}
// 		handler := NewInvoiceHandler(nil, nil, addlMock, nil)

// 		fields := map[string]string{
// 			"agreement_date":     "2026-05-15",
// 			"delivery_date":      "2026-06-15",
// 			"agreement_end_date": "2026-07-15",
// 			"return_days":        "20",
// 		}
// 		req := createMultipartAARequest(t, fields)
// 		rec := httptest.NewRecorder()

// 		handler.CreateAdditionalAgreement(rec, req)

// 		if rec.Code != http.StatusCreated {
// 			t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
// 		}
// 		if addlMock.created.ReturnDays == nil || *addlMock.created.ReturnDays != 20 {
// 			t.Errorf("expected ReturnDays 20, got %v", addlMock.created.ReturnDays)
// 		}
// 	})
// }

// func containsSubstring(s, sub string) bool {
// 	return bytes.Contains([]byte(s), []byte(sub))
// }
