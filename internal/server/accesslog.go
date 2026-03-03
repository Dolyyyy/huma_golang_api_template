package server

import (
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/Dolyyyy/huma_golang_api_template/internal/logger"
)

func newAccessLogMiddleware(logFn func(entry logger.AccessLogEntry)) func(http.Handler) http.Handler {
	if logFn == nil {
		return func(next http.Handler) http.Handler { return next }
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			wrapped := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(wrapped, r)

			status := wrapped.Status()
			if status == 0 {
				status = http.StatusOK
			}

			logFn(logger.AccessLogEntry{
				Method:   r.Method,
				Target:   r.URL.RequestURI(),
				Proto:    r.Proto,
				Status:   status,
				Bytes:    wrapped.BytesWritten(),
				Duration: time.Since(started),
				RemoteIP: clientIP(r),
			})
		})
	}
}

func clientIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		if first, _, ok := strings.Cut(forwarded, ","); ok {
			return strings.TrimSpace(first)
		}
		return forwarded
	}

	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}

	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}

	return strings.TrimSpace(r.RemoteAddr)
}
