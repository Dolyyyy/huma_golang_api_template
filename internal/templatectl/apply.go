package templatectl

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"
)

type templateData struct {
	ProjectModulePath string
}

func installModule(projectRoot string, module CatalogModule, lock *LockFile, modulePath string) (InstalledModule, error) {
	data := templateData{ProjectModulePath: modulePath}
	files, err := materializeModuleFiles(projectRoot, module, data)
	if err != nil {
		return InstalledModule{}, err
	}

	envPath, err := ensureEnvFile(projectRoot)
	if err != nil {
		return InstalledModule{}, fmt.Errorf("failed to prepare .env: %w", err)
	}

	env, err := loadEnvFile(envPath)
	if err != nil {
		return InstalledModule{}, fmt.Errorf("failed to load .env: %w", err)
	}

	moduleEnvKeys := manifestEnvKeys(module.Manifest)
	env.UpsertModuleSection(module.Manifest.ID, moduleEnvKeys, module.Manifest.Defaults)
	if err := env.Save(); err != nil {
		return InstalledModule{}, fmt.Errorf("failed to save .env: %w", err)
	}

	installed := InstalledModule{
		ID:          module.Manifest.ID,
		Name:        module.Manifest.Name,
		PackagePath: module.Manifest.PackagePath,
		Files:       files,
		EnvKeys:     append([]string(nil), moduleEnvKeys...),
		InstalledAt: time.Now().Format(time.RFC3339),
	}

	lock.upsert(installed)
	return installed, nil
}

func uninstallModule(projectRoot string, module InstalledModule, lock *LockFile) error {
	for _, relPath := range module.Files {
		absolute, err := safeJoinBase(projectRoot, relPath)
		if err != nil {
			return err
		}

		if fileExists(absolute) {
			if err := os.Remove(absolute); err != nil {
				return fmt.Errorf("failed to remove %s: %w", relPath, err)
			}
		}

		removeEmptyParents(projectRoot, filepath.Dir(absolute))
	}

	envPath := filepath.Join(projectRoot, envFileName)
	if fileExists(envPath) {
		env, err := loadEnvFile(envPath)
		if err != nil {
			return fmt.Errorf("failed to load .env: %w", err)
		}
		env.RemoveModuleSection(module.ID, module.EnvKeys)
		if err := env.Save(); err != nil {
			return fmt.Errorf("failed to save .env: %w", err)
		}
	}

	lock.remove(module.ID)
	return nil
}

func materializeModuleFiles(projectRoot string, module CatalogModule, data templateData) ([]string, error) {
	created := make([]string, 0, len(module.Manifest.Files))

	for _, file := range module.Manifest.Files {
		sourcePath, err := safeJoinBase(module.Dir, file.Source)
		if err != nil {
			return nil, fmt.Errorf("invalid source path for %s: %w", module.Manifest.ID, err)
		}

		rawSource, err := os.ReadFile(sourcePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read module source file %s: %w", sourcePath, err)
		}

		rendered, err := renderTemplate(sourcePath, string(rawSource), data)
		if err != nil {
			return nil, err
		}

		destinationPath, err := safeJoinBase(projectRoot, file.Destination)
		if err != nil {
			return nil, err
		}

		if err := os.MkdirAll(filepath.Dir(destinationPath), 0o755); err != nil {
			return nil, fmt.Errorf("failed to create directory for %s: %w", destinationPath, err)
		}

		if fileExists(destinationPath) {
			existing, err := os.ReadFile(destinationPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read destination file %s: %w", destinationPath, err)
			}
			if !bytes.Equal(existing, rendered) {
				return nil, fmt.Errorf("destination file already exists with different content: %s", file.Destination)
			}
		} else {
			if err := os.WriteFile(destinationPath, rendered, 0o644); err != nil {
				return nil, fmt.Errorf("failed to write %s: %w", destinationPath, err)
			}
		}

		created = append(created, filepath.ToSlash(filepath.Clean(file.Destination)))
	}

	sort.Strings(created)
	return created, nil
}

func safeJoinBase(basePath, relativePath string) (string, error) {
	cleaned := filepath.Clean(filepath.FromSlash(strings.TrimSpace(relativePath)))
	if cleaned == "." || cleaned == "" {
		return "", fmt.Errorf("invalid path %q", relativePath)
	}
	if filepath.IsAbs(cleaned) {
		return "", fmt.Errorf("absolute path is not allowed: %q", relativePath)
	}
	if strings.HasPrefix(cleaned, "..") {
		return "", fmt.Errorf("path traversal is not allowed: %q", relativePath)
	}

	return filepath.Join(basePath, cleaned), nil
}

func removeEmptyParents(projectRoot, startDir string) {
	current := startDir
	for {
		if current == "" || current == "." {
			return
		}

		relative, err := filepath.Rel(projectRoot, current)
		if err != nil || strings.HasPrefix(relative, "..") {
			return
		}
		if relative == "." {
			return
		}

		if err := os.Remove(current); err != nil {
			return
		}

		current = filepath.Dir(current)
	}
}

func renderTemplate(sourcePath, source string, data templateData) ([]byte, error) {
	tpl, err := template.New(filepath.Base(sourcePath)).Option("missingkey=error").Parse(source)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template %s: %w", sourcePath, err)
	}

	var buffer bytes.Buffer
	if err := tpl.Execute(&buffer, data); err != nil {
		return nil, fmt.Errorf("failed to execute template %s: %w", sourcePath, err)
	}

	return buffer.Bytes(), nil
}

func runGoProjectVerification(projectRoot string) error {
	if err := runCommand(projectRoot, "go", "mod", "tidy"); err != nil {
		return err
	}
	if err := runCommand(projectRoot, "go", "test", "./..."); err != nil {
		return err
	}
	return nil
}

func runCommand(projectRoot string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = projectRoot

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s failed: %w (%s)", name, strings.Join(args, " "), err, strings.TrimSpace(output.String()))
	}

	return nil
}

func manifestEnvKeys(manifest ModuleManifest) []string {
	return normalizeModuleKeys(manifest.CleanupEnvKeys, manifest.Defaults)
}
