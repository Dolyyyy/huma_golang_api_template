package services

import (
	"context"

	"github.com/Dolyyyy/golang_api_template/internal/domain"
)

// HealthService defines the business contract for health-style responses.
type HealthService interface {
	GetStatus(context.Context) domain.HealthStatus
}

// StaticHealthService returns a static response for the starter template.
type StaticHealthService struct{}

func NewStaticHealthService() *StaticHealthService {
	return &StaticHealthService{}
}

// GetStatus keeps the first endpoint deterministic and easy to test.
func (s *StaticHealthService) GetStatus(context.Context) domain.HealthStatus {
	return domain.HealthStatus{
		OK:     true,
		Detail: "API is up and ready to listen.",
	}
}
