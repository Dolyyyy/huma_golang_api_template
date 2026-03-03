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

const (
	envFileName        = ".env"
	envExampleFileName = ".env.example"
)

type lineKind string

const (
	lineKindRaw lineKind = "raw"
	lineKindKV  lineKind = "kv"
)

type envLine struct {
	kind  lineKind
	raw   string
	key   string
	value string
}

// EnvFile is a small .env editor that preserves comments and unrelated lines.
type EnvFile struct {
	path  string
	lines []envLine
	index map[string]int
}

func ensureEnvFile(projectRoot string) (string, error) {
	path := filepath.Join(projectRoot, envFileName)
	if fileExists(path) {
		return path, nil
	}

	examplePath := filepath.Join(projectRoot, envExampleFileName)
	if fileExists(examplePath) {
		if err := copyFile(examplePath, path); err != nil {
			return "", err
		}
		return path, nil
	}

	if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
		return "", err
	}

	return path, nil
}

func loadEnvFile(path string) (*EnvFile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	text := string(raw)
	entries := strings.Split(text, "\n")
	lines := make([]envLine, 0, len(entries))
	index := make(map[string]int)

	for _, entry := range entries {
		parsed := parseLine(entry)
		if parsed.kind == lineKindKV {
			index[parsed.key] = len(lines)
		}
		lines = append(lines, parsed)
	}

	return &EnvFile{
		path:  path,
		lines: lines,
		index: index,
	}, nil
}

func (f *EnvFile) Save() error {
	output := make([]string, 0, len(f.lines))
	previousBlank := false

	for _, item := range f.lines {
		line := item.raw
		switch item.kind {
		case lineKindKV:
			line = fmt.Sprintf("%s=%s", item.key, encodeValue(item.value))
		}

		if strings.TrimSpace(line) == "" {
			if len(output) == 0 || previousBlank {
				continue
			}
			previousBlank = true
			output = append(output, "")
			continue
		}

		previousBlank = false
		output = append(output, line)
	}

	for len(output) > 0 && strings.TrimSpace(output[len(output)-1]) == "" {
		output = output[:len(output)-1]
	}

	content := strings.Join(output, "\n")
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	return os.WriteFile(f.path, []byte(content), 0o644)
}

func (f *EnvFile) Set(key, value string) {
	normalized := strings.TrimSpace(key)
	if normalized == "" {
		return
	}

	if idx, ok := f.index[normalized]; ok {
		f.lines[idx].kind = lineKindKV
		f.lines[idx].key = normalized
		f.lines[idx].value = value
		return
	}

	f.lines = append(f.lines, envLine{
		kind:  lineKindKV,
		key:   normalized,
		value: value,
	})
	f.index[normalized] = len(f.lines) - 1
}

func (f *EnvFile) SetIfMissing(key, value string) {
	current, exists := f.Get(key)
	if exists && strings.TrimSpace(current) != "" {
		return
	}

	f.Set(key, value)
}

func (f *EnvFile) Unset(key string) {
	normalized := strings.TrimSpace(key)
	if normalized == "" {
		return
	}

	idx, ok := f.index[normalized]
	if !ok {
		return
	}

	f.lines = append(f.lines[:idx], f.lines[idx+1:]...)
	f.rebuildIndex()
}

func (f *EnvFile) UpsertModuleSection(moduleID string, orderedKeys []string, defaults map[string]string) {
	normalizedModuleID := strings.TrimSpace(moduleID)
	if normalizedModuleID == "" {
		return
	}

	keys := normalizeModuleKeys(orderedKeys, defaults)
	if len(keys) == 0 {
		return
	}

	values := make(map[string]string, len(keys))
	for _, key := range keys {
		existing, ok := f.Get(key)
		if ok && strings.TrimSpace(existing) != "" {
			values[key] = existing
			continue
		}
		values[key] = defaults[key]
	}

	header := moduleSectionHeader(normalizedModuleID)
	f.removeKeys(keys)
	f.removeRawLine(header)

	if !f.endsWithBlankOrEmpty() {
		f.lines = append(f.lines, envLine{kind: lineKindRaw, raw: ""})
	}
	f.lines = append(f.lines, envLine{kind: lineKindRaw, raw: header})
	for _, key := range keys {
		f.lines = append(f.lines, envLine{
			kind:  lineKindKV,
			key:   key,
			value: values[key],
		})
	}
	f.rebuildIndex()
}

func (f *EnvFile) RemoveModuleSection(moduleID string, keys []string) {
	normalizedModuleID := strings.TrimSpace(moduleID)
	if normalizedModuleID == "" {
		return
	}

	f.removeKeys(keys)
	f.removeRawLine(moduleSectionHeader(normalizedModuleID))
	f.rebuildIndex()
}

func (f *EnvFile) Get(key string) (string, bool) {
	idx, ok := f.index[key]
	if !ok {
		return "", false
	}

	return f.lines[idx].value, true
}

func (f *EnvFile) Values() map[string]string {
	values := make(map[string]string)
	for _, item := range f.lines {
		if item.kind != lineKindKV {
			continue
		}
		values[item.key] = item.value
	}
	return values
}

func (f *EnvFile) rebuildIndex() {
	index := make(map[string]int)
	for idx, item := range f.lines {
		if item.kind != lineKindKV {
			continue
		}
		index[item.key] = idx
	}
	f.index = index
}

func (f *EnvFile) removeKeys(keys []string) {
	if len(keys) == 0 {
		return
	}

	keySet := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		normalized := strings.TrimSpace(key)
		if normalized == "" {
			continue
		}
		keySet[normalized] = struct{}{}
	}
	if len(keySet) == 0 {
		return
	}

	filtered := make([]envLine, 0, len(f.lines))
	for _, line := range f.lines {
		if line.kind == lineKindKV {
			if _, ok := keySet[line.key]; ok {
				continue
			}
		}
		filtered = append(filtered, line)
	}
	f.lines = filtered
}

func (f *EnvFile) removeRawLine(raw string) {
	target := strings.TrimSpace(raw)
	if target == "" {
		return
	}

	filtered := make([]envLine, 0, len(f.lines))
	for _, line := range f.lines {
		if line.kind == lineKindRaw && strings.TrimSpace(line.raw) == target {
			continue
		}
		filtered = append(filtered, line)
	}
	f.lines = filtered
}

func (f *EnvFile) endsWithBlankOrEmpty() bool {
	for idx := len(f.lines) - 1; idx >= 0; idx-- {
		line := f.lines[idx]
		if line.kind == lineKindKV {
			return false
		}
		if strings.TrimSpace(line.raw) == "" {
			return true
		}
		return false
	}
	return true
}

func moduleSectionHeader(moduleID string) string {
	return fmt.Sprintf("# Module: %s (used only if %s module is installed)", moduleID, moduleID)
}

func normalizeModuleKeys(orderedKeys []string, defaults map[string]string) []string {
	seen := make(map[string]struct{})
	keys := make([]string, 0, len(orderedKeys)+len(defaults))

	for _, key := range orderedKeys {
		normalized := strings.TrimSpace(key)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		keys = append(keys, normalized)
	}

	extra := make([]string, 0, len(defaults))
	for key := range defaults {
		normalized := strings.TrimSpace(key)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		extra = append(extra, normalized)
	}
	sort.Strings(extra)

	keys = append(keys, extra...)
	return keys
}

func parseLine(raw string) envLine {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return envLine{
			kind: lineKindRaw,
			raw:  raw,
		}
	}

	key, value, ok := strings.Cut(raw, "=")
	if !ok {
		return envLine{
			kind: lineKindRaw,
			raw:  raw,
		}
	}

	normalizedKey := strings.TrimSpace(key)
	if strings.HasPrefix(normalizedKey, "export ") {
		normalizedKey = strings.TrimSpace(strings.TrimPrefix(normalizedKey, "export "))
	}
	if normalizedKey == "" {
		return envLine{
			kind: lineKindRaw,
			raw:  raw,
		}
	}

	return envLine{
		kind:  lineKindKV,
		key:   normalizedKey,
		value: decodeValue(strings.TrimSpace(value)),
	}
}

func encodeValue(value string) string {
	if value == "" {
		return ""
	}

	if strings.ContainsAny(value, " \t#\"'") {
		return strconv.Quote(value)
	}

	return value
}

func decodeValue(value string) string {
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
			unquoted, err := strconv.Unquote(value)
			if err == nil {
				return unquoted
			}
		}
	}

	return value
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func copyFile(sourcePath, destinationPath string) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(destinationPath)
	if err != nil {
		return err
	}
	defer destination.Close()

	if _, err := io.Copy(destination, source); err != nil {
		return err
	}

	return destination.Sync()
}
