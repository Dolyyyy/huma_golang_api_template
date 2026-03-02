package templatectl

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const (
	lockFileName     = ".templatectl.lock.json"
	lockFileVersion1 = 1
)

// InstalledModule tracks one applied module in the local lockfile.
type InstalledModule struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	PackagePath string   `json:"package"`
	Files       []string `json:"files"`
	EnvKeys     []string `json:"env_keys,omitempty"`
	InstalledAt string   `json:"installed_at"`
}

// LockFile tracks module state in a project.
type LockFile struct {
	Version int               `json:"version"`
	Modules []InstalledModule `json:"modules"`
}

func loadLockFile(projectRoot string) (*LockFile, error) {
	path := filepath.Join(projectRoot, lockFileName)
	if !fileExists(path) {
		return &LockFile{
			Version: lockFileVersion1,
			Modules: []InstalledModule{},
		}, nil
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var lock LockFile
	if err := json.Unmarshal(raw, &lock); err != nil {
		return nil, err
	}

	if lock.Version == 0 {
		lock.Version = lockFileVersion1
	}
	if lock.Modules == nil {
		lock.Modules = []InstalledModule{}
	}

	sort.Slice(lock.Modules, func(i, j int) bool {
		return lock.Modules[i].ID < lock.Modules[j].ID
	})

	return &lock, nil
}

func saveLockFile(projectRoot string, lock *LockFile) error {
	lock.Version = lockFileVersion1
	sort.Slice(lock.Modules, func(i, j int) bool {
		return lock.Modules[i].ID < lock.Modules[j].ID
	})

	raw, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')

	path := filepath.Join(projectRoot, lockFileName)
	return os.WriteFile(path, raw, 0o644)
}

func (f *LockFile) findInstalledModule(moduleID string) (InstalledModule, int, bool) {
	for idx, module := range f.Modules {
		if module.ID == moduleID {
			return module, idx, true
		}
	}
	return InstalledModule{}, -1, false
}

func (f *LockFile) upsert(module InstalledModule) {
	if module.InstalledAt == "" {
		module.InstalledAt = time.Now().Format(time.RFC3339)
	}

	for idx, current := range f.Modules {
		if current.ID == module.ID {
			f.Modules[idx] = module
			return
		}
	}

	f.Modules = append(f.Modules, module)
}

func (f *LockFile) remove(moduleID string) bool {
	_, idx, ok := f.findInstalledModule(moduleID)
	if !ok {
		return false
	}

	f.Modules = append(f.Modules[:idx], f.Modules[idx+1:]...)
	return true
}
