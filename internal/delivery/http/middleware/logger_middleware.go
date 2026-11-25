package middleware

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"CurrencyControl/internal/logger"
	"github.com/rs/zerolog"
)

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

func newLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		body:           &bytes.Buffer{},
	}
}
func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}
func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
	if lrw.body.Len() < 2048 {
		lrw.body.Write(b)
	}
	return lrw.ResponseWriter.Write(b)
}

func (lrw *loggingResponseWriter) Flush() {
	if f, ok := lrw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func GetClientIP(r *http.Request) string {
	clientIP := r.Header.Get("X-Forwarded-For")
	if clientIP != "" {
		if idx := strings.Index(clientIP, ","); idx != -1 {
			return strings.TrimSpace(clientIP[:idx])
		}
		return strings.TrimSpace(clientIP)
	}
	clientIP = r.Header.Get("X-Real-IP")
	if clientIP != "" {
		return strings.TrimSpace(clientIP)
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

func LoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		var reqBody string
		if r.Body != nil && (r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch) {
			contentType := r.Header.Get("Content-Type")
			if !strings.HasPrefix(contentType, "multipart/form-data") {
				bodyBytes, err := io.ReadAll(r.Body)
				if err == nil {
					r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
					trimmed := strings.TrimSpace(string(bodyBytes))
					if len(trimmed) > 2048 {
						trimmed = trimmed[:2048] + "..."
					}
					reqBody = trimmed
				}
			}
		}
		lrw := newLoggingResponseWriter(w)
		next.ServeHTTP(lrw, r)
		latency := time.Since(start)
		statusCode := lrw.statusCode
		log := logger.GetLogger()
		var event *zerolog.Event
		switch {
		case statusCode >= 500:
			event = log.Error()
		case statusCode >= 400:
			event = log.Warn()
		default:
			event = log.Info()
		}
		event.
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("status", statusCode).
			Str("ip", GetClientIP(r)).
			Dur("latency", latency)
		if statusCode >= 400 {
			if r.URL.RawQuery != "" {
				event.Str("query_params", r.URL.RawQuery)
			}
			if reqBody != "" {
				event.Str("request_fields", reqBody)
			}
			if lrw.body.Len() > 0 {
				event.Str("error", strings.TrimSpace(lrw.body.String()))
			}
		}

		event.Msg("HTTP request")
	})
}
