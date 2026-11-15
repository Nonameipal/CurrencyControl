package response

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type CommonError struct {
	Error string `json:"error"`
}

func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

func DecodeJSON(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

func ParseDate(s string) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	datePart := s
	if idx := strings.IndexAny(s, " T"); idx != -1 {
		datePart = s[:idx]
	}
	formats := []string{
		"2006-01-02",
		"02.01.2006",
		"2.1.2006",
		"02.01.06",
		"2006.01.02",
		"2006.1.2",
		"02/01/2006",
		"2/1/2006",
		"01/02/2006",
		"1/2/2006",
		"02-01-2006",
		"2-1-2006",
		time.RFC3339,
		"2006-01-02 15:04:05",
	}
	for _, f := range formats {
		if d, err := time.Parse(f, datePart); err == nil {
			return &d
		}
		if d, err := time.Parse(f, s); err == nil {
			return &d
		}
	}
	return nil
}

func ParseID(r *http.Request, key string) (int64, error) {
	valStr := mux.Vars(r)[key]
	if valStr == "" || strings.HasPrefix(valStr, "{") {
		if f := strings.TrimSpace(r.FormValue(key)); f != "" {
			valStr = f
		} else if q := strings.TrimSpace(r.URL.Query().Get(key)); q != "" {
			valStr = q
		}
	}
	if valStr == "" || strings.HasPrefix(valStr, "{") {
		return 0, fmt.Errorf("параметр %s отсутствует", key)
	}
	val, err := strconv.ParseInt(valStr, 10, 64)
	if err != nil || val <= 0 {
		return 0, fmt.Errorf("некорректный идентификатор для %s", key)
	}
	return val, nil
}

func RequireID(w http.ResponseWriter, r *http.Request, key string) (int64, bool) {
	id, err := ParseID(r, key)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, CommonError{Error: err.Error()})
		return 0, false
	}
	return id, true
}

var defaultDocExts = []string{".pdf"}

func SaveUploadedFile(r *http.Request, formKey, targetDir string, required bool) (string, error) {
	file, handler, err := r.FormFile(formKey)
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) && !required {
			return "", nil
		}
		if !required && file == nil {
			return "", nil
		}
		return "", fmt.Errorf("Файл документа обязателен")
	}
	defer file.Close()
	ext := strings.ToLower(filepath.Ext(handler.Filename))
	valid := false
	for _, e := range defaultDocExts {
		if ext == e {
			valid = true
			break
		}
	}
	if !valid {
		return "", fmt.Errorf("Разрешены только файлы формата PDF (.pdf)")
	}
	header := make([]byte, 4)
	n, _ := io.ReadFull(file, header)
	if seeker, ok := file.(io.Seeker); ok {
		_, _ = seeker.Seek(0, io.SeekStart)
	}
	if n < 4 || string(header[:4]) != "%PDF" {
		return "", fmt.Errorf("Файл поврежден или не является корректным PDF документом")
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", fmt.Errorf("Ошибка при сохранении файла на сервер: %w", err)
	}
	safeFileName := filepath.Base(handler.Filename)
	filePath := filepath.Join(targetDir, fmt.Sprintf("%d_%s", time.Now().UnixNano(), safeFileName))
	dst, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("Ошибка при сохранении файла на сервер")
	}
	defer dst.Close()
	if _, err := io.Copy(dst, file); err != nil {
		return "", fmt.Errorf("Ошибка при записи файла")
	}
	return filePath, nil
}

// Lowercase aliases for convenience
func writeJSON(w http.ResponseWriter, status int, data interface{}) { WriteJSON(w, status, data) }
func decodeJSON(r *http.Request, v interface{}) error              { return DecodeJSON(r, v) }
func parseDate(s string) *time.Time                                { return ParseDate(s) }
func parseID(r *http.Request, key string) (int64, error)           { return ParseID(r, key) }
func requireID(w http.ResponseWriter, r *http.Request, key string) (int64, bool) {
	return RequireID(w, r, key)
}
func saveUploadedFile(r *http.Request, formKey, targetDir string, required bool) (string, error) {
	return SaveUploadedFile(r, formKey, targetDir, required)
}
