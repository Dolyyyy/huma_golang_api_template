package templatectl

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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

	dirtyPaths, err := gitDirtyPaths(projectRoot)
	if err != nil {
		return err
	}
	if len(dirtyPaths) > 0 {
		return fmt.Errorf("git working tree is not clean (commit/stash before add/remove)")
	}

	return nil
}

func ensureGitSafeForRemove(projectRoot string, lock *LockFile) error {
	gitDir := filepath.Join(projectRoot, ".git")
	if !directoryExists(gitDir) {
		return nil
	}

	dirtyPaths, err := gitDirtyPaths(projectRoot)
	if err != nil {
		return err
	}
	if len(dirtyPaths) == 0 {
		return nil
	}

	allowed := allowedTemplatectlPaths(lock)
	unexpected := make([]string, 0)
	for _, path := range dirtyPaths {
		if _, ok := allowed[path]; !ok {
			unexpected = append(unexpected, path)
		}
	}
	if len(unexpected) == 0 {
		return nil
	}

	sort.Strings(unexpected)
	preview := unexpected
	remaining := 0
	if len(preview) > 6 {
		preview = preview[:6]
		remaining = len(unexpected) - len(preview)
	}

	message := fmt.Sprintf("git working tree has non-templatectl changes; commit/stash before remove (%s", strings.Join(preview, ", "))
	if remaining > 0 {
		message += fmt.Sprintf(", +%d more", remaining)
	}
	message += ")"

	return fmt.Errorf("%s", message)
}

func allowedTemplatectlPaths(lock *LockFile) map[string]struct{} {
	allowed := map[string]struct{}{
		normalizeRepoPath(lockFileName):         {},
		normalizeRepoPath(generatedImportsPath): {},
		normalizeRepoPath(envFileName):          {},
		normalizeRepoPath("go.mod"):             {},
		normalizeRepoPath("go.sum"):             {},
	}

	for _, module := range lock.Modules {
		for _, relPath := range module.Files {
			normalized := normalizeRepoPath(relPath)
			if normalized == "" {
				continue
			}
			allowed[normalized] = struct{}{}
		}
	}

	return allowed
}

func gitDirtyPaths(projectRoot string) ([]string, error) {
	cmd := exec.Command("git", "-C", projectRoot, "status", "--porcelain=1", "-z", "--untracked-files=all")
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to inspect git status: %w (%s)", err, strings.TrimSpace(output.String()))
	}

	raw := output.Bytes()
	if len(raw) == 0 {
		return nil, nil
	}

	paths := make(map[string]struct{})
	index := 0
	for index < len(raw) {
		end := bytes.IndexByte(raw[index:], 0)
		if end < 0 {
			break
		}

		entry := raw[index : index+end]
		index += end + 1
		if len(entry) < 4 {
			continue
		}

		status := entry[:2]
		firstPath := normalizeRepoPath(string(entry[3:]))
		if firstPath != "" {
			paths[firstPath] = struct{}{}
		}

		if status[0] == 'R' || status[0] == 'C' || status[1] == 'R' || status[1] == 'C' {
			nextEnd := bytes.IndexByte(raw[index:], 0)
			if nextEnd < 0 {
				break
			}

			secondPath := normalizeRepoPath(string(raw[index : index+nextEnd]))
			index += nextEnd + 1
			if secondPath != "" {
				paths[secondPath] = struct{}{}
			}
		}
	}

	result := make([]string, 0, len(paths))
	for path := range paths {
		result = append(result, path)
	}
	sort.Strings(result)
	return result, nil
}

func normalizeRepoPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}

	cleaned := filepath.ToSlash(filepath.Clean(trimmed))
	if cleaned == "." {
		return ""
	}

	return strings.TrimPrefix(cleaned, "./")
}

func directoryExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
