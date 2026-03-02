package templatectl

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

const modulesIndexFileName = "modules.json"

type listedModule struct {
	ID          string
	Description string
}

type moduleIndexEntry struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func resolveListCatalog(projectRoot, providedSource string) ([]listedModule, string, func(), error) {
	candidates := sourceCandidates(projectRoot, providedSource)
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}

		if directoryExists(candidate) {
			catalog, err := loadCatalog(candidate)
			if err != nil {
				return nil, "", nil, err
			}
			return listFromCatalog(catalog), candidate, nil, nil
		}

		if !looksLikeRemoteRepo(candidate) {
			continue
		}

		modules, err := loadRemoteModulesIndex(candidate)
		if err == nil {
			return modules, candidate, nil, nil
		}

		// Keep compatibility for remote repositories without modules.json.
		tempDir, mkdirErr := os.MkdirTemp("", "templatectl-modules-*")
		if mkdirErr != nil {
			return nil, "", nil, fmt.Errorf("failed to prepare temp dir for modules source: %w", mkdirErr)
		}

		if cloneErr := gitClone(candidate, tempDir); cloneErr != nil {
			_ = os.RemoveAll(tempDir)
			return nil, "", nil, fmt.Errorf("failed to fetch modules index and clone source %q: %v; %v", candidate, err, cloneErr)
		}

		catalog, catalogErr := loadCatalog(tempDir)
		if catalogErr != nil {
			_ = os.RemoveAll(tempDir)
			return nil, "", nil, catalogErr
		}

		return listFromCatalog(catalog), candidate, func() {
			_ = os.RemoveAll(tempDir)
		}, nil
	}

	return nil, "", nil, fmt.Errorf("modules source not found (use --source or %s)", modulesSourceEnv)
}

func listFromCatalog(catalog []CatalogModule) []listedModule {
	modules := make([]listedModule, 0, len(catalog))
	for _, module := range catalog {
		modules = append(modules, listedModule{
			ID:          module.Manifest.ID,
			Description: module.Manifest.Description,
		})
	}
	return modules
}

func loadRemoteModulesIndex(source string) ([]listedModule, error) {
	candidates := modulesIndexURLCandidates(source)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("unsupported remote modules source %q for %s lookup", source, modulesIndexFileName)
	}

	var lastErr error
	for _, indexURL := range candidates {
		modules, err := fetchModulesIndex(indexURL)
		if err == nil {
			return modules, nil
		}
		lastErr = err
	}

	return nil, fmt.Errorf("failed to fetch %s from %q: %v", modulesIndexFileName, source, lastErr)
}

func modulesIndexURLCandidates(source string) []string {
	trimmed := strings.TrimSpace(source)
	if trimmed == "" {
		return nil
	}

	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		if strings.HasSuffix(trimmed, "/"+modulesIndexFileName) || strings.HasSuffix(trimmed, modulesIndexFileName) {
			return []string{trimmed}
		}

		if owner, repo, ok := parseGitHubOwnerRepo(trimmed); ok {
			return []string{
				fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/main/%s", owner, repo, modulesIndexFileName),
				fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/master/%s", owner, repo, modulesIndexFileName),
			}
		}

		return []string{strings.TrimRight(trimmed, "/") + "/" + modulesIndexFileName}
	}

	if owner, repo, ok := parseGitHubOwnerRepo(trimmed); ok {
		return []string{
			fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/main/%s", owner, repo, modulesIndexFileName),
			fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/master/%s", owner, repo, modulesIndexFileName),
		}
	}

	return nil
}

func parseGitHubOwnerRepo(source string) (string, string, bool) {
	candidate := strings.TrimSpace(source)
	if candidate == "" {
		return "", "", false
	}

	if strings.HasPrefix(candidate, "git@github.com:") {
		candidate = strings.TrimPrefix(candidate, "git@github.com:")
	} else if strings.HasPrefix(candidate, "ssh://git@github.com/") {
		candidate = strings.TrimPrefix(candidate, "ssh://git@github.com/")
	} else if strings.HasPrefix(candidate, "https://github.com/") {
		candidate = strings.TrimPrefix(candidate, "https://github.com/")
	} else if strings.HasPrefix(candidate, "http://github.com/") {
		candidate = strings.TrimPrefix(candidate, "http://github.com/")
	} else {
		return "", "", false
	}

	candidate = strings.TrimSuffix(candidate, ".git")
	candidate = strings.Trim(candidate, "/")
	parts := strings.Split(candidate, "/")
	if len(parts) < 2 {
		return "", "", false
	}

	if len(parts) >= 4 && parts[2] == "tree" {
		return parts[0], parts[1], true
	}

	return parts[0], parts[1], true
}

func fetchModulesIndex(indexURL string) ([]listedModule, error) {
	client := http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodGet, indexURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build request %q: %w", indexURL, err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed for %q: %w", indexURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("unexpected HTTP %d for %q (%s)", resp.StatusCode, indexURL, strings.TrimSpace(string(body)))
	}

	var entries []moduleIndexEntry
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&entries); err != nil {
		return nil, fmt.Errorf("invalid %s at %q: %w", modulesIndexFileName, indexURL, err)
	}

	modules := make([]listedModule, 0, len(entries))
	for _, entry := range entries {
		if strings.TrimSpace(entry.ID) == "" {
			return nil, fmt.Errorf("invalid %s at %q: id is required", modulesIndexFileName, indexURL)
		}

		modules = append(modules, listedModule{
			ID:          strings.TrimSpace(entry.ID),
			Description: strings.TrimSpace(entry.Description),
		})
	}

	sort.Slice(modules, func(i, j int) bool {
		return modules[i].ID < modules[j].ID
	})

	return modules, nil
}
