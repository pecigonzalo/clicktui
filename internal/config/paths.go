// Package config manages application configuration, profiles, and config paths.
package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	appName    = "clicktui"
	configFile = "config.yaml"
)

// ConfigDir returns the OS-appropriate application config directory.
// On Unix-like systems this follows XDG_CONFIG_HOME.
func ConfigDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("config dir: %w", err)
	}
	return filepath.Join(base, appName), nil
}

// ConfigFilePath returns the path to the main config file.
func ConfigFilePath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configFile), nil
}

// DataDir returns the OS-appropriate application data directory.
// Falls back to ConfigDir if XDG_DATA_HOME is unavailable.
func DataDir() (string, error) {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, appName), nil
	}
	// Fallback: use config dir (acceptable for a CLI/TUI app).
	return ConfigDir()
}
