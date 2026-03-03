package templatectl

import (
	"os"
	"path/filepath"
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
