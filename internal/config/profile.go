// Package config manages application configuration, profiles, and config paths.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// AuthMethod identifies the authentication mechanism used by a profile.
type AuthMethod string

const (
	// AuthMethodPersonalToken authenticates via a ClickUp personal API token.
	AuthMethodPersonalToken AuthMethod = "personal_token"
	// AuthMethodOAuth authenticates via OAuth 2.0 (not yet implemented).
	AuthMethodOAuth AuthMethod = "oauth"
)

// Profile holds per-profile settings including auth method and workspace selection.
type Profile struct {
	// Name is the profile identifier.
	Name string `json:"name"`
	// AuthMethod identifies how this profile authenticates.
	AuthMethod AuthMethod `json:"auth_method"`
	// WorkspaceID is an optional ClickUp workspace (team) ID to use by default.
	WorkspaceID string `json:"workspace_id,omitempty"`
	// SpaceID is an optional ClickUp space ID to navigate to on launch.
	SpaceID string `json:"space_id,omitempty"`
	// ListID is an optional ClickUp list ID to navigate to on launch.
	// Requires WorkspaceID and SpaceID to also be set.
	ListID string `json:"list_id,omitempty"`
}

// Config is the top-level application configuration.
type Config struct {
	// ActiveProfile is the name of the currently selected profile.
	ActiveProfile string `json:"active_profile"`
	// Profiles holds all configured profiles keyed by name.
	Profiles map[string]*Profile `json:"profiles"`
}

var (
	// ErrProfileNotFound is returned when the requested profile does not exist.
	ErrProfileNotFound = errors.New("profile not found")
)

// DefaultProfile returns the default profile name.
func DefaultProfile() string {
	return "default"
}

// New returns an empty Config with an initialised profiles map.
func New() *Config {
	return &Config{
		ActiveProfile: DefaultProfile(),
		Profiles:      make(map[string]*Profile),
	}
}

// Active returns the currently active profile, or ErrProfileNotFound.
func (c *Config) Active() (*Profile, error) {
	return c.Profile(c.ActiveProfile)
}

// Profile returns the named profile, or ErrProfileNotFound.
func (c *Config) Profile(name string) (*Profile, error) {
	p, ok := c.Profiles[name]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrProfileNotFound, name)
	}
	return p, nil
}

// SetProfile upserts p into the config under p.Name.
func (c *Config) SetProfile(p *Profile) {
	c.Profiles[p.Name] = p
}

// Load reads the config file at ConfigFilePath, returning a new default Config
// if the file does not yet exist.
func Load() (*Config, error) {
	path, err := ConfigFilePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return New(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	cfg := New()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}

// Save persists cfg to ConfigFilePath, creating the directory if needed.
func Save(cfg *Config) error {
	path, err := ConfigFilePath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}
