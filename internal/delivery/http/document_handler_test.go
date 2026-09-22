package http_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	delivery "CurrencyControl/internal/delivery/http"
	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/errs"
	"CurrencyControl/internal/service"

	"github.com/gorilla/mux"
)

type mockDocumentRepo struct {
	files map[string]*domain.DocumentFileInfo
}

func newMockDocumentRepo() *mockDocumentRepo {
	return &mockDocumentRepo{
		files: make(map[string]*domain.DocumentFileInfo),
	}
}

func (m *mockDocumentRepo) key(entityType string, id int64) string {
	return fmt.Sprintf("%s:%d", strings.ToLower(entityType), id)
}

func (m *mockDocumentRepo) GetDocumentFileInfo(ctx context.Context, entityType string, id int64) (*domain.DocumentFileInfo, error) {
	norm := strings.ToLower(entityType)
	switch norm {
	case "contract", "invoice", "gtd", "additional_agreement", "payment_order", "gtd_extension":
	default:
		return nil, fmt.Errorf("неизвестный тип сущности: %s", entityType)
	}

	info, exists := m.files[m.key(norm, id)]
	if !exists {
		return nil, errs.ErrNotFound
	}
	return info, nil
}

func TestDocumentHandler_GetFile(t *testing.T) {
	// Создаем временную директорию с тестовыми файлами
	tempDir, err := os.MkdirTemp("", "doc_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Тестовый PDF
	pdfPath := filepath.Join(tempDir, "1774260344_dogovor_postavki.pdf")
	pdfContent := []byte("%PDF-1.4 test contract content")
	if err := os.WriteFile(pdfPath, pdfContent, 0644); err != nil {
		t.Fatalf("Failed to write test pdf: %v", err)
	}

	// Тестовый Word
	docxPath := filepath.Join(tempDir, "1774260345_schet_faktyra.docx")
	docxContent := []byte("docx dummy content")
	if err := os.WriteFile(docxPath, docxContent, 0644); err != nil {
		t.Fatalf("Failed to write test docx: %v", err)
	}

	// Тестовый ГТД
	gtdPath := filepath.Join(tempDir, "1774260346_tamozhnya_gtd.pdf")
	gtdContent := []byte("%PDF-1.4 test gtd content")
	if err := os.WriteFile(gtdPath, gtdContent, 0644); err != nil {
		t.Fatalf("Failed to write test gtd: %v", err)
	}

	repo := newMockDocumentRepo()
	repo.files[repo.key("contract", 1)] = &domain.DocumentFileInfo{
		EntityType:     "contract",
		EntityID:       1,
		DocumentNumber: "№100/2026",
		DocumentPath:   pdfPath,
	}
	repo.files[repo.key("invoice", 2)] = &domain.DocumentFileInfo{
		EntityType:     "invoice",
		EntityID:       2,
		DocumentNumber: "INV-2026-05",
		DocumentPath:   docxPath,
	}
	repo.files[repo.key("gtd", 3)] = &domain.DocumentFileInfo{
		EntityType:     "gtd",
		EntityID:       3,
		DocumentNumber: "GTD-9988",
		DocumentPath:   gtdPath,
	}
	// Документ без прикрепленного файла
	repo.files[repo.key("contract", 4)] = &domain.DocumentFileInfo{
		EntityType:     "contract",
		EntityID:       4,
		DocumentNumber: "NO-FILE",
		DocumentPath:   "",
	}
	// Документ со ссылкой на несуществующий файл
	repo.files[repo.key("contract", 5)] = &domain.DocumentFileInfo{
		EntityType:     "contract",
		EntityID:       5,
		DocumentNumber: "MISSING",
		DocumentPath:   filepath.Join(tempDir, "non_existent.pdf"),
	}
	// Документ на стадии согласования
	repo.files[repo.key("contract", 6)] = &domain.DocumentFileInfo{
		EntityType:     "contract",
		EntityID:       6,
		DocumentNumber: "PENDING-01",
		DocumentPath:   pdfPath,
		ApprovalStatus: domain.ApprovalStatusPendingCurrencyControl,
	}

	svc := service.NewDocumentService(repo)
	handler := delivery.NewDocumentHandler(svc)

	router := mux.NewRouter()
	router.HandleFunc("/api/documents/{entity_type}/{id:[0-9]+}/file", handler.GetFile).Methods(http.MethodGet)

	t.Run("Inline view for contract (PDF)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/documents/contract/1/file", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
		}

		disp := rr.Header().Get("Content-Disposition")
		if !strings.HasPrefix(disp, "inline") {
			t.Errorf("expected Content-Disposition inline, got %s", disp)
		}
		if !strings.Contains(disp, "dogovor_postavki.pdf") {
			t.Errorf("expected cleaned filename dogovor_postavki.pdf in disposition, got %s", disp)
		}

		cType := rr.Header().Get("Content-Type")
		if cType != "application/pdf" {
			t.Errorf("expected Content-Type application/pdf, got %s", cType)
		}

		if !strings.Contains(rr.Body.String(), "test contract content") {
			t.Errorf("expected body to contain contract content, got %s", rr.Body.String())
		}
	})

	t.Run("Attachment download for contract (?download=true)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/documents/contract/1/file?download=true", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rr.Code)
		}

		disp := rr.Header().Get("Content-Disposition")
		if !strings.HasPrefix(disp, "attachment") {
			t.Errorf("expected Content-Disposition attachment, got %s", disp)
		}
		if !strings.Contains(disp, "dogovor_postavki.pdf") {
			t.Errorf("expected filename dogovor_postavki.pdf in disposition, got %s", disp)
		}
	})

	t.Run("Attachment download via mode=download", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/documents/contract/1/file?mode=download", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rr.Code)
		}

		disp := rr.Header().Get("Content-Disposition")
		if !strings.HasPrefix(disp, "attachment") {
			t.Errorf("expected Content-Disposition attachment, got %s", disp)
		}
	})

	t.Run("Inline view for invoice (Word docx)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/documents/invoice/2/file", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rr.Code)
		}

		cType := rr.Header().Get("Content-Type")
		if cType != "application/vnd.openxmlformats-officedocument.wordprocessingml.document" {
			t.Errorf("expected word docx mime, got %s", cType)
		}
	})

	t.Run("Inline view for GTD", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/documents/gtd/3/file", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rr.Code)
		}
		if !strings.Contains(rr.Body.String(), "test gtd content") {
			t.Errorf("expected gtd content in body")
		}
	})

	t.Run("Document without attached file returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/documents/contract/4/file", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("Document with file missing on disk returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/documents/contract/5/file", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("Document not found returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/documents/contract/9999/file", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("Unknown entity type returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/documents/invalid_type/1/file", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("Invalid document ID (zero) returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/documents/contract/0/file", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("Pending approval returns 403 for operator", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/documents/contract/6/file", nil)
		ctx := context.WithValue(req.Context(), delivery.RoleContextKey, domain.RoleOperator)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req.WithContext(ctx))

		if rr.Code != http.StatusForbidden {
			t.Fatalf("expected status 403, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("Pending approval returns 200 for currency control", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/documents/contract/6/file", nil)
		ctx := context.WithValue(req.Context(), delivery.RoleContextKey, domain.RoleCurrencyControl)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req.WithContext(ctx))

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})
}
