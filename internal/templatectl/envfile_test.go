package templatectl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnvUnsetRemovesLineWithoutLeavingGap(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("A=1\nB=2\n"), 0o644); err != nil {
		t.Fatalf("failed to write env file: %v", err)
	}

	env, err := loadEnvFile(path)
	if err != nil {
		t.Fatalf("failed to load env file: %v", err)
	}

	env.Unset("A")
	if err := env.Save(); err != nil {
		t.Fatalf("failed to save env file: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read env file: %v", err)
	}

	if string(raw) != "B=2\n" {
		t.Fatalf("unexpected env content after unset:\n%s", string(raw))
	}
}

func TestEnvSaveCompactsBlankLines(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), ".env")
	input := "A=1\n\n\n\nB=2\n\n\n"
	if err := os.WriteFile(path, []byte(input), 0o644); err != nil {
		t.Fatalf("failed to write env file: %v", err)
	}

	env, err := loadEnvFile(path)
	if err != nil {
		t.Fatalf("failed to load env file: %v", err)
	}

	if err := env.Save(); err != nil {
		t.Fatalf("failed to save env file: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read env file: %v", err)
	}

	expected := "A=1\n\nB=2\n"
	if string(raw) != expected {
		t.Fatalf("unexpected compacted env content:\n%s", string(raw))
	}
}

func TestEnvUpsertModuleSectionGroupsKeysAndPreservesValues(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), ".env")
	input := "PORT=8888\nAUTH_TOKEN=custom-token\n\nSQLITE_PATH=legacy.db\n"
	if err := os.WriteFile(path, []byte(input), 0o644); err != nil {
		t.Fatalf("failed to write env file: %v", err)
	}

	env, err := loadEnvFile(path)
	if err != nil {
		t.Fatalf("failed to load env file: %v", err)
	}

	env.UpsertModuleSection("db-sqlite", []string{"SQLITE_PATH", "SQLITE_BUSY_TIMEOUT_MS", "SQLITE_CONNECT_TIMEOUT_SEC"}, map[string]string{
		"SQLITE_PATH":                "data/app.db",
		"SQLITE_BUSY_TIMEOUT_MS":     "5000",
		"SQLITE_CONNECT_TIMEOUT_SEC": "5",
	})
	env.UpsertModuleSection("auth-token", []string{"AUTH_TOKEN"}, map[string]string{
		"AUTH_TOKEN": "change-me",
	})

	if err := env.Save(); err != nil {
		t.Fatalf("failed to save env file: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read env file: %v", err)
	}

	expected := strings.Join([]string{
		"PORT=8888",
		"",
		"# Module: db-sqlite (used only if db-sqlite module is installed)",
		"SQLITE_PATH=legacy.db",
		"SQLITE_BUSY_TIMEOUT_MS=5000",
		"SQLITE_CONNECT_TIMEOUT_SEC=5",
		"",
		"# Module: auth-token (used only if auth-token module is installed)",
		"AUTH_TOKEN=custom-token",
		"",
	}, "\n")
	if string(raw) != expected {
		t.Fatalf("unexpected module section content:\n%s", string(raw))
	}
}

func TestEnvRemoveModuleSectionRemovesHeaderAndKeys(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), ".env")
	input := strings.Join([]string{
		"PORT=8888",
		"",
		"# Module: db-sqlite (used only if db-sqlite module is installed)",
		"SQLITE_PATH=data/app.db",
		"SQLITE_BUSY_TIMEOUT_MS=5000",
		"SQLITE_CONNECT_TIMEOUT_SEC=5",
		"",
		"LOG_LEVEL=INFO",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(input), 0o644); err != nil {
		t.Fatalf("failed to write env file: %v", err)
	}

	env, err := loadEnvFile(path)
	if err != nil {
		t.Fatalf("failed to load env file: %v", err)
	}

	env.RemoveModuleSection("db-sqlite", []string{"SQLITE_PATH", "SQLITE_BUSY_TIMEOUT_MS", "SQLITE_CONNECT_TIMEOUT_SEC"})
	if err := env.Save(); err != nil {
		t.Fatalf("failed to save env file: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read env file: %v", err)
	}

	expected := "PORT=8888\n\nLOG_LEVEL=INFO\n"
	if string(raw) != expected {
		t.Fatalf("unexpected env content after section removal:\n%s", string(raw))
	}
}
