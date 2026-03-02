package templatectl

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

const (
	modulesDirectoryName = "modules"
	manifestFileName     = "module.json"
)

// ModuleFile describes one source file to copy into the target project.
type ModuleFile struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
}

// ModuleManifest is the declarative metadata for one installable module pack.
type ModuleManifest struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Description    string            `json:"description"`
	PackagePath    string            `json:"package"`
	Defaults       map[string]string `json:"defaults,omitempty"`
	CleanupEnvKeys []string          `json:"cleanup_env_keys,omitempty"`
	Files          []ModuleFile      `json:"files"`
}

// CatalogModule links one manifest to its source directory.
type CatalogModule struct {
	Manifest ModuleManifest
	Dir      string
}

func loadCatalog(sourceRoot string) ([]CatalogModule, error) {
	modulesRoot := filepath.Join(sourceRoot, modulesDirectoryName)
	entries, err := os.ReadDir(modulesRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to read modules catalog at %s: %w", modulesRoot, err)
	}

	catalog := make([]CatalogModule, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		moduleDir := filepath.Join(modulesRoot, entry.Name())
		manifestPath := filepath.Join(moduleDir, manifestFileName)
		manifest, err := loadManifest(manifestPath)
		if err != nil {
			return nil, err
		}

		catalog = append(catalog, CatalogModule{
			Manifest: manifest,
			Dir:      moduleDir,
		})
	}

	sort.Slice(catalog, func(i, j int) bool {
		return catalog[i].Manifest.ID < catalog[j].Manifest.ID
	})

	return catalog, nil
}

func loadManifest(path string) (ModuleManifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ModuleManifest{}, fmt.Errorf("failed to read manifest %s: %w", path, err)
	}

	var manifest ModuleManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return ModuleManifest{}, fmt.Errorf("invalid manifest %s: %w", path, err)
	}

	if manifest.ID == "" {
		return ModuleManifest{}, fmt.Errorf("invalid manifest %s: id is required", path)
	}
	if manifest.Name == "" {
		return ModuleManifest{}, fmt.Errorf("invalid manifest %s: name is required", path)
	}
	if manifest.PackagePath == "" {
		return ModuleManifest{}, fmt.Errorf("invalid manifest %s: package is required", path)
	}
	if len(manifest.Files) == 0 {
		return ModuleManifest{}, fmt.Errorf("invalid manifest %s: files list cannot be empty", path)
	}

	return manifest, nil
}

func findCatalogModule(catalog []CatalogModule, moduleID string) (CatalogModule, bool) {
	for _, module := range catalog {
		if module.Manifest.ID == moduleID {
			return module, true
		}
	}

	return CatalogModule{}, false
}
