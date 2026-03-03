package modulekit

import (
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
)

func TestRegisterLegacyRouteDocsAddsChiOnlyModuleRoutes(t *testing.T) {
	resetRegistry(t)
	Register(Definition{
		ID:   "db-sqlite",
		Name: "SQLite connector",
		Routes: func(router chi.Router) {
			router.Get("/db/sqlite/ping", func(http.ResponseWriter, *http.Request) {})
		},
	})

	api := humachi.New(chi.NewRouter(), huma.DefaultConfig("test", "1.0.0"))
	RegisterLegacyRouteDocs(api)

	pathItem := api.OpenAPI().Paths["/db/sqlite/ping"]
	if pathItem == nil {
		pathItem = api.OpenAPI().Paths["/db/sqlite/ping/"]
	}
	if pathItem == nil || pathItem.Get == nil {
		paths := make([]string, 0, len(api.OpenAPI().Paths))
		for path := range api.OpenAPI().Paths {
			paths = append(paths, path)
		}
		t.Fatalf("expected legacy chi route to be present in OpenAPI, got paths=%v", paths)
	}
}

func TestRegisterLegacyRouteDocsSkipsModulesWithHumaRoutes(t *testing.T) {
	resetRegistry(t)
	Register(Definition{
		ID:   "db-sqlite",
		Name: "SQLite connector",
		Routes: func(router chi.Router) {
			router.Get("/db/sqlite/ping", func(http.ResponseWriter, *http.Request) {})
		},
		HumaRoutes: func(api huma.API) {},
	})

	api := humachi.New(chi.NewRouter(), huma.DefaultConfig("test", "1.0.0"))
	RegisterLegacyRouteDocs(api)

	pathItem := api.OpenAPI().Paths["/db/sqlite/ping"]
	if pathItem != nil {
		t.Fatal("expected no legacy route docs when module provides HumaRoutes")
	}
}

func resetRegistry(t *testing.T) {
	t.Helper()

	registryMu.Lock()
	registry = map[string]Definition{}
	registryMu.Unlock()

	t.Cleanup(func() {
		registryMu.Lock()
		registry = map[string]Definition{}
		registryMu.Unlock()
	})
}
