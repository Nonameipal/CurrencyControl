package http

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestParseDate(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Time
	}{
		{"13.09.2026", time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)},
		{"01.05.2025", time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC)},
		{"5.9.2026", time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)},
		{"2026-09-13", time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)},
		{"2026.09.13", time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)},
		{"13/09/2026", time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)},
		{"13.09.2026 14:30:00", time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)},
		{"2026-09-13T10:00:00Z", time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			parsed := parseDate(tc.input)
			if parsed == nil {
				t.Fatalf("expected parsed date for %q, got nil", tc.input)
			}
			if parsed.Year() != tc.expected.Year() || parsed.Month() != tc.expected.Month() || parsed.Day() != tc.expected.Day() {
				t.Errorf("for input %q expected %v, got %v", tc.input, tc.expected, *parsed)
			}
		})
	}

	if parseDate("") != nil {
		t.Errorf("expected empty string to return nil")
	}
	if parseDate("not-a-date") != nil {
		t.Errorf("expected invalid string to return nil")
	}
}

func TestSaveUploadedFile_OnlyPDF(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "upload_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	createRequest := func(fieldName, filename string, content []byte) *http.Request {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		if filename != "" {
			part, _ := writer.CreateFormFile(fieldName, filename)
			if len(content) > 0 {
				_, _ = part.Write(content)
			}
		}
		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/test-upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		return req
	}

	t.Run("Valid PDF with %PDF header succeeds", func(t *testing.T) {
		req := createRequest("document", "contract.pdf", []byte("%PDF-1.4 valid pdf content"))
		path, err := saveUploadedFile(req, "document", tempDir, true)
		if err != nil {
			t.Fatalf("unexpected error for valid pdf: %v", err)
		}
		if path == "" {
			t.Fatalf("expected non-empty file path")
		}
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected file to exist on disk: %v", err)
		}
	})

	t.Run("Valid PDF with uppercase .PDF extension succeeds", func(t *testing.T) {
		req := createRequest("document", "contract.PDF", []byte("%PDF-1.7 uppercase extension test"))
		path, err := saveUploadedFile(req, "document", tempDir, true)
		if err != nil {
			t.Fatalf("unexpected error for .PDF: %v", err)
		}
		if path == "" {
			t.Fatalf("expected non-empty file path")
		}
	})

	t.Run("DOCX file is rejected", func(t *testing.T) {
		req := createRequest("document", "contract.docx", []byte("PK\x03\x04 dummy docx"))
		_, err := saveUploadedFile(req, "document", tempDir, true)
		if err == nil {
			t.Fatalf("expected error for .docx file, got nil")
		}
		expectedMsg := "Разрешены только файлы формата PDF (.pdf)"
		if err.Error() != expectedMsg {
			t.Errorf("got %q, want %q", err.Error(), expectedMsg)
		}
	})

	t.Run("DOC file is rejected", func(t *testing.T) {
		req := createRequest("document", "contract.doc", []byte("\xD0\xCF\x11\xE0 dummy doc"))
		_, err := saveUploadedFile(req, "document", tempDir, true)
		if err == nil {
			t.Fatalf("expected error for .doc file, got nil")
		}
		expectedMsg := "Разрешены только файлы формата PDF (.pdf)"
		if err.Error() != expectedMsg {
			t.Errorf("got %q, want %q", err.Error(), expectedMsg)
		}
	})

	t.Run("TXT file is rejected", func(t *testing.T) {
		req := createRequest("document", "notes.txt", []byte("some text"))
		_, err := saveUploadedFile(req, "document", tempDir, true)
		if err == nil {
			t.Fatalf("expected error for .txt file, got nil")
		}
	})

	t.Run("Fake PDF with .pdf extension but invalid header is rejected", func(t *testing.T) {
		req := createRequest("document", "fake.pdf", []byte("THIS IS NOT A PDF FILE"))
		_, err := saveUploadedFile(req, "document", tempDir, true)
		if err == nil {
			t.Fatalf("expected error for fake pdf, got nil")
		}
		expectedMsg := "Файл поврежден или не является корректным PDF документом"
		if err.Error() != expectedMsg {
			t.Errorf("got %q, want %q", err.Error(), expectedMsg)
		}
	})

	t.Run("Empty 0-byte file with .pdf extension is rejected", func(t *testing.T) {
		req := createRequest("document", "empty.pdf", []byte{})
		_, err := saveUploadedFile(req, "document", tempDir, true)
		if err == nil {
			t.Fatalf("expected error for 0-byte pdf, got nil")
		}
		expectedMsg := "Файл поврежден или не является корректным PDF документом"
		if err.Error() != expectedMsg {
			t.Errorf("got %q, want %q", err.Error(), expectedMsg)
		}
	})

	t.Run("Optional document omitted returns empty path without error", func(t *testing.T) {
		req := createRequest("document", "", nil)
		path, err := saveUploadedFile(req, "document", tempDir, false)
		if err != nil {
			t.Fatalf("unexpected error when optional document omitted: %v", err)
		}
		if path != "" {
			t.Fatalf("expected empty path, got %q", path)
		}
	})

	t.Run("Required document omitted returns error", func(t *testing.T) {
		req := createRequest("document", "", nil)
		_, err := saveUploadedFile(req, "document", tempDir, true)
		if err == nil {
			t.Fatalf("expected error when required document omitted, got nil")
		}
		expectedMsg := "Файл документа обязателен"
		if err.Error() != expectedMsg {
			t.Errorf("got %q, want %q", err.Error(), expectedMsg)
		}
	})
}

