package domain

// HealthStatus is returned by readiness/health-check style endpoints.
type HealthStatus struct {
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}
