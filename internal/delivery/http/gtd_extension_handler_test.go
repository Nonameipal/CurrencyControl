package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	delivery "CurrencyControl/internal/delivery/http"
	"CurrencyControl/internal/domain"

	"github.com/gorilla/mux"
)

type mockGTDExtensionService struct {
	createFunc func(ctx context.Context, login, role string, gtdID int64, requestedDeadline time.Time, documentPath string) (domain.GTDExtensionRequest, error)
	reviewFunc func(ctx context.Context, role, login string, id int64, decision string, approvedDeadline *time.Time, comment string) (*domain.GTDExtensionRequest, error)
}

func (m *mockGTDExtensionService) CreateRequest(ctx context.Context, login, role string, gtdID int64, requestedDeadline time.Time, documentPath string) (domain.GTDExtensionRequest, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, login, role, gtdID, requestedDeadline, documentPath)
	}
	return domain.GTDExtensionRequest{
		ID:                1,
		GTDID:             gtdID,
		RequestedDeadline: requestedDeadline,
		DocumentPath:      documentPath,
		Status:            domain.ExtensionStatusPending,
		CreatedBy:         login,
	}, nil
}

func (m *mockGTDExtensionService) GetByID(ctx context.Context, id int64) (*domain.GTDExtensionRequest, error) {
	return &domain.GTDExtensionRequest{ID: id}, nil
}

func (m *mockGTDExtensionService) GetByGTDID(ctx context.Context, gtdID int64) ([]domain.GTDExtensionRequest, error) {
	return []domain.GTDExtensionRequest{{ID: 1, GTDID: gtdID}}, nil
}

func (m *mockGTDExtensionService) GetPendingRequests(ctx context.Context, branchID *int, page, pageSize int) ([]domain.GTDExtensionRequest, int, error) {
	return []domain.GTDExtensionRequest{}, 0, nil
}

func (m *mockGTDExtensionService) ReviewRequest(ctx context.Context, role, login string, id int64, decision string, approvedDeadline *time.Time, comment string) (*domain.GTDExtensionRequest, error) {
	if m.reviewFunc != nil {
		return m.reviewFunc(ctx, role, login, id, decision, approvedDeadline, comment)
	}
	return &domain.GTDExtensionRequest{
		ID:         id,
		Status:     decision,
		ReviewedBy: &login,
		Comment:    comment,
	}, nil
}

func TestGTDExtensionHandler_RequestExtension(t *testing.T) {
	mockSvc := &mockGTDExtensionService{}
	handler := delivery.NewGTDExtensionHandler(mockSvc)

	sendRequest := func(deadline string, includeFile bool) *httptest.ResponseRecorder {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		if deadline != "" {
			_ = writer.WriteField("requested_deadline", deadline)
		}
		if includeFile {
			part, _ := writer.CreateFormFile("document", "letter.pdf")
			_, _ = part.Write([]byte("%PDF-1.4 mock content"))
		}
		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/branches/1/dashboard/companies/1/contracts/1/invoices/1/gtd/10/extend", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		ctx := context.WithValue(req.Context(), delivery.LoginContextKey, "operator_user")
		ctx = context.WithValue(ctx, delivery.RoleContextKey, domain.RoleOperator)
		req = req.WithContext(ctx)

		req = mux.SetURLVars(req, map[string]string{
			"id":          "1",
			"company_id":  "1",
			"contract_id": "1",
			"invoice_id":  "1",
			"gtd_id":      "10",
		})

		rr := httptest.NewRecorder()
		handler.RequestExtension(rr, req)
		return rr
	}

	t.Run("success with 2 fields: calendar date and document", func(t *testing.T) {
		rr := sendRequest("2026-11-20", true)
		if rr.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("missing requested_deadline", func(t *testing.T) {
		rr := sendRequest("", true)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", rr.Code)
		}
	})

	t.Run("missing document file", func(t *testing.T) {
		rr := sendRequest("2026-11-20", false)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", rr.Code)
		}
	})
}

func TestGTDExtensionHandler_ReviewExtension(t *testing.T) {
	mockSvc := &mockGTDExtensionService{}
	handler := delivery.NewGTDExtensionHandler(mockSvc)

	sendReview := func(role string, decision string, comment string) *httptest.ResponseRecorder {
		payload := map[string]string{
			"decision": decision,
			"comment":  comment,
		}
		bodyBytes, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/api/approvals/gtd-extensions/1/review", bytes.NewReader(bodyBytes))

		ctx := context.WithValue(req.Context(), delivery.LoginContextKey, "cc_user")
		ctx = context.WithValue(ctx, delivery.RoleContextKey, role)
		req = req.WithContext(ctx)

		req = mux.SetURLVars(req, map[string]string{
			"request_id": "1",
		})

		rr := httptest.NewRecorder()
		handler.ReviewExtension(rr, req)
		return rr
	}

	t.Run("currency control officer can approve", func(t *testing.T) {
		rr := sendReview(domain.RoleCurrencyControl, "approve", "Согласовано на основании письма таможни")
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rr.Code, rr.Body.String())
		}
	})
}
