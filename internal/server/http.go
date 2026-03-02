package server

import (
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"github.com/Dolyyyy/huma_golang_api_template/internal/config"
	routerpkg "github.com/Dolyyyy/huma_golang_api_template/internal/router"
	"github.com/Dolyyyy/huma_golang_api_template/internal/services"
)

// New builds the HTTP server and wires all API dependencies.
func New(cfg config.Config) *http.Server {
	mux := chi.NewRouter()
	mux.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/docs", http.StatusFound)
	})

	apiConfig := huma.DefaultConfig("Golang API Template", "1.0.0")
	api := humachi.New(mux, apiConfig)

	routerpkg.Register(api, routerpkg.Dependencies{
		HealthService: services.NewStaticHealthService(),
	})

	return &http.Server{
		Addr:              cfg.Address(),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
}
