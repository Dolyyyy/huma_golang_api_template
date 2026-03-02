package handlers

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/Dolyyyy/huma_golang_api_template/internal/domain"
	"github.com/Dolyyyy/huma_golang_api_template/internal/services"
)

// HealthHandler exposes transport-level operations for health checks.
type HealthHandler struct {
	service services.HealthService
}

// HealthOutput maps the operation payload returned to the client.
type HealthOutput struct {
	Body domain.HealthStatus
}

func NewHealthHandler(service services.HealthService) *HealthHandler {
	return &HealthHandler{
		service: service,
	}
}

// Register wires the handler operations into the Huma API.
func (h *HealthHandler) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "get-api-test",
		Method:      http.MethodGet,
		Path:        "/api/test",
		Summary:     "Return a simple API readiness payload",
		Tags:        []string{"health"},
	}, h.getAPITest)
}

// getAPITest delegates business logic to the service layer.
func (h *HealthHandler) getAPITest(ctx context.Context, _ *struct{}) (*HealthOutput, error) {
	status := h.service.GetStatus(ctx)

	return &HealthOutput{
		Body: status,
	}, nil
}
