package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rigofekete/mocho/internal/config"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// withXDG points the config-file lookup at a temp directory.
func withXDG(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	return dir
}

func TestDefaultWhenNothingSet(t *testing.T) {
	withXDG(t)
	t.Setenv("MOCHO_WIKI", "")
	t.Setenv("MOCHO_ADDR", "")
	cfg, err := config.Resolve("", "")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addr == "" {
		t.Fatalf("expected defaults, got %+v", cfg)
	}
}

func TestEnvOverridesDefault(t *testing.T) {
	withXDG(t)
	t.Setenv("MOCHO_WIKI", "/tmp/env-wiki")
	cfg, err := config.Resolve("", "")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WikiPath != "/tmp/env-wiki" {
		t.Fatalf("env did not override default: %+v", cfg)
	}
}

func TestConfigFileOverridesDefault(t *testing.T) {
	dir := withXDG(t)
	writeFile(t, filepath.Join(dir, "mocho", "config.json"), `{"wikiPath": "/tmp/file-wiki", "addr": "127.0.0.1:9999"}`)
	t.Setenv("MOCHO_WIKI", "")
	cfg, err := config.Resolve("", "")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WikiPath != "/tmp/file-wiki" {
		t.Fatalf("config file did not override default: %+v", cfg)
	}
	if cfg.Addr != "127.0.0.1:9999" {
		t.Fatalf("addr from config file wrong: %+v", cfg)
	}
}

func TestEnvOverridesConfigFile(t *testing.T) {
	dir := withXDG(t)
	writeFile(t, filepath.Join(dir, "mocho", "config.json"), `{"wikiPath":"/tmp/file-wiki"}`)
	t.Setenv("MOCHO_WIKI", "/tmp/env-wiki")
	cfg, err := config.Resolve("", "")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WikiPath != "/tmp/env-wiki" {
		t.Fatalf("env should win over file: %+v", cfg)
	}
}

func TestFlagWins(t *testing.T) {
	dir := withXDG(t)
	writeFile(t, filepath.Join(dir, "mocho", "config.json"), `{"wikiPath":"/tmp/file-wiki"}`)
	t.Setenv("MOCHO_WIKI", "/tmp/env-wiki")
	cfg, err := config.Resolve("/tmp/flag-wiki", "")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WikiPath != "/tmp/flag-wiki" {
		t.Fatalf("flag should win: %+v", cfg)
	}
}

func TestDefaultModels(t *testing.T) {
	withXDG(t)
	t.Setenv("MOCHO_LIGHT_MODEL", "")
	t.Setenv("MOCHO_WIKI_MODEL", "")
	cfg, err := config.Resolve("", "")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LightModel != config.DefaultLightModel {
		t.Fatalf("light model = %q, want default %q", cfg.LightModel, config.DefaultLightModel)
	}
	if cfg.WikiModel != config.DefaultWikiModel {
		t.Fatalf("wiki model = %q, want default %q", cfg.WikiModel, config.DefaultWikiModel)
	}
}

func TestModelEnvOverrides(t *testing.T) {
	withXDG(t)
	t.Setenv("MOCHO_LIGHT_MODEL", "opencode/cheap")
	t.Setenv("MOCHO_WIKI_MODEL", "opencode/frontier")
	cfg, err := config.Resolve("", "")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LightModel != "opencode/cheap" {
		t.Fatalf("light model = %q", cfg.LightModel)
	}
	if cfg.WikiModel != "opencode/frontier" {
		t.Fatalf("wiki model = %q", cfg.WikiModel)
	}
}

func TestModelConfigFileOverrides(t *testing.T) {
	dir := withXDG(t)
	writeFile(t, filepath.Join(dir, "mocho", "config.json"),
		`{"wikiPath":"/tmp/file-wiki","lightModel":"opencode/file-light","wikiModel":"opencode/file-wiki"}`)
	t.Setenv("MOCHO_LIGHT_MODEL", "")
	t.Setenv("MOCHO_WIKI_MODEL", "")
	cfg, err := config.Resolve("", "")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LightModel != "opencode/file-light" {
		t.Fatalf("light model = %q", cfg.LightModel)
	}
	if cfg.WikiModel != "opencode/file-wiki" {
		t.Fatalf("wiki model = %q", cfg.WikiModel)
	}
}

func TestModelEnvOverridesConfigFile(t *testing.T) {
	dir := withXDG(t)
	writeFile(t, filepath.Join(dir, "mocho", "config.json"),
		`{"wikiPath":"/tmp/file-wiki","lightModel":"opencode/file-light","wikiModel":"opencode/file-wiki"}`)
	t.Setenv("MOCHO_LIGHT_MODEL", "opencode/env-light")
	cfg, err := config.Resolve("", "")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LightModel != "opencode/env-light" {
		t.Fatalf("env should win over file: %q", cfg.LightModel)
	}
	// wiki model not overridden by env keeps config file value.
	if cfg.WikiModel != "opencode/file-wiki" {
		t.Fatalf("wiki model should keep file value: %q", cfg.WikiModel)
	}
}
