package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"CurrencyControl/internal/domain"
)

type mockDocRepoForService struct {
	info *domain.DocumentFileInfo
	err  error
}

func (m *mockDocRepoForService) GetDocumentFileInfo(ctx context.Context, entityType string, id int64) (*domain.DocumentFileInfo, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.info, nil
}

func TestDocumentService_GetDocumentFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "doc_svc_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	validFilePath := filepath.Join(tempDir, "171234567890_Договор_№12.pdf")
	if err := os.WriteFile(validFilePath, []byte("%PDF test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	t.Run("Valid file with cyrillic name and timestamp stripping", func(t *testing.T) {
		repo := &mockDocRepoForService{
			info: &domain.DocumentFileInfo{
				EntityType:     "contract",
				EntityID:       10,
				DocumentNumber: "12",
				DocumentPath:   validFilePath,
			},
		}
		svc := NewDocumentService(repo)

		res, err := svc.GetDocumentFile(context.Background(), "contract", 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if res.FileName != "Договор_№12.pdf" {
			t.Errorf("expected cleaned filename Договор_№12.pdf, got %s", res.FileName)
		}
		if res.ContentType != "application/pdf" {
			t.Errorf("expected application/pdf, got %s", res.ContentType)
		}
	})

	t.Run("Path traversal attempt is blocked", func(t *testing.T) {
		repo := &mockDocRepoForService{
			info: &domain.DocumentFileInfo{
				EntityType:     "contract",
				EntityID:       10,
				DocumentNumber: "12",
				DocumentPath:   "../secret/file.txt",
			},
		}
		svc := NewDocumentService(repo)

		_, err := svc.GetDocumentFile(context.Background(), "contract", 10)
		if err == nil {
			t.Fatal("expected path traversal error, got nil")
		}
		if !strings.Contains(err.Error(), "недопустимый путь") {
			t.Errorf("expected 'недопустимый путь' error, got %v", err)
		}
	})

	t.Run("Directory path instead of file is rejected", func(t *testing.T) {
		repo := &mockDocRepoForService{
			info: &domain.DocumentFileInfo{
				EntityType:     "contract",
				EntityID:       10,
				DocumentNumber: "12",
				DocumentPath:   tempDir,
			},
		}
		svc := NewDocumentService(repo)

		_, err := svc.GetDocumentFile(context.Background(), "contract", 10)
		if err == nil {
			t.Fatal("expected directory error, got nil")
		}
		if !strings.Contains(err.Error(), "каталог") {
			t.Errorf("expected 'каталог' error, got %v", err)
		}
	})

	t.Run("Empty path is rejected", func(t *testing.T) {
		repo := &mockDocRepoForService{
			info: &domain.DocumentFileInfo{
				EntityType:     "contract",
				EntityID:       10,
				DocumentNumber: "12",
				DocumentPath:   "",
			},
		}
		svc := NewDocumentService(repo)

		_, err := svc.GetDocumentFile(context.Background(), "contract", 10)
		if err == nil {
			t.Fatal("expected empty path error, got nil")
		}
		if !strings.Contains(err.Error(), "не прикреплен") {
			t.Errorf("expected 'не прикреплен' error, got %v", err)
		}
	})
}
