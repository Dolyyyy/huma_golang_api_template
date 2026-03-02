package testutils

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// PerformRequest runs a handler against a synthetic HTTP request.
func PerformRequest(t *testing.T, handler http.Handler, method, path string, body io.Reader) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, body)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return perform(t, handler, req)
}

// PerformRequestWithHeaders runs a request and applies custom headers.
func PerformRequestWithHeaders(t *testing.T, handler http.Handler, method, path string, body io.Reader, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, body)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return perform(t, handler, req)
}

func perform(t *testing.T, handler http.Handler, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	return rec
}

// PerformJSONRequest marshals payload into JSON before executing the request.
func PerformJSONRequest(t *testing.T, handler http.Handler, method, path string, payload any) *httptest.ResponseRecorder {
	t.Helper()

	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("failed to marshal payload: %v", err)
		}
		body = strings.NewReader(string(raw))
	}

	return PerformRequest(t, handler, method, path, body)
}

// DecodeJSON decodes a JSON response body into the requested type.
func DecodeJSON[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()

	var value T
	if err := json.Unmarshal(rec.Body.Bytes(), &value); err != nil {
		t.Fatalf("response body should be valid JSON: %v", err)
	}

	return value
}
