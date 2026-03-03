package templatectl

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type runOptions struct {
	Root       string
	Source     string
	NoColor    bool
	NoSpinner  bool
	SkipVerify bool
}

const catalogDividerWidth = 64

// Run executes the template module CLI.
func Run(args []string, stdout, stderr io.Writer) int {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "failed to resolve working directory: %v\n", err)
		return 1
	}

	return RunWithRoot(args, cwd, stdout, stderr)
}

// RunWithRoot executes the CLI against an explicit project root.
func RunWithRoot(args []string, projectRoot string, stdout, stderr io.Writer) int {
	commandArgs, options, err := parseOptions(args, projectRoot)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	ui := newCLIUI(stdout, stderr, options.NoColor, options.NoSpinner)
	if len(commandArgs) == 0 {
		printUsage(stdout)
		return 0
	}

	switch commandArgs[0] {
	case "help", "--help", "-h":
		printUsage(stdout)
		return 0
	case "list":
		return runList(options, ui)
	case "add":
		if len(commandArgs) < 2 {
			ui.failure("missing module target: templatectl add <module-id|index>")
			return 1
		}
		return runAdd(options, commandArgs[1], ui)
	case "remove":
		if len(commandArgs) < 2 {
			ui.failure("missing module target: templatectl remove <module-id|index>")
			return 1
		}
		return runRemove(options, commandArgs[1], ui)
	case "doctor":
		return runDoctor(options, ui)
	default:
		ui.failure("unknown command %q", commandArgs[0])
		printUsage(stderr)
		return 1
	}
}

func runList(options runOptions, ui *cliUI) int {
	lock, err := loadLockFile(options.Root)
	if err != nil {
		ui.failure("failed to read lockfile: %v", err)
		return 1
	}

	modules, sourceDisplay, cleanup, err := resolveListCatalog(options.Root, options.Source)
	if err != nil {
		ui.failure("%v", err)
		return 1
	}
	if cleanup != nil {
		defer cleanup()
	}

	ui.info("modules source: %s", sourceDisplay)
	if len(modules) == 0 {
		ui.warn("no module found in catalog")
		return 0
	}

	installed := make(map[string]InstalledModule, len(lock.Modules))
	installedCount := 0
	for _, module := range lock.Modules {
		installed[module.ID] = module
		installedCount++
	}
	availableCount := len(modules) - installedCount

	ui.print("%s\n", catalogDivider(ui, '='))
	ui.print("%s\n", catalogTitle(ui, " TEMPLATECTL MODULE CATALOG "))
	ui.print("%s\n", catalogDivider(ui, '='))
	ui.print("%s\n", catalogSummary(ui, fmt.Sprintf("Total: %d | Installed: %d | Available: %d", len(modules), installedCount, availableCount)))
	ui.print("%s\n", catalogDivider(ui, '-'))

	for index, module := range modules {
		status := "available"
		if _, ok := installed[module.ID]; ok {
			status = "installed"
		}

		ui.print("%s %s %s\n", catalogIndex(ui, index+1), catalogModuleID(ui, module.ID), catalogStatus(ui, status))
		ui.print("    %s\n", catalogDescription(ui, module.Description))
	}
	ui.print("%s\n", catalogDivider(ui, '-'))
	ui.print("%s\n", catalogTip(ui, "Tip: templatectl add <module-id|index>"))

	return 0
}

func catalogDivider(ui *cliUI, char rune) string {
	return colorizeIfEnabled(ui, colorBlue, strings.Repeat(string(char), catalogDividerWidth))
}

func catalogTitle(ui *cliUI, title string) string {
	return colorizeIfEnabled(ui, colorBold+colorCyan, title)
}

func catalogSummary(ui *cliUI, summary string) string {
	return colorizeIfEnabled(ui, colorCyan, summary)
}

func catalogIndex(ui *cliUI, index int) string {
	return colorizeIfEnabled(ui, colorBold+colorMagenta, fmt.Sprintf("%02d.", index))
}

func catalogModuleID(ui *cliUI, moduleID string) string {
	return colorizeIfEnabled(ui, colorBold, moduleID)
}

func catalogStatus(ui *cliUI, status string) string {
	label := fmt.Sprintf("[%s]", status)
	if status == "installed" {
		return colorizeIfEnabled(ui, colorBold+colorGreen, label)
	}
	return colorizeIfEnabled(ui, colorYellow, label)
}

func catalogDescription(ui *cliUI, description string) string {
	text := strings.TrimSpace(description)
	if text == "" {
		text = "(no description provided)"
	}
	return colorizeIfEnabled(ui, colorGray, text)
}

func catalogTip(ui *cliUI, tip string) string {
	return colorizeIfEnabled(ui, colorBlue, tip)
}

func colorizeIfEnabled(ui *cliUI, colorCode, value string) string {
	if !ui.colors {
		return value
	}
	return colorCode + value + colorReset
}

func runAdd(options runOptions, moduleID string, ui *cliUI) int {
	if warning, err := gitDirtyWarningForAdd(options.Root); err != nil {
		ui.warn("failed to inspect git status; continuing add (%v)", err)
	} else if warning != "" {
		ui.warn("%s", warning)
		proceed, confirmErr := ui.confirmYesNo("Continue module installation? [y/N]")
		if confirmErr != nil {
			ui.failure("failed to read confirmation: %v", confirmErr)
			return 1
		}
		if !proceed {
			ui.warn("add canceled by user")
			return 1
		}
	}

	source, err := resolveModulesSource(options.Root, options.Source)
	if err != nil {
		ui.failure("%v", err)
		return 1
	}
	defer source.close()

	catalog, err := loadCatalog(source.Path)
	if err != nil {
		ui.failure("%v", err)
		return 1
	}

	module, resolvedModuleID, ok, providedIndex, indexOutOfRange := resolveModuleSelection(catalog, moduleID)
	if !ok {
		if providedIndex && indexOutOfRange {
			ui.failure("unknown module index %q (valid range: 1-%d)", strings.TrimSpace(moduleID), len(catalog))
		} else {
			ui.failure("unknown module %q", moduleID)
		}
		return runList(options, ui)
	}
	if strings.TrimSpace(moduleID) != resolvedModuleID {
		ui.info("resolved module index %q -> %s", strings.TrimSpace(moduleID), resolvedModuleID)
	}

	lock, err := loadLockFile(options.Root)
	if err != nil {
		ui.failure("failed to load lockfile: %v", err)
		return 1
	}

	if _, _, exists := lock.findInstalledModule(resolvedModuleID); exists {
		ui.warn("module %q is already installed", resolvedModuleID)
		return 0
	}

	modulePath, err := readGoModulePath(options.Root)
	if err != nil {
		ui.failure("%v", err)
		return 1
	}

	if err := ui.runStep(fmt.Sprintf("installing %s", resolvedModuleID), func() error {
		_, installErr := installModule(options.Root, module, lock, modulePath)
		return installErr
	}); err != nil {
		ui.failure("%v", err)
		return 1
	}

	if err := writeGeneratedImports(options.Root, modulePath, lock); err != nil {
		ui.failure("failed to update generated imports: %v", err)
		return 1
	}

	if err := saveLockFile(options.Root, lock); err != nil {
		ui.failure("failed to save lockfile: %v", err)
		return 1
	}

	if !options.SkipVerify {
		if err := ui.runStep("running go mod tidy and go test ./...", func() error {
			return runGoProjectVerification(options.Root)
		}); err != nil {
			ui.failure("%v", err)
			return 1
		}
	}

	ui.success("module %q installed", resolvedModuleID)
	return 0
}

func resolveModuleSelection(catalog []CatalogModule, selection string) (CatalogModule, string, bool, bool, bool) {
	trimmed := strings.TrimSpace(selection)
	if index, err := strconv.Atoi(trimmed); err == nil {
		if index >= 1 && index <= len(catalog) {
			module := catalog[index-1]
			return module, module.Manifest.ID, true, true, false
		}
		return CatalogModule{}, trimmed, false, true, true
	}

	module, ok := findCatalogModule(catalog, trimmed)
	if !ok {
		return CatalogModule{}, trimmed, false, false, false
	}

	return module, module.Manifest.ID, true, false, false
}

func runRemove(options runOptions, moduleID string, ui *cliUI) int {
	resolvedModuleID := strings.TrimSpace(moduleID)
	if index, err := strconv.Atoi(resolvedModuleID); err == nil {
		modules, _, cleanup, listErr := resolveListCatalog(options.Root, options.Source)
		if cleanup != nil {
			defer cleanup()
		}
		if listErr != nil {
			ui.failure("failed to resolve module index %q: %v", resolvedModuleID, listErr)
			return 1
		}
		if index < 1 || index > len(modules) {
			ui.failure("unknown module index %q (valid range: 1-%d)", resolvedModuleID, len(modules))
			return runList(options, ui)
		}

		resolvedModuleID = modules[index-1].ID
		ui.info("resolved module index %q -> %s", strings.TrimSpace(moduleID), resolvedModuleID)
	}

	lock, err := loadLockFile(options.Root)
	if err != nil {
		ui.failure("failed to load lockfile: %v", err)
		return 1
	}

	installed, _, ok := lock.findInstalledModule(resolvedModuleID)
	if !ok {
		ui.warn("module %q is not installed", resolvedModuleID)
		return 0
	}

	if warning, warnErr := gitDirtyWarningForRemove(options.Root, lock); warnErr != nil {
		ui.warn("failed to inspect git status; continuing remove (%v)", warnErr)
	} else if warning != "" {
		ui.warn("%s", warning)
		proceed, confirmErr := ui.confirmYesNo("Continue module removal? [y/N]")
		if confirmErr != nil {
			ui.failure("failed to read confirmation: %v", confirmErr)
			return 1
		}
		if !proceed {
			ui.warn("remove canceled by user")
			return 1
		}
	}

	modulePath, err := readGoModulePath(options.Root)
	if err != nil {
		ui.failure("%v", err)
		return 1
	}

	if err := ui.runStep(fmt.Sprintf("removing %s", resolvedModuleID), func() error {
		return uninstallModule(options.Root, installed, lock)
	}); err != nil {
		ui.failure("%v", err)
		return 1
	}

	if err := writeGeneratedImports(options.Root, modulePath, lock); err != nil {
		ui.failure("failed to update generated imports: %v", err)
		return 1
	}

	if err := saveLockFile(options.Root, lock); err != nil {
		ui.failure("failed to save lockfile: %v", err)
		return 1
	}

	if !options.SkipVerify {
		if err := ui.runStep("running go mod tidy and go test ./...", func() error {
			return runGoProjectVerification(options.Root)
		}); err != nil {
			ui.failure("%v", err)
			return 1
		}
	}

	ui.success("module %q removed", resolvedModuleID)
	return 0
}

func runDoctor(options runOptions, ui *cliUI) int {
	lock, err := loadLockFile(options.Root)
	if err != nil {
		ui.failure("failed to load lockfile: %v", err)
		return 1
	}

	modulePath, err := readGoModulePath(options.Root)
	if err != nil {
		ui.failure("%v", err)
		return 1
	}

	failures := make([]string, 0)
	for _, module := range lock.Modules {
		for _, relPath := range module.Files {
			absolute, joinErr := safeJoinBase(options.Root, relPath)
			if joinErr != nil {
				failures = append(failures, fmt.Sprintf("%s: %v", module.ID, joinErr))
				continue
			}
			if !fileExists(absolute) {
				failures = append(failures, fmt.Sprintf("%s: missing file %s", module.ID, relPath))
			}
		}
	}

	expectedImports := make([]string, 0, len(lock.Modules))
	for _, module := range lock.Modules {
		expectedImports = append(expectedImports, buildImportPath(modulePath, module.PackagePath))
	}
	sort.Strings(expectedImports)

	currentImportsPath := filepath.Join(options.Root, filepath.FromSlash(generatedImportsPath))
	if fileExists(currentImportsPath) {
		raw, readErr := os.ReadFile(currentImportsPath)
		if readErr != nil {
			failures = append(failures, fmt.Sprintf("failed to read generated imports: %v", readErr))
		} else {
			for _, expected := range expectedImports {
				if !strings.Contains(string(raw), expected) {
					failures = append(failures, fmt.Sprintf("generated imports missing %s", expected))
				}
			}
		}
	} else if len(expectedImports) > 0 {
		failures = append(failures, "generated imports file is missing")
	}

	if len(failures) > 0 {
		for _, issue := range failures {
			ui.failure("%s", issue)
		}
		return 1
	}

	ui.success("doctor passed (%d module(s) installed)", len(lock.Modules))
	return 0
}

func parseOptions(args []string, defaultRoot string) ([]string, runOptions, error) {
	options := runOptions{Root: defaultRoot}
	commandArgs := make([]string, 0, len(args))

	for idx := 0; idx < len(args); idx++ {
		current := args[idx]
		switch current {
		case "--root":
			if idx+1 >= len(args) {
				return nil, runOptions{}, fmt.Errorf("--root expects a path")
			}
			options.Root = args[idx+1]
			idx++
		case "--source":
			if idx+1 >= len(args) {
				return nil, runOptions{}, fmt.Errorf("--source expects a path or repository URL")
			}
			options.Source = args[idx+1]
			idx++
		case "--no-color":
			options.NoColor = true
		case "--no-spinner":
			options.NoSpinner = true
		case "--skip-verify":
			options.SkipVerify = true
		default:
			commandArgs = append(commandArgs, current)
		}
	}

	if !filepath.IsAbs(options.Root) {
		absoluteRoot, err := filepath.Abs(options.Root)
		if err != nil {
			return nil, runOptions{}, fmt.Errorf("failed to resolve root %q: %w", options.Root, err)
		}
		options.Root = absoluteRoot
	}

	return commandArgs, options, nil
}

func printUsage(output io.Writer) {
	fmt.Fprintln(output, "templatectl - install optional modules from an external catalog")
	fmt.Fprintln(output)
	fmt.Fprintln(output, "Usage:")
	fmt.Fprintln(output, "  templatectl [--root <project-path>] [--source <path-or-url>] list")
	fmt.Fprintln(output, "  templatectl [--root <project-path>] [--source <path-or-url>] add <module-id|index>")
	fmt.Fprintln(output, "  templatectl [--root <project-path>] [--source <path-or-url>] remove <module-id|index>")
	fmt.Fprintln(output, "  templatectl [--root <project-path>] doctor")
	fmt.Fprintln(output)
	fmt.Fprintln(output, "Flags:")
	fmt.Fprintln(output, "  --skip-verify   Skip go mod tidy + go test ./... after add/remove")
	fmt.Fprintln(output, "  --no-color      Disable ANSI colors")
	fmt.Fprintln(output, "  --no-spinner    Disable loading spinner")
}
