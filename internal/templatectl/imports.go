package templatectl

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const generatedImportsPath = "internal/modules/generated_imports.go"

func writeGeneratedImports(projectRoot, modulePath string, lock *LockFile) error {
	imports := make([]string, 0, len(lock.Modules))
	for _, module := range lock.Modules {
		importPath := buildImportPath(modulePath, module.PackagePath)
		imports = append(imports, importPath)
	}

	sort.Strings(imports)

	destination := filepath.Join(projectRoot, filepath.FromSlash(generatedImportsPath))
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}

	content := renderGeneratedImports(imports)
	return os.WriteFile(destination, []byte(content), 0o644)
}

func buildImportPath(projectModulePath, packagePath string) string {
	cleaned := strings.Trim(strings.TrimSpace(packagePath), "/")
	return fmt.Sprintf("%s/%s", strings.TrimRight(projectModulePath, "/"), cleaned)
}

func renderGeneratedImports(imports []string) string {
	if len(imports) == 0 {
		return "package modules\n\n// No optional modules installed.\n"
	}

	lines := make([]string, 0, len(imports)+4)
	lines = append(lines, "package modules", "", "import (")
	for _, importPath := range imports {
		lines = append(lines, fmt.Sprintf("\t_ %q", importPath))
	}
	lines = append(lines, ")", "")

	return strings.Join(lines, "\n")
}
