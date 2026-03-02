package router

import (
	"github.com/danielgtaylor/huma/v2"

	"github.com/Dolyyyy/huma_golang_api_template/internal/handlers"
	"github.com/Dolyyyy/huma_golang_api_template/internal/services"
)

// Dependencies groups services needed by route handlers.
type Dependencies struct {
	HealthService services.HealthService
}

// Register attaches all API operations to the Huma instance.
func Register(api huma.API, deps Dependencies) {
	healthHandler := handlers.NewHealthHandler(deps.HealthService)
	healthHandler.Register(api)
}
