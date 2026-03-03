package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Dolyyyy/huma_golang_api_template/internal/logger"
)

func TestAccessLogMiddlewareCapturesRequestMetadata(t *testing.T) {
	t.Parallel()

	entries := make([]logger.AccessLogEntry, 0, 1)
	handler := newAccessLogMiddleware(func(entry logger.AccessLogEntry) {
		entries = append(entries, entry)
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "missing", http.StatusNotFound)
	}))

	req := httptest.NewRequest(http.MethodGet, "http://example.com/missing?debug=1", nil)
	req.RemoteAddr = "10.0.0.8:3210"
	req.Header.Set("X-Forwarded-For", "203.0.113.9, 10.0.0.8")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 access log entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Method != http.MethodGet {
		t.Fatalf("expected method %q, got %q", http.MethodGet, entry.Method)
	}
	if entry.Target != "/missing?debug=1" {
		t.Fatalf("expected target %q, got %q", "/missing?debug=1", entry.Target)
	}
	if entry.Status != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, entry.Status)
	}
	if entry.RemoteIP != "203.0.113.9" {
		t.Fatalf("expected remote IP %q, got %q", "203.0.113.9", entry.RemoteIP)
	}
	if entry.Bytes <= 0 {
		t.Fatalf("expected positive bytes written, got %d", entry.Bytes)
	}
}

func TestAccessLogMiddlewareDefaultsStatus200(t *testing.T) {
	t.Parallel()

	entries := make([]logger.AccessLogEntry, 0, 1)
	handler := newAccessLogMiddleware(func(entry logger.AccessLogEntry) {
		entries = append(entries, entry)
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "http://example.com/ok", nil)
	req.RemoteAddr = "127.0.0.1:9898"

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if len(entries) != 1 {
		t.Fatalf("expected 1 access log entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Status != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, entry.Status)
	}
	if entry.Bytes != 2 {
		t.Fatalf("expected 2 bytes written, got %d", entry.Bytes)
	}
}
