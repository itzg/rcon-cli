package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindConfigFileIn(t *testing.T) {
	t.Run("no config", func(t *testing.T) {
		dir := t.TempDir()
		if got := findConfigFileIn(dir); got != "" {
			t.Errorf("findConfigFileIn() = %q, want empty", got)
		}
	})

	t.Run("missing directory", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "rcon-cli")
		if got := findConfigFileIn(dir); got != "" {
			t.Errorf("findConfigFileIn() = %q, want empty", got)
		}
	})

	t.Run("yaml", func(t *testing.T) {
		dir := t.TempDir()
		want := filepath.Join(dir, "config.yaml")
		writeFile(t, want)
		if got := findConfigFileIn(dir); got != want {
			t.Errorf("findConfigFileIn() = %q, want %q", got, want)
		}
	})

	t.Run("other extension", func(t *testing.T) {
		dir := t.TempDir()
		want := filepath.Join(dir, "config.toml")
		writeFile(t, want)
		if got := findConfigFileIn(dir); got != want {
			t.Errorf("findConfigFileIn() = %q, want %q", got, want)
		}
	})

	t.Run("ignores directories", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.Mkdir(filepath.Join(dir, "config.yaml"), 0o755); err != nil {
			t.Fatal(err)
		}
		if got := findConfigFileIn(dir); got != "" {
			t.Errorf("findConfigFileIn() = %q, want empty", got)
		}
	})
}

func writeFile(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("port: 25575\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}
