package templatectl

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const modulesSourceEnv = "TEMPLATECTL_MODULES_SOURCE"

type resolvedSource struct {
	Path    string
	Cleanup func()
}

func (s resolvedSource) close() {
	if s.Cleanup != nil {
		s.Cleanup()
	}
}

func resolveModulesSource(projectRoot, providedSource string) (resolvedSource, error) {
	candidates := sourceCandidates(projectRoot, providedSource)
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}

		if directoryExists(candidate) {
			return resolvedSource{Path: candidate}, nil
		}

		if !looksLikeRemoteRepo(candidate) {
			continue
		}

		tempDir, err := os.MkdirTemp("", "templatectl-modules-*")
		if err != nil {
			return resolvedSource{}, fmt.Errorf("failed to prepare temp dir for modules source: %w", err)
		}

		if err := gitClone(candidate, tempDir); err != nil {
			_ = os.RemoveAll(tempDir)
			return resolvedSource{}, err
		}

		return resolvedSource{
			Path: tempDir,
			Cleanup: func() {
				_ = os.RemoveAll(tempDir)
			},
		}, nil
	}

	return resolvedSource{}, fmt.Errorf("modules source not found (use --source or %s)", modulesSourceEnv)
}

func sourceCandidates(projectRoot, providedSource string) []string {
	if strings.TrimSpace(providedSource) != "" {
		source := strings.TrimSpace(providedSource)
		if !filepath.IsAbs(source) && !looksLikeRemoteRepo(source) {
			source = filepath.Join(projectRoot, source)
		}
		return []string{source}
	}

	if envSource := strings.TrimSpace(os.Getenv(modulesSourceEnv)); envSource != "" {
		if !filepath.IsAbs(envSource) && !looksLikeRemoteRepo(envSource) {
			envSource = filepath.Join(projectRoot, envSource)
		}
		return []string{envSource}
	}

	return []string{
		filepath.Join(projectRoot, "huma_golang_api_template_modules"),
		filepath.Join(filepath.Dir(projectRoot), "huma_golang_api_template_modules"),
	}
}

func looksLikeRemoteRepo(candidate string) bool {
	return strings.HasPrefix(candidate, "https://") ||
		strings.HasPrefix(candidate, "http://") ||
		strings.HasPrefix(candidate, "ssh://") ||
		strings.HasPrefix(candidate, "git@") ||
		strings.HasSuffix(candidate, ".git")
}

func gitClone(source, destination string) error {
	cmd := exec.Command("git", "clone", "--depth", "1", source, destination)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone modules source %q: %w (%s)", source, err, strings.TrimSpace(output.String()))
	}

	return nil
}
