package http

import (
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
}

func newLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func (lrw *loggingResponseWriter) Flush() {
	if f, ok := lrw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func getClientIP(r *http.Request) string {
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
		lrw := newLoggingResponseWriter(w)

		next.ServeHTTP(lrw, r)

		latency := time.Since(start)
		statusCode := lrw.statusCode

		log := logger.GetLogger()
		var event *zerolog.Event
		switch {
		case statusCode >= http.StatusInternalServerError:
			event = log.Error()
		case statusCode >= http.StatusBadRequest:
			event = log.Warn()
		default:
			event = log.Info()
		}

		event.
			Int("status", statusCode).
			Str("method", r.Method).
			Str("path", r.URL.RequestURI()).
			Str("ip", getClientIP(r)).
			Dur("latency", latency).
			Str("user_agent", r.UserAgent()).
			Msg("HTTP request")
	})
}
