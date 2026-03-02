package config

import "testing"

func TestValidateAcceptsDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := Config{}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected no validation error, got %v", err)
	}
}
