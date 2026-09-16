package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoggerMiddleware_StatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		clientIP   string
		forwarded  string
		realIP     string
	}{
		{
			name:       "200 OK",
			statusCode: http.StatusOK,
			clientIP:   "192.168.1.1:12345",
		},
		{
			name:       "404 Not Found",
			statusCode: http.StatusNotFound,
			clientIP:   "192.168.1.1:12345",
			realIP:     "10.0.0.1",
		},
		{
			name:       "500 Internal Server Error",
			statusCode: http.StatusInternalServerError,
			clientIP:   "192.168.1.1:12345",
			forwarded:  "172.16.0.1, 10.0.0.2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte("ok"))
			})

			middleware := LoggerMiddleware(handler)

			req := httptest.NewRequest(http.MethodGet, "/test/path?foo=bar", nil)
			req.RemoteAddr = tt.clientIP
			if tt.forwarded != "" {
				req.Header.Set("X-Forwarded-For", tt.forwarded)
			}
			if tt.realIP != "" {
				req.Header.Set("X-Real-IP", tt.realIP)
			}
			req.Header.Set("User-Agent", "TestAgent/1.0")

			rec := httptest.NewRecorder()
			middleware.ServeHTTP(rec, req)

			if rec.Code != tt.statusCode {
				t.Errorf("expected status %d, got %d", tt.statusCode, rec.Code)
			}
		})
	}
}

func TestLoggingResponseWriter_Flush(t *testing.T) {
	rec := httptest.NewRecorder()
	lrw := newLoggingResponseWriter(rec)
	lrw.Flush() // httptest.ResponseRecorder implements Flusher
	if !rec.Flushed {
		t.Errorf("expected recorder to be flushed")
	}
}
