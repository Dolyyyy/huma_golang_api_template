package config

import "testing"

func TestValidateAcceptsDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := Config{}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected no validation error, got %v", err)
	}
}

func TestLoadDocsRendererNormalization(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "default empty", input: "", expected: DocsRendererSwaggerUI},
		{name: "swagger alias", input: "swagger", expected: DocsRendererSwaggerUI},
		{name: "swagger ui", input: "swagger-ui", expected: DocsRendererSwaggerUI},
		{name: "stoplight alias", input: "stoplight-elements", expected: DocsRendererStoplight},
		{name: "stoplight canonical", input: "stoplight", expected: DocsRendererStoplight},
		{name: "scalar", input: "scalar", expected: DocsRendererScalar},
		{name: "invalid fallback", input: "whatever", expected: DocsRendererSwaggerUI},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DOCS_RENDERER", tc.input)

			cfg := Load()
			if cfg.Docs.Renderer != tc.expected {
				t.Fatalf("expected renderer %q, got %q", tc.expected, cfg.Docs.Renderer)
			}
		})
	}
}

func TestValidateRejectsInvalidDocsRenderer(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Docs: DocsConfig{
			Renderer: "invalid",
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for invalid docs renderer")
	}
}
