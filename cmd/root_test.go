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

	t.Run("other extension ignored", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "config.toml"))
		if got := findConfigFileIn(dir); got != "" {
			t.Errorf("findConfigFileIn() = %q, want empty (only yaml is read)", got)
		}
	})

	t.Run("yml", func(t *testing.T) {
		dir := t.TempDir()
		want := filepath.Join(dir, "config.yml")
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

func TestInitConfigLoadsFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("host: mc1\nport: 25576\npassword: secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	oldCfgFile := cfgFile
	oldConfig := config
	t.Cleanup(func() {
		cfgFile = oldCfgFile
		config = oldConfig
	})

	cfgFile = path
	config = Config{Host: "localhost", Port: 25575, AbortOnCtrlC: true}
	initConfig()

	if config.Host != "mc1" || config.Port != 25576 || config.Password != "secret" {
		t.Errorf("config after initConfig() = %+v, want host mc1, port 25576, password secret", config)
	}
}

func TestApplyConfigFilePrecedence(t *testing.T) {
	str := func(s string) *string { return &s }
	num := func(n int) *int { return &n }
	boolean := func(b bool) *bool { return &b }

	t.Run("file beats defaults", func(t *testing.T) {
		old := config
		t.Cleanup(func() { config = old })
		config = Config{Host: "localhost", Port: 25575, AbortOnCtrlC: true}
		applyConfigFile(&fileConfig{Host: str("mc1"), Port: num(25576)})
		if config.Host != "mc1" || config.Port != 25576 {
			t.Errorf("config = %+v, want host mc1, port 25576", config)
		}
	})

	t.Run("omitted keys keep their values", func(t *testing.T) {
		old := config
		t.Cleanup(func() { config = old })
		config = Config{Host: "localhost", Port: 25575, Password: "keep", AbortOnCtrlC: true}
		applyConfigFile(&fileConfig{Host: str("mc1")})
		if config.Port != 25575 || config.Password != "keep" || !config.AbortOnCtrlC {
			t.Errorf("config = %+v, want port 25575, password keep, abort true", config)
		}
	})

	t.Run("env beats file", func(t *testing.T) {
		old := config
		t.Cleanup(func() { config = old })
		t.Setenv("RCON_PORT", "25577")
		config = Config{Host: "localhost", Port: 25575, AbortOnCtrlC: true}
		applyConfigFile(&fileConfig{Port: num(25576)})
		if config.Port != 25575 {
			t.Errorf("config.Port = %d, want 25575 (env set, file must not win)", config.Port)
		}
	})

	t.Run("flag beats file", func(t *testing.T) {
		old := config
		t.Cleanup(func() {
			config = old
			RootCmd.PersistentFlags().Lookup("port").Changed = false
		})
		config = Config{Host: "localhost", Port: 25575, AbortOnCtrlC: true}
		if err := RootCmd.PersistentFlags().Set("port", "25578"); err != nil {
			t.Fatal(err)
		}
		applyConfigFile(&fileConfig{Port: num(25576)})
		if config.Port != 25578 {
			t.Errorf("config.Port = %d, want 25578 (flag set, file must not win)", config.Port)
		}
	})

	t.Run("bool from file", func(t *testing.T) {
		old := config
		t.Cleanup(func() { config = old })
		config = Config{Host: "localhost", Port: 25575, AbortOnCtrlC: true}
		applyConfigFile(&fileConfig{AbortOnCtrlC: boolean(false)})
		if config.AbortOnCtrlC {
			t.Errorf("config.AbortOnCtrlC = true, want false from file")
		}
	})
}
