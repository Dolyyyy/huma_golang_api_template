package metrics_prometheus

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/Dolyyyy/huma_golang_api_template/internal/modulekit"
)

func init() {
	modulekit.Register(modulekit.Definition{
		ID:          "metrics-prometheus",
		Name:        "Prometheus metrics",
		Description: "Expose /metrics for Prometheus scraping.",
		Routes:      registerRoutes,
	})
}

func registerRoutes(router chi.Router) {
	router.Handle("/metrics", promhttp.Handler())
	router.Get("/metrics/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/metrics", http.StatusMovedPermanently)
	})
}
