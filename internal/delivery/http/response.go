package http

import (
	"encoding/json"
	"io"
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

func decodeJSON(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}

func parseDate(s string) *time.Time {
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
