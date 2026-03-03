package server

import (
	"net/http"
	"strings"
	"testing"

	"github.com/Dolyyyy/huma_golang_api_template/internal/config"
	"github.com/Dolyyyy/huma_golang_api_template/internal/domain"
	"github.com/Dolyyyy/huma_golang_api_template/internal/testutils"
)

func TestAPITestEndpoint(t *testing.T) {
	t.Parallel()

	srv := New(config.Config{Port: "0"}, nil)
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

	srv := New(config.Config{Port: "0"}, nil)
	rec := testutils.PerformRequest(t, srv.Handler, http.MethodGet, "/", nil)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected status %d, got %d", http.StatusFound, rec.Code)
	}

	location := rec.Header().Get("Location")
	if location != "/docs" {
		t.Fatalf("expected redirect location %q, got %q", "/docs", location)
	}
}

func TestOpenAPIIncludesCoreEndpoints(t *testing.T) {
	t.Parallel()

	srv := New(config.Config{Port: "0"}, nil)
	rec := testutils.PerformRequest(t, srv.Handler, http.MethodGet, "/openapi.yaml", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	body := rec.Body.String()
	expectedPaths := []string{"/api/test:"}

	for _, path := range expectedPaths {
		if !strings.Contains(body, path) {
			t.Fatalf("expected OpenAPI to contain %q", path)
		}
	}
}

func TestDocsRendererDefaultsToSwaggerUI(t *testing.T) {
	t.Parallel()

	srv := New(config.Config{Port: "0"}, nil)
	rec := testutils.PerformRequest(t, srv.Handler, http.MethodGet, "/docs", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if !strings.Contains(strings.ToLower(rec.Body.String()), "swagger") {
		t.Fatal("expected docs page to use swagger renderer by default")
	}
}

func TestDocsRendererCanBeScalar(t *testing.T) {
	t.Parallel()

	srv := New(config.Config{
		Port: "0",
		Docs: config.DocsConfig{
			Renderer: config.DocsRendererScalar,
		},
	}, nil)
	rec := testutils.PerformRequest(t, srv.Handler, http.MethodGet, "/docs", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if !strings.Contains(strings.ToLower(rec.Body.String()), "scalar") {
		t.Fatal("expected docs page to use scalar renderer")
	}
}
