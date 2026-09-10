package http

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/mattn/go-colorable"
)

const (
	greenBg   = "\033[97;42m"
	whiteBg   = "\033[90;47m"
	yellowBg  = "\033[90;43m"
	redBg     = "\033[97;41m"
	blueBg    = "\033[97;44m"
	magentaBg = "\033[97;45m"
	cyanBg    = "\033[97;46m"
	reset     = "\033[0m"
)

var stdOutput io.Writer = colorable.NewColorableStdout()

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

func colorForStatus(code int) string {
	switch {
	case code >= http.StatusOK && code < http.StatusMultipleChoices:
		return greenBg
	case code >= http.StatusMultipleChoices && code < http.StatusBadRequest:
		return whiteBg
	case code >= http.StatusBadRequest && code < http.StatusInternalServerError:
		return yellowBg
	default:
		return redBg
	}
}

func colorForMethod(method string) string {
	switch method {
	case http.MethodGet:
		return blueBg
	case http.MethodPost:
		return cyanBg
	case http.MethodPut:
		return yellowBg
	case http.MethodDelete:
		return redBg
	case http.MethodPatch:
		return greenBg
	case http.MethodHead:
		return magentaBg
	case http.MethodOptions:
		return whiteBg
	default:
		return reset
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

		end := time.Now()
		latency := end.Sub(start)
		clientIP := getClientIP(r)
		method := r.Method
		statusCode := lrw.statusCode
		statusColor := colorForStatus(statusCode)
		methodColor := colorForMethod(method)
		path := r.URL.RequestURI()

		fmt.Fprintf(stdOutput, "[GIN] %v |%s %3d %s| %13v | %15s |%s %-7s %s %#v\n",
			end.Format("2006/01/02 - 15:04:05"),
			statusColor, statusCode, reset,
			latency,
			clientIP,
			methodColor, method, reset,
			path,
		)
	})
}
