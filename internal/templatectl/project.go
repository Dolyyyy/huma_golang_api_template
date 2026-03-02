package templatectl

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func readGoModulePath(projectRoot string) (string, error) {
	path := filepath.Join(projectRoot, "go.mod")
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read go.mod: %w", err)
	}

	for _, line := range strings.Split(string(raw), "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "module ") {
			continue
		}

		modulePath := strings.TrimSpace(strings.TrimPrefix(trimmed, "module "))
		if modulePath != "" {
			return modulePath, nil
		}
	}

	return "", fmt.Errorf("failed to resolve module path in go.mod")
}

func ensureGitClean(projectRoot string) error {
	gitDir := filepath.Join(projectRoot, ".git")
	if !directoryExists(gitDir) {
		return nil
	}

	cmd := exec.Command("git", "-C", projectRoot, "status", "--porcelain")
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to inspect git status: %w (%s)", err, strings.TrimSpace(output.String()))
	}

	if strings.TrimSpace(output.String()) != "" {
		return fmt.Errorf("git working tree is not clean (commit/stash before add/remove)")
	}

	return nil
}

func directoryExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
