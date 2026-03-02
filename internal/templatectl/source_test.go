package templatectl

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestSourceCandidatesUsesProvidedSource(t *testing.T) {
	t.Parallel()

	projectRoot := filepath.Join("C:", "workspace", "project")
	candidates := sourceCandidates(projectRoot, "catalog")

	expected := []string{filepath.Join(projectRoot, "catalog")}
	if !reflect.DeepEqual(candidates, expected) {
		t.Fatalf("unexpected candidates: got=%v want=%v", candidates, expected)
	}
}

func TestSourceCandidatesUsesEnvSource(t *testing.T) {
	projectRoot := filepath.Join("C:", "workspace", "project")
	t.Setenv(modulesSourceEnv, "shared-modules")

	candidates := sourceCandidates(projectRoot, "")

	expected := []string{filepath.Join(projectRoot, "shared-modules")}
	if !reflect.DeepEqual(candidates, expected) {
		t.Fatalf("unexpected candidates: got=%v want=%v", candidates, expected)
	}
}

func TestSourceCandidatesIncludesDefaultGitHubFallback(t *testing.T) {
	projectRoot := filepath.Join("C:", "workspace", "project")
	t.Setenv(modulesSourceEnv, "")

	candidates := sourceCandidates(projectRoot, "")

	expected := []string{
		filepath.Join(projectRoot, "huma_golang_api_template_modules"),
		filepath.Join(filepath.Dir(projectRoot), "huma_golang_api_template_modules"),
		defaultModulesSourceURL,
	}

	if !reflect.DeepEqual(candidates, expected) {
		t.Fatalf("unexpected candidates: got=%v want=%v", candidates, expected)
	}
}
