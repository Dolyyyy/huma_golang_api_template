package templatectl

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListIncludesCatalogModules(t *testing.T) {
	t.Parallel()

	projectRoot := prepareProjectRoot(t)
	sourceRoot := prepareModulesSource(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := RunWithRoot([]string{"--source", sourceRoot, "list"}, projectRoot, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr=%s)", code, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "auth-token [available]") {
		t.Fatalf("expected auth-token in output, got:\n%s", output)
	}
	if !strings.Contains(output, "metrics-prometheus [available]") {
		t.Fatalf("expected metrics-prometheus in output, got:\n%s", output)
	}
	if !strings.Contains(output, "01. auth-token [available]") {
		t.Fatalf("expected numbered module entry for auth-token, got:\n%s", output)
	}
	if !strings.Contains(output, "Total: 2 | Installed: 0 | Available: 2") {
		t.Fatalf("expected catalog summary, got:\n%s", output)
	}
}

func TestAddWritesModuleFilesLockAndImports(t *testing.T) {
	t.Parallel()

	projectRoot := prepareProjectRoot(t)
	sourceRoot := prepareModulesSource(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := RunWithRoot([]string{"--source", sourceRoot, "--skip-verify", "add", "auth-token"}, projectRoot, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr=%s)", code, stderr.String())
	}

	moduleFile := filepath.Join(projectRoot, "internal", "modules", "auth_token", "module.go")
	if !fileExists(moduleFile) {
		t.Fatalf("expected module file at %s", moduleFile)
	}

	rawEnv, err := os.ReadFile(filepath.Join(projectRoot, ".env"))
	if err != nil {
		t.Fatalf("failed to read .env: %v", err)
	}
	if !strings.Contains(string(rawEnv), "AUTH_TOKEN=change-me") {
		t.Fatalf("expected AUTH_TOKEN default in .env, got:\n%s", string(rawEnv))
	}

	rawImports, err := os.ReadFile(filepath.Join(projectRoot, "internal", "modules", "generated_imports.go"))
	if err != nil {
		t.Fatalf("failed to read generated imports: %v", err)
	}
	if !strings.Contains(string(rawImports), `github.com/acme/demo/internal/modules/auth_token`) {
		t.Fatalf("expected generated import, got:\n%s", string(rawImports))
	}
}

func TestRemoveDeletesModuleFilesAndUpdatesImports(t *testing.T) {
	t.Parallel()

	projectRoot := prepareProjectRoot(t)
	sourceRoot := prepareModulesSource(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	addCode := RunWithRoot([]string{"--source", sourceRoot, "--skip-verify", "add", "auth-token"}, projectRoot, &stdout, &stderr)
	if addCode != 0 {
		t.Fatalf("add failed: code=%d stderr=%s", addCode, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()

	removeCode := RunWithRoot([]string{"--skip-verify", "remove", "auth-token"}, projectRoot, &stdout, &stderr)
	if removeCode != 0 {
		t.Fatalf("remove failed: code=%d stderr=%s", removeCode, stderr.String())
	}

	moduleFile := filepath.Join(projectRoot, "internal", "modules", "auth_token", "module.go")
	if fileExists(moduleFile) {
		t.Fatalf("expected module file to be removed: %s", moduleFile)
	}

	rawLock, err := os.ReadFile(filepath.Join(projectRoot, lockFileName))
	if err != nil {
		t.Fatalf("failed to read lockfile: %v", err)
	}
	if strings.Contains(string(rawLock), "auth-token") {
		t.Fatalf("expected auth-token to be removed from lockfile, got:\n%s", string(rawLock))
	}
}

func prepareProjectRoot(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module github.com/acme/demo\n\ngo 1.26.0\n"), 0o644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".env.example"), []byte("PORT=8888\n"), 0o644); err != nil {
		t.Fatalf("failed to write .env.example: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "internal", "modules"), 0o755); err != nil {
		t.Fatalf("failed to prepare modules dir: %v", err)
	}

	return root
}

func prepareModulesSource(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	writeModuleManifest(t, root, "auth-token", `{
  "id": "auth-token",
  "name": "API token auth",
  "description": "Protect API routes with a static token.",
  "package": "internal/modules/auth_token",
  "defaults": {
    "AUTH_TOKEN": "change-me"
  },
  "cleanup_env_keys": ["AUTH_TOKEN"],
  "files": [
    {
      "source": "files/module.go.tmpl",
      "destination": "internal/modules/auth_token/module.go"
    }
  ]
}`)
	writeModuleTemplate(t, root, "auth-token", "files/module.go.tmpl", `package auth_token

const ModuleName = "auth-token"
`)

	writeModuleManifest(t, root, "metrics-prometheus", `{
  "id": "metrics-prometheus",
  "name": "Prometheus metrics",
  "description": "Expose /metrics endpoint.",
  "package": "internal/modules/metrics_prometheus",
  "files": [
    {
      "source": "files/module.go.tmpl",
      "destination": "internal/modules/metrics_prometheus/module.go"
    }
  ]
}`)
	writeModuleTemplate(t, root, "metrics-prometheus", "files/module.go.tmpl", `package metrics_prometheus
`)

	return root
}

func writeModuleManifest(t *testing.T, sourceRoot, moduleID, content string) {
	t.Helper()
	path := filepath.Join(sourceRoot, "modules", moduleID, manifestFileName)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("failed to create manifest dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}
}

func writeModuleTemplate(t *testing.T, sourceRoot, moduleID, relativePath, content string) {
	t.Helper()
	path := filepath.Join(sourceRoot, "modules", moduleID, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("failed to create template dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write template file: %v", err)
	}
}
