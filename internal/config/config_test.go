package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pecigonzalo/clicktui/internal/config"
)

// setConfigDir redirects the OS user-config directory for the duration of t.
// On Linux/BSD it sets XDG_CONFIG_HOME. On macOS it also sets HOME so that
// os.UserConfigDir returns <dir>/Library/Application Support.
// t.Setenv handles both cleanup and error checking automatically.
func setConfigDir(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)
}

// configFileInDir returns the path where config.Load/Save will read/write when
// the home/config dir is set to dir.  It calls config.ConfigFilePath() after
// redirecting the dir so we don't hard-code platform differences.
func configFileInDir(t *testing.T, dir string) string {
	t.Helper()
	setConfigDir(t, dir)
	path, err := config.ConfigFilePath()
	require.NoError(t, err)
	return path
}

func TestNewConfig_Defaults(t *testing.T) {
	cfg := config.New()
	assert.Equal(t, config.DefaultProfile(), cfg.ActiveProfile)
	assert.NotNil(t, cfg.Profiles)
}

func TestSetAndGetProfile(t *testing.T) {
	cfg := config.New()
	p := &config.Profile{Name: "work", AuthMethod: config.AuthMethodPersonalToken}
	cfg.SetProfile(p)

	got, err := cfg.Profile("work")
	require.NoError(t, err)
	assert.Equal(t, p, got)
}

func TestGetProfile_NotFound(t *testing.T) {
	cfg := config.New()
	_, err := cfg.Profile("nonexistent")
	assert.ErrorIs(t, err, config.ErrProfileNotFound)
}

func TestSaveAndLoad_RoundTrip(t *testing.T) {
	// Write a config directly to a temp file and read it back to verify the
	// JSON serialisation round-trips correctly without touching the real OS
	// config directory.
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.json")

	cfg := config.New()
	cfg.SetProfile(&config.Profile{
		Name:        "default",
		AuthMethod:  config.AuthMethodPersonalToken,
		WorkspaceID: "ws123",
	})
	cfg.ActiveProfile = "default"

	data, err := json.MarshalIndent(cfg, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(cfgFile, data, 0o600))

	var loaded config.Config
	raw, err := os.ReadFile(cfgFile)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &loaded))

	assert.Equal(t, "default", loaded.ActiveProfile)
	p, err := loaded.Profile("default")
	require.NoError(t, err)
	assert.Equal(t, "ws123", p.WorkspaceID)
	assert.Equal(t, config.AuthMethodPersonalToken, p.AuthMethod)
}

func TestActive_ReturnsActiveProfile(t *testing.T) {
	cfg := config.New()
	cfg.SetProfile(&config.Profile{Name: "default", AuthMethod: config.AuthMethodPersonalToken})

	p, err := cfg.Active()
	require.NoError(t, err)
	assert.Equal(t, "default", p.Name)
}

func TestActive_NotFound(t *testing.T) {
	cfg := config.New()
	// No profiles added; "default" does not exist.
	_, err := cfg.Active()
	require.ErrorIs(t, err, config.ErrProfileNotFound)
}

func TestLoad_MissingFile_ReturnsDefault(t *testing.T) {
	setConfigDir(t, t.TempDir())
	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, config.DefaultProfile(), cfg.ActiveProfile)
	assert.NotNil(t, cfg.Profiles)
}

func TestSave_And_Load(t *testing.T) {
	setConfigDir(t, t.TempDir())

	cfg := config.New()
	cfg.SetProfile(&config.Profile{
		Name:        "default",
		AuthMethod:  config.AuthMethodPersonalToken,
		WorkspaceID: "ws42",
	})
	require.NoError(t, config.Save(cfg))

	loaded, err := config.Load()
	require.NoError(t, err)
	p, err := loaded.Profile("default")
	require.NoError(t, err)
	assert.Equal(t, "ws42", p.WorkspaceID)
	assert.Equal(t, config.AuthMethodPersonalToken, p.AuthMethod)
}

func TestLoad_InvalidJSON_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	cfgPath := configFileInDir(t, dir) // sets HOME / XDG_CONFIG_HOME for us

	// Create parent directories and write a broken JSON file.
	require.NoError(t, os.MkdirAll(filepath.Dir(cfgPath), 0o700))
	require.NoError(t, os.WriteFile(cfgPath, []byte("not-json{{{"), 0o600))

	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse config")
}

func TestDataDir_UsesXDGDataHome(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)

	dir, err := config.DataDir()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(tmp, "clicktui"), dir)
}

func TestConfigDir_And_ConfigFilePath(t *testing.T) {
	setConfigDir(t, t.TempDir())

	dir, err := config.ConfigDir()
	require.NoError(t, err)
	assert.NotEmpty(t, dir)

	path, err := config.ConfigFilePath()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "config.json"), path)
}
