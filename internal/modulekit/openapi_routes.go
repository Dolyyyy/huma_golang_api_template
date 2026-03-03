package modulekit

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
)

// RegisterLegacyRouteDocs adds OpenAPI operations for modules that still only
// register chi routes (without Huma operations).
func RegisterLegacyRouteDocs(api huma.API) {
	if api == nil || api.OpenAPI() == nil {
		return
	}

	spec := api.OpenAPI()
	if spec.Paths == nil {
		spec.Paths = map[string]*huma.PathItem{}
	}

	for _, module := range All() {
		if module.Routes == nil || module.HumaRoutes != nil {
			continue
		}

		tmp := chi.NewRouter()
		module.Routes(tmp)

		walkChiRoutes("", tmp.Routes(), func(method, path string) {
			normalizedMethod, ok := normalizeMethod(method)
			if !ok {
				return
			}

			normalizedPath := normalizePath(path)
			if hasOperation(spec, normalizedMethod, normalizedPath) {
				return
			}

			spec.AddOperation(&huma.Operation{
				Method:      normalizedMethod,
				Path:        normalizedPath,
				OperationID: legacyOperationID(module.ID, normalizedMethod, normalizedPath),
				Summary:     fmt.Sprintf("%s endpoint", module.Name),
				Tags:        []string{module.ID},
				Responses: map[string]*huma.Response{
					"200": {
						Description: "OK",
					},
				},
			})
		})
	}
}

func walkChiRoutes(prefix string, routes []chi.Route, visit func(method, path string)) {
	for _, route := range routes {
		fullPath := joinRoutePattern(prefix, route.Pattern)

		for method := range route.Handlers {
			visit(method, fullPath)
		}

		if route.SubRoutes != nil {
			walkChiRoutes(fullPath, route.SubRoutes.Routes(), visit)
		}
	}
}

func joinRoutePattern(prefix, pattern string) string {
	prefix = strings.TrimSpace(prefix)
	pattern = strings.TrimSpace(pattern)

	prefix = strings.TrimSuffix(prefix, "/*")
	pattern = strings.TrimSuffix(pattern, "/*")

	switch {
	case prefix == "" && pattern == "":
		return "/"
	case prefix == "":
		return ensureLeadingSlash(pattern)
	case pattern == "":
		return ensureLeadingSlash(prefix)
	}

	left := strings.TrimSuffix(prefix, "/")
	right := strings.TrimPrefix(pattern, "/")
	return ensureLeadingSlash(left + "/" + right)
}

func ensureLeadingSlash(value string) string {
	if value == "" {
		return "/"
	}
	if strings.HasPrefix(value, "/") {
		return value
	}
	return "/" + value
}

func normalizePath(path string) string {
	path = ensureLeadingSlash(strings.TrimSpace(path))
	if path == "" {
		return "/"
	}
	return path
}

func normalizeMethod(raw string) (string, bool) {
	method := strings.ToUpper(strings.TrimSpace(raw))
	if method == "" || method == "*" {
		return http.MethodGet, true
	}

	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch,
		http.MethodDelete, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return method, true
	default:
		return "", false
	}
}

func legacyOperationID(moduleID, method, path string) string {
	parts := []string{
		strings.ToLower(strings.TrimSpace(moduleID)),
		strings.ToLower(strings.TrimSpace(method)),
	}

	trimmedPath := strings.Trim(path, "/")
	if trimmedPath != "" {
		replacer := strings.NewReplacer("/", "-", "{", "", "}", "", ".", "-", ":", "-")
		parts = append(parts, replacer.Replace(strings.ToLower(trimmedPath)))
	}

	return strings.Join(parts, "-")
}

func hasOperation(spec *huma.OpenAPI, method, path string) bool {
	pathItem := spec.Paths[path]
	if pathItem == nil {
		return false
	}

	switch method {
	case http.MethodGet:
		return pathItem.Get != nil
	case http.MethodPost:
		return pathItem.Post != nil
	case http.MethodPut:
		return pathItem.Put != nil
	case http.MethodPatch:
		return pathItem.Patch != nil
	case http.MethodDelete:
		return pathItem.Delete != nil
	case http.MethodHead:
		return pathItem.Head != nil
	case http.MethodOptions:
		return pathItem.Options != nil
	case http.MethodTrace:
		return pathItem.Trace != nil
	default:
		return false
	}
}
