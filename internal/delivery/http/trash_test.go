package http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	delivery "CurrencyControl/internal/delivery/http"
	"CurrencyControl/internal/domain"
)

type mockPermService struct {
	canEdit   bool
	canDelete bool
}

func (m *mockPermService) GetCurrencyControlPermissions(ctx context.Context) ([]domain.CurrencyControlPermission, error) {
	return nil, nil
}
func (m *mockPermService) CanEditFiles(ctx context.Context, role, login string) bool {
	if role == domain.RoleAdmin || role == domain.RoleCompliance {
		return true
	}
	if role == domain.RoleCurrencyControl || role == domain.RoleCurrencyController {
		return m.canEdit
	}
	return false
}
func (m *mockPermService) CanDeleteFiles(ctx context.Context, role, login string) bool {
	if role == domain.RoleAdmin || role == domain.RoleCompliance {
		return true
	}
	if role == domain.RoleCurrencyControl || role == domain.RoleCurrencyController {
		return m.canDelete
	}
	return false
}
func (m *mockPermService) GrantCurrencyControlPermission(ctx context.Context, perm domain.CurrencyControlPermission) error {
	return nil
}
func (m *mockPermService) RevokeCurrencyControlPermission(ctx context.Context, login string) error {
	return nil
}

func TestTrashAccessControl(t *testing.T) {
	middleware := delivery.RequireRoles(
		domain.RoleCompliance,
		domain.RoleCurrencyControl,
		domain.RoleCurrencyController,
		domain.RoleAdmin,
	)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	protectedHandler := middleware(testHandler)

	tests := []struct {
		name         string
		role         string
		expectedCode int
	}{
		{"Operator forbidden", domain.RoleOperator, http.StatusForbidden},
		{"BranchHead forbidden", domain.RoleBranchHead, http.StatusForbidden},
		{"InternalAudit forbidden", domain.RoleInternalAudit, http.StatusForbidden},
		{"Empty role unauthorized", "", http.StatusUnauthorized},
		{"Compliance allowed", domain.RoleCompliance, http.StatusOK},
		{"CurrencyControl allowed", domain.RoleCurrencyControl, http.StatusOK},
		{"CurrencyController allowed", domain.RoleCurrencyController, http.StatusOK},
		{"Admin allowed", domain.RoleAdmin, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/trash", nil)
			if tt.role != "" {
				ctx := context.WithValue(req.Context(), delivery.RoleContextKey, tt.role)
				ctx = context.WithValue(ctx, delivery.LoginContextKey, "testuser")
				req = req.WithContext(ctx)
			}
			rec := httptest.NewRecorder()

			protectedHandler.ServeHTTP(rec, req)

			if rec.Code != tt.expectedCode {
				t.Errorf("role %q: expected status %d, got %d", tt.role, tt.expectedCode, rec.Code)
			}
		})
	}
}

func TestDocumentEditAndDeleteAccess(t *testing.T) {
	mockPerm := &mockPermService{canEdit: true, canDelete: false}
	delivery.SetPermissionService(mockPerm)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	editHandler := delivery.RequireDocumentEditAccess()(testHandler)
	deleteHandler := delivery.RequireDocumentDeleteAccess()(testHandler)

	t.Run("Edit: Compliance always allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/api/contracts/1", nil)
		ctx := context.WithValue(req.Context(), delivery.RoleContextKey, domain.RoleCompliance)
		ctx = context.WithValue(ctx, delivery.LoginContextKey, "compliance_user")
		rec := httptest.NewRecorder()
		editHandler.ServeHTTP(rec, req.WithContext(ctx))
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("Edit: Operator forbidden", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/api/contracts/1", nil)
		ctx := context.WithValue(req.Context(), delivery.RoleContextKey, domain.RoleOperator)
		ctx = context.WithValue(ctx, delivery.LoginContextKey, "operator_user")
		rec := httptest.NewRecorder()
		editHandler.ServeHTTP(rec, req.WithContext(ctx))
		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", rec.Code)
		}
	})

	t.Run("Edit: BranchHead forbidden", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/api/contracts/1", nil)
		ctx := context.WithValue(req.Context(), delivery.RoleContextKey, domain.RoleBranchHead)
		ctx = context.WithValue(ctx, delivery.LoginContextKey, "branch_head_user")
		rec := httptest.NewRecorder()
		editHandler.ServeHTTP(rec, req.WithContext(ctx))
		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", rec.Code)
		}
	})

	t.Run("Edit: CurrencyControl allowed with permission", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/api/contracts/1", nil)
		ctx := context.WithValue(req.Context(), delivery.RoleContextKey, domain.RoleCurrencyControl)
		ctx = context.WithValue(ctx, delivery.LoginContextKey, "cc_user")
		rec := httptest.NewRecorder()
		editHandler.ServeHTTP(rec, req.WithContext(ctx))
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("Delete: CurrencyControl allowed to delete document into trash", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/contracts/1", nil)
		ctx := context.WithValue(req.Context(), delivery.RoleContextKey, domain.RoleCurrencyControl)
		ctx = context.WithValue(ctx, delivery.LoginContextKey, "cc_user")
		rec := httptest.NewRecorder()
		deleteHandler.ServeHTTP(rec, req.WithContext(ctx))
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("Delete: Operator forbidden", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/contracts/1", nil)
		ctx := context.WithValue(req.Context(), delivery.RoleContextKey, domain.RoleOperator)
		ctx = context.WithValue(ctx, delivery.LoginContextKey, "operator_user")
		rec := httptest.NewRecorder()
		deleteHandler.ServeHTTP(rec, req.WithContext(ctx))
		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", rec.Code)
		}
	})

	t.Run("Delete: BranchHead forbidden", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/contracts/1", nil)
		ctx := context.WithValue(req.Context(), delivery.RoleContextKey, domain.RoleBranchHead)
		ctx = context.WithValue(ctx, delivery.LoginContextKey, "branch_head_user")
		rec := httptest.NewRecorder()
		deleteHandler.ServeHTTP(rec, req.WithContext(ctx))
		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", rec.Code)
		}
	})
}
