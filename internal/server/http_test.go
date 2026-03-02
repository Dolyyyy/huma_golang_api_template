package server

import (
	"net/http"
	"testing"

	"github.com/Dolyyyy/huma_golang_api_template/internal/config"
	"github.com/Dolyyyy/huma_golang_api_template/internal/domain"
	"github.com/Dolyyyy/huma_golang_api_template/internal/testutils"
)

func TestAPITestEndpoint(t *testing.T) {
	t.Parallel()

	srv := New(config.Config{Port: "0"})
	rec := testutils.PerformRequest(t, srv.Handler, http.MethodGet, "/api/test", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	body := testutils.DecodeJSON[domain.HealthStatus](t, rec)

	if !body.OK {
		t.Fatalf("expected ok=true, got ok=%v", body.OK)
	}

	expectedDetail := "API is up and ready to listen."
	if body.Detail != expectedDetail {
		t.Fatalf("expected detail %q, got %q", expectedDetail, body.Detail)
	}
}

func TestRootRedirectsToDocs(t *testing.T) {
	t.Parallel()

	srv := New(config.Config{Port: "0"})
	rec := testutils.PerformRequest(t, srv.Handler, http.MethodGet, "/", nil)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected status %d, got %d", http.StatusFound, rec.Code)
	}

	location := rec.Header().Get("Location")
	if location != "/docs" {
		t.Fatalf("expected redirect location %q, got %q", "/docs", location)
	}
}
