package service

import (
	"context"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"CurrencyControl/internal/service/ports"
)

type documentService struct {
	repo ports.DocumentRepository
}

func NewDocumentService(repo ports.DocumentRepository) ports.DocumentService {
	return &documentService{repo: repo}
}

func (s *documentService) GetDocumentFile(ctx context.Context, role, entityType string, id int64) (*ports.DocumentFileResult, error) {
	if id <= 0 {
		return nil, fmt.Errorf("некорректный ID документа: %d", id)
	}

	info, err := s.repo.GetDocumentFileInfo(ctx, entityType, id)
	if err != nil {
		return nil, err
	}

	if info.DocumentPath == "" {
		return nil, fmt.Errorf("файл документа не прикреплен к данной записи")
	}
	cleanPath := filepath.Clean(info.DocumentPath)

	// Защита от path traversal для относительных путей:
	// filepath.Clean разрешает ".." компоненты — проверяем что результат не уходит
	// выше рабочей директории. Для абсолютных путей проверяем отдельно.
	if !filepath.IsAbs(cleanPath) {
		absPath, err := filepath.Abs(cleanPath)
		if err != nil {
			return nil, fmt.Errorf("недопустимый путь к файлу")
		}
		workDir, err := filepath.Abs(".")
		if err != nil {
			return nil, fmt.Errorf("недопустимый путь к файлу")
		}
		if !strings.HasPrefix(absPath, workDir+string(filepath.Separator)) {
			return nil, fmt.Errorf("недопустимый путь к файлу")
		}
		cleanPath = absPath
	}

	stat, err := os.Stat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("файл документа отсутствует на сервере: %s", cleanPath)
		}
		return nil, fmt.Errorf("ошибка при чтении файла: %w", err)
	}

	if stat.IsDir() {
		return nil, fmt.Errorf("путь указывает на каталог, а не на файл")
	}

	fileName := extractCleanFileName(cleanPath, info.EntityType, id, info.DocumentNumber)
	contentType := detectContentType(cleanPath)

	return &ports.DocumentFileResult{
		FilePath:    cleanPath,
		FileName:    fileName,
		ContentType: contentType,
	}, nil
}

func extractCleanFileName(pathStr, entityType string, id int64, docNumber string) string {
	base := filepath.Base(pathStr)
	parts := strings.SplitN(base, "_", 2)
	if len(parts) == 2 && isAllDigits(parts[0]) && len(parts[1]) > 0 {
		return parts[1]
	}
	if base != "" && base != "." {
		return base
	}
	ext := filepath.Ext(pathStr)
	if ext == "" {
		ext = ".pdf"
	}
	if docNumber != "" {
		return fmt.Sprintf("%s_%s%s", entityType, docNumber, ext)
	}
	return fmt.Sprintf("%s_%d%s", entityType, id, ext)
}

func isAllDigits(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func detectContentType(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".pdf":
		return "application/pdf"
	default:
		if mimeType := mime.TypeByExtension(ext); mimeType != "" {
			return mimeType
		}
		return "application/octet-stream"
	}
}
