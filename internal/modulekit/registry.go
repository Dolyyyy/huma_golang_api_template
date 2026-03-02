package modulekit

import (
	"fmt"
	"net/http"
	"sort"
	"sync"

	"github.com/go-chi/chi/v5"
)

// Definition describes one runtime module that can hook into the API server.
type Definition struct {
	ID          string
	Name        string
	Description string
	Validate    func() error
	Middleware  func(http.Handler) http.Handler
	Routes      func(chi.Router)
}

var (
	registryMu sync.RWMutex
	registry   = map[string]Definition{}
)

// Register stores a module definition globally. Panics on invalid duplicates.
func Register(def Definition) {
	if def.ID == "" {
		panic("module id is required")
	}

	registryMu.Lock()
	defer registryMu.Unlock()

	if _, exists := registry[def.ID]; exists {
		panic(fmt.Sprintf("module %q already registered", def.ID))
	}

	registry[def.ID] = def
}

// All returns registered modules in stable ID order.
func All() []Definition {
	registryMu.RLock()
	defer registryMu.RUnlock()

	out := make([]Definition, 0, len(registry))
	for _, module := range registry {
		out = append(out, module)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})

	return out
}

// IDs returns registered module IDs in stable order.
func IDs() []string {
	modules := All()
	ids := make([]string, 0, len(modules))
	for _, module := range modules {
		ids = append(ids, module.ID)
	}
	return ids
}

// ValidateAll executes module-level validation hooks.
func ValidateAll() error {
	for _, module := range All() {
		if module.Validate == nil {
			continue
		}

		if err := module.Validate(); err != nil {
			return fmt.Errorf("%s: %w", module.ID, err)
		}
	}

	return nil
}

// ApplyMiddlewares registers all module middlewares onto the provided router.
func ApplyMiddlewares(router chi.Router) {
	for _, module := range All() {
		if module.Middleware == nil {
			continue
		}
		router.Use(module.Middleware)
	}
}

// RegisterRoutes attaches all module routes onto the provided router.
func RegisterRoutes(router chi.Router) {
	for _, module := range All() {
		if module.Routes == nil {
			continue
		}
		module.Routes(router)
	}
}
