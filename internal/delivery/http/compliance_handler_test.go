package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"

	"CurrencyControl/internal/domain"
)

type mockPermissionService struct {
	lastGranted domain.CurrencyControlPermission
	lastRevoked string
	perms       []domain.CurrencyControlPermission
	err         error
}

func (m *mockPermissionService) GetCurrencyControlPermissions(ctx context.Context) ([]domain.CurrencyControlPermission, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.perms, nil
}
func (m *mockPermissionService) CanCreateFiles(ctx context.Context, role, login string) bool { return true }
func (m *mockPermissionService) CanEditFiles(ctx context.Context, role, login string) bool   { return true }
func (m *mockPermissionService) CanDeleteFiles(ctx context.Context, role, login string) bool { return true }
func (m *mockPermissionService) GrantCurrencyControlPermission(ctx context.Context, perm domain.CurrencyControlPermission) error {
	m.lastGranted = perm
	return m.err
}
func (m *mockPermissionService) RevokeCurrencyControlPermission(ctx context.Context, login string) error {
	m.lastRevoked = login
	return m.err
}

func TestComplianceHandler_GrantPermission(t *testing.T) {
	mockSvc := &mockPermissionService{}
	h := NewComplianceHandler(mockSvc)

	t.Run("success granting permission", func(t *testing.T) {
		body, _ := json.Marshal(GrantPermissionRequest{
			Login:     "ivanov",
			CanCreate: true,
			CanEdit:   false,
			CanDelete: true,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/compliance/permissions/currency-control", bytes.NewReader(body))
		ctx := context.WithValue(req.Context(), LoginContextKey, "compliance_admin")
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		h.GrantPermission(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}
		if mockSvc.lastGranted.Login != "ivanov" || !mockSvc.lastGranted.CanCreate || mockSvc.lastGranted.CanEdit || !mockSvc.lastGranted.CanDelete {
			t.Fatalf("unexpected granted permission: %+v", mockSvc.lastGranted)
		}
	})

	t.Run("empty login returns 400", func(t *testing.T) {
		body, _ := json.Marshal(GrantPermissionRequest{
			Login:     "",
			CanCreate: true,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/compliance/permissions/currency-control", bytes.NewReader(body))
		ctx := context.WithValue(req.Context(), LoginContextKey, "compliance_admin")
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		h.GrantPermission(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for empty login, got %d", w.Code)
		}
	})

	t.Run("invalid json returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/compliance/permissions/currency-control", bytes.NewReader([]byte("{invalid-json")))
		ctx := context.WithValue(req.Context(), LoginContextKey, "compliance_admin")
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		h.GrantPermission(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}
	})
}

func TestComplianceHandler_UpdatePermission(t *testing.T) {
	mockSvc := &mockPermissionService{}
	h := NewComplianceHandler(mockSvc)

	t.Run("success updating permission", func(t *testing.T) {
		body, _ := json.Marshal(UpdatePermissionRequest{
			CanCreate: false,
			CanEdit:   true,
			CanDelete: true,
		})
		req := httptest.NewRequest(http.MethodPut, "/api/compliance/permissions/currency-control/ivanov", bytes.NewReader(body))
		req = mux.SetURLVars(req, map[string]string{"login": "ivanov"})
		ctx := context.WithValue(req.Context(), LoginContextKey, "compliance_admin")
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		h.UpdatePermission(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}
		if mockSvc.lastGranted.Login != "ivanov" || mockSvc.lastGranted.CanCreate || !mockSvc.lastGranted.CanEdit || !mockSvc.lastGranted.CanDelete {
			t.Fatalf("unexpected granted permission: %+v", mockSvc.lastGranted)
		}
	})

	t.Run("missing or placeholder login returns 400", func(t *testing.T) {
		body, _ := json.Marshal(UpdatePermissionRequest{CanEdit: true})
		req := httptest.NewRequest(http.MethodPut, "/api/compliance/permissions/currency-control/{login}", bytes.NewReader(body))
		req = mux.SetURLVars(req, map[string]string{"login": "{login}"})
		ctx := context.WithValue(req.Context(), LoginContextKey, "compliance_admin")
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		h.UpdatePermission(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	t.Run("conflicting login in body returns 400", func(t *testing.T) {
		body, _ := json.Marshal(UpdatePermissionRequest{
			Login:   "petrov",
			CanEdit: true,
		})
		req := httptest.NewRequest(http.MethodPut, "/api/compliance/permissions/currency-control/ivanov", bytes.NewReader(body))
		req = mux.SetURLVars(req, map[string]string{"login": "ivanov"})
		ctx := context.WithValue(req.Context(), LoginContextKey, "compliance_admin")
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		h.UpdatePermission(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}
	})
}

func TestComplianceHandler_RevokePermission(t *testing.T) {
	mockSvc := &mockPermissionService{}
	h := NewComplianceHandler(mockSvc)

	t.Run("success revoking permission", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/compliance/permissions/currency-control/ivanov", nil)
		req = mux.SetURLVars(req, map[string]string{"login": "ivanov"})
		ctx := context.WithValue(req.Context(), LoginContextKey, "compliance_admin")
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		h.RevokePermission(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", w.Code)
		}
		if mockSvc.lastRevoked != "ivanov" {
			t.Fatalf("expected revoked login 'ivanov', got '%s'", mockSvc.lastRevoked)
		}
	})

	t.Run("placeholder login returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/compliance/permissions/currency-control/{login}", nil)
		req = mux.SetURLVars(req, map[string]string{"login": "{login}"})
		ctx := context.WithValue(req.Context(), LoginContextKey, "compliance_admin")
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		h.RevokePermission(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}
	})
}
