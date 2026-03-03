package server

import (
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"github.com/Dolyyyy/huma_golang_api_template/internal/config"
	"github.com/Dolyyyy/huma_golang_api_template/internal/logger"
	"github.com/Dolyyyy/huma_golang_api_template/internal/modulekit"
	_ "github.com/Dolyyyy/huma_golang_api_template/internal/modules"
	routerpkg "github.com/Dolyyyy/huma_golang_api_template/internal/router"
	"github.com/Dolyyyy/huma_golang_api_template/internal/services"
)

// New builds the HTTP server and wires all API dependencies.
func New(cfg config.Config, appLogger *logger.Logger) *http.Server {
	mux := chi.NewRouter()
	mux.Use(newAccessLogMiddleware(func(entry logger.AccessLogEntry) {
		if appLogger == nil {
			return
		}
		appLogger.Access(entry)
	}))

	modulekit.ApplyMiddlewares(mux)

	mux.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/docs", http.StatusFound)
	})

	modulekit.RegisterRoutes(mux)

	apiConfig := huma.DefaultConfig("Golang API Template", "1.0.0")
	switch cfg.Docs.Renderer {
	case config.DocsRendererScalar:
		apiConfig.DocsRenderer = huma.DocsRendererScalar
	case config.DocsRendererStoplight:
		apiConfig.DocsRenderer = huma.DocsRendererStoplightElements
	default:
		apiConfig.DocsRenderer = huma.DocsRendererSwaggerUI
	}

	api := humachi.New(mux, apiConfig)

	routerpkg.Register(api, routerpkg.Dependencies{
		HealthService: services.NewStaticHealthService(),
	})
	modulekit.RegisterHumaRoutes(api)
	modulekit.RegisterLegacyRouteDocs(api)

	return &http.Server{
		Addr:              cfg.Address(),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
}
