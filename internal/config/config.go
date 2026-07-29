// Package config resolves the runtime configuration for the mocho server.
// Wiki path precedence: flag > env var > config file > default.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds runtime settings.
type Config struct {
	WikiPath string `json:"wikiPath"`
	Addr     string `json:"addr"`
}

// DefaultWikiPath is ~/Work/dev/mocho-wiki (independent of the app repo).
func DefaultWikiPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	} else if home == "" {
		return "", errors.New("$HOME env not set")
	}
	return filepath.Join(home, "Work", "dev", "mocho-wiki"), nil
}

// configFile returns the path to the user config file, if any.
func configFile() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "mocho", "config.json")
}

// loadFile reads the config file into cfg if it exists. Missing is not an error.
func loadFile(path string, cfg *Config) error {
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read config %s: %w", path, err)
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("parse config %s: %w", path, err)
	}
	return nil
}

// Resolve builds the effective config given any explicit flag overrides. Flag
// values are non-empty only when the user actually set them; empty flags defer
// to the next layer down.
func Resolve(flagWiki, flagAddr string) (Config, error) {
	wikiPath, err := DefaultWikiPath()
	if err != nil {
		return Config{}, err
	}
	cfg := Config{
		WikiPath: wikiPath,
		Addr:     "127.0.0.1:7777",
	}
	if path := configFile(); path != "" {
		if err := loadFile(path, &cfg); err != nil {
			return Config{}, err
		}
	}
	if v := os.Getenv("MOCHO_WIKI"); v != "" {
		cfg.WikiPath = v
	}
	if v := os.Getenv("MOCHO_ADDR"); v != "" {
		cfg.Addr = v
	}
	if flagWiki != "" {
		cfg.WikiPath = flagWiki
	}
	if flagAddr != "" {
		cfg.Addr = flagAddr
	}
	if cfg.WikiPath == "" {
		return Config{}, errors.New("wiki path is empty")
	}
	return cfg, nil
}
