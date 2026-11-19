// Package config reads the user's settings from the XDG configuration
// directory.
package config

import (
	"encoding/json/v2"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// Settings are what the server falls back on when the editor does not say
// otherwise.
type Settings struct {
	// Locale and CatalogPath come from systemd-lsp/config.json.
	Locale      string
	CatalogPath string
	// UserUnitDir is where the user service manager reads the user's own units.
	UserUnitDir string
}

type file struct {
	Locale      string `json:"locale,omitempty"`
	CatalogPath string `json:"catalogPath,omitempty"`
}

// Home returns the base directory for user configuration: $XDG_CONFIG_HOME,
// or $HOME/.config when that is unset. The XDG Base Directory Specification
// has relative paths ignored. Home returns "" when neither variable is usable.
func Home(getenv func(string) string) string {
	if dir := getenv("XDG_CONFIG_HOME"); filepath.IsAbs(dir) {
		return filepath.Clean(dir)
	}
	if home := getenv("HOME"); filepath.IsAbs(home) {
		return filepath.Join(home, ".config")
	}
	return ""
}

// Path returns the configuration file below the directory Home returned.
func Path(home string) string {
	return filepath.Join(home, "systemd-lsp", "config.json")
}

// Load returns the settings for the given environment. A missing configuration
// file is not an error. On any other error the settings that do not come from
// the file are still returned.
func Load(getenv func(string) string) (Settings, error) {
	home := Home(getenv)
	if home == "" {
		return Settings{}, nil
	}
	settings := Settings{UserUnitDir: filepath.Join(home, "systemd", "user")}
	path := Path(home)
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return settings, nil
	}
	if err != nil {
		return settings, err
	}
	var f file
	if err := json.Unmarshal(data, &f); err != nil {
		return settings, err
	}
	settings.Locale = f.Locale
	settings.CatalogPath = f.CatalogPath
	// The server's working directory is whatever the editor happened to use.
	if settings.CatalogPath != "" && !filepath.IsAbs(settings.CatalogPath) {
		settings.CatalogPath = filepath.Join(filepath.Dir(path), settings.CatalogPath)
	}
	return settings, nil
}
