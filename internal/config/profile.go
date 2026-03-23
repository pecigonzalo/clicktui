// Package config manages application configuration, profiles, and config paths.
package config

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed config.schema.json
var schemaJSON []byte

// schemaFile is the filename written alongside config.yaml for editor hints.
const schemaFile = "config.schema.json"

// yamlLSPComment is the yaml-language-server modeline prepended to config.yaml.
const yamlLSPComment = "# yaml-language-server: $schema=config.schema.json\n"

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
	Name string `yaml:"name" json:"name"`
	// AuthMethod identifies how this profile authenticates.
	AuthMethod AuthMethod `yaml:"auth_method" json:"auth_method"`
	// WorkspaceID is an optional ClickUp workspace (team) ID to use by default.
	WorkspaceID string `yaml:"workspace_id,omitempty" json:"workspace_id,omitempty"`
	// SpaceID is an optional ClickUp space ID to navigate to on launch.
	SpaceID string `yaml:"space_id,omitempty" json:"space_id,omitempty"`
	// ListID is an optional ClickUp list ID to navigate to on launch.
	// Requires WorkspaceID and SpaceID to also be set.
	ListID string `yaml:"list_id,omitempty" json:"list_id,omitempty"`
}

// Config is the top-level application configuration.
type Config struct {
	// ActiveProfile is the name of the currently selected profile.
	ActiveProfile string `yaml:"active_profile" json:"active_profile"`
	// Profiles holds all configured profiles keyed by name.
	Profiles map[string]*Profile `yaml:"profiles" json:"profiles"`
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

// legacyConfigFile is the old JSON config filename used before the YAML migration.
const legacyConfigFile = "config.json"

// legacyConfigPath returns the path to the old JSON config file.
func legacyConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, legacyConfigFile), nil
}

// migrateFromJSON checks for a legacy config.json and migrates it to config.yaml.
// If config.yaml already exists, no migration is performed.
func migrateFromJSON(yamlPath string) (*Config, bool, error) {
	legacyPath, err := legacyConfigPath()
	if err != nil {
		return nil, false, err
	}

	data, err := os.ReadFile(legacyPath)
	if err != nil {
		return nil, false, nil //nolint:nilerr // file doesn't exist, no migration needed
	}

	cfg := New()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, false, fmt.Errorf("parse legacy config: %w", err)
	}

	// Write the migrated YAML config.
	if err := saveToPath(cfg, yamlPath); err != nil {
		return nil, false, fmt.Errorf("migrate config to yaml: %w", err)
	}

	// Remove the legacy JSON file after successful migration.
	_ = os.Remove(legacyPath)

	return cfg, true, nil
}

// Load reads the config file at ConfigFilePath, returning a new default Config
// if the file does not yet exist. If config.yaml is missing but a legacy
// config.json exists, it is migrated automatically.
func Load() (*Config, error) {
	path, err := ConfigFilePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		// Attempt migration from legacy JSON config.
		cfg, migrated, migrateErr := migrateFromJSON(path)
		if migrateErr != nil {
			return nil, migrateErr
		}
		if migrated {
			return cfg, nil
		}
		return New(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	cfg := New()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}

// saveToPath marshals cfg to YAML and writes it to the given path.
// The output is prefixed with a yaml-language-server modeline that references
// the co-located config.schema.json file.
func saveToPath(cfg *Config, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	// Ensure the file ends with a single trailing newline and prepend the
	// yaml-language-server modeline so editors pick up the co-located schema.
	body := strings.TrimRight(string(data), "\n") + "\n"
	out := yamlLSPComment + "\n" + body

	if err := os.WriteFile(path, []byte(out), 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// Save persists cfg to ConfigFilePath, creating the directory if needed.
// It also writes the embedded config.schema.json alongside config.yaml so that
// editors with the YAML Language Server extension can provide inline validation.
func Save(cfg *Config) error {
	path, err := ConfigFilePath()
	if err != nil {
		return err
	}
	if err := saveToPath(cfg, path); err != nil {
		return err
	}
	schemaPath := filepath.Join(filepath.Dir(path), schemaFile)
	if err := os.WriteFile(schemaPath, schemaJSON, 0o600); err != nil {
		return fmt.Errorf("write config schema: %w", err)
	}
	return nil
}
