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
	if len(s) > 10 {
		s = s[:10]
	}
	formats := []string{
		"2006-01-02", 
		"02.01.2006", 
		"01/02/2006", 
	}
	for _, f := range formats {
		if d, err := time.Parse(f, s); err == nil {
			return &d
		}
	}
	return nil
}
