package http

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type CommonError struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

// parseDate пробует распарсить дату из разных форматов
func parseDate(s string) *time.Time {
	s = strings.TrimSpace(s)
	// Если пришёл ISO формат с временем - обрезаем до даты
	if len(s) > 10 {
		s = s[:10]
	}
	formats := []string{
		"2006-01-02", // YYYY-MM-DD (основной)
		"02.01.2006", // DD.MM.YYYY
		"01/02/2006", // MM/DD/YYYY
	}
	for _, f := range formats {
		if d, err := time.Parse(f, s); err == nil {
			return &d
		}
	}
	return nil
}
