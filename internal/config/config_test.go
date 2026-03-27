package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

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
	// YAML serialisation round-trips correctly without touching the real OS
	// config directory.
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")

	cfg := config.New()
	cfg.SetProfile(&config.Profile{
		Name:        "default",
		AuthMethod:  config.AuthMethodPersonalToken,
		WorkspaceID: "ws123",
	})
	cfg.ActiveProfile = "default"

	data, err := yaml.Marshal(cfg)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(cfgFile, data, 0o600))

	var loaded config.Config
	raw, err := os.ReadFile(cfgFile)
	require.NoError(t, err)
	require.NoError(t, yaml.Unmarshal(raw, &loaded))

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

func TestSave_WritesSchemaFile(t *testing.T) {
	setConfigDir(t, t.TempDir())

	cfg := config.New()
	cfg.SetProfile(&config.Profile{
		Name:       "default",
		AuthMethod: config.AuthMethodPersonalToken,
	})
	require.NoError(t, config.Save(cfg))

	cfgDir, err := config.ConfigDir()
	require.NoError(t, err)

	// config.schema.json should exist alongside config.yaml.
	schemaPath := filepath.Join(cfgDir, "config.schema.json")
	info, err := os.Stat(schemaPath)
	require.NoError(t, err, "config.schema.json should be written by Save()")
	assert.Greater(t, info.Size(), int64(0), "config.schema.json should not be empty")

	// The schema file must be valid JSON.
	raw, err := os.ReadFile(schemaPath)
	require.NoError(t, err)
	var schemaObj map[string]any
	require.NoError(t, json.Unmarshal(raw, &schemaObj), "config.schema.json must be valid JSON")
}

func TestSave_PrependsYAMLLSPComment(t *testing.T) {
	setConfigDir(t, t.TempDir())

	cfg := config.New()
	cfg.SetProfile(&config.Profile{
		Name:       "default",
		AuthMethod: config.AuthMethodPersonalToken,
	})
	require.NoError(t, config.Save(cfg))

	cfgPath, err := config.ConfigFilePath()
	require.NoError(t, err)

	raw, err := os.ReadFile(cfgPath)
	require.NoError(t, err)
	content := string(raw)

	const wantComment = "# yaml-language-server: $schema=config.schema.json"
	assert.True(t, strings.HasPrefix(content, wantComment), "config.yaml should start with the yaml-language-server modeline")
}

func TestSave_SchemaComment_SurvivesRoundTrip(t *testing.T) {
	setConfigDir(t, t.TempDir())

	cfg := config.New()
	cfg.SetProfile(&config.Profile{
		Name:        "default",
		AuthMethod:  config.AuthMethodPersonalToken,
		WorkspaceID: "ws-round",
	})
	require.NoError(t, config.Save(cfg))

	// Load should ignore YAML comments naturally.
	loaded, err := config.Load()
	require.NoError(t, err)

	// Save again — comment must still be present.
	require.NoError(t, config.Save(loaded))

	cfgPath, err := config.ConfigFilePath()
	require.NoError(t, err)
	raw, err := os.ReadFile(cfgPath)
	require.NoError(t, err)

	const wantComment = "# yaml-language-server: $schema=config.schema.json"
	assert.True(t, strings.HasPrefix(string(raw), wantComment), "comment should survive a save → load → save cycle")

	// Data integrity must be preserved.
	loaded2, err := config.Load()
	require.NoError(t, err)
	p, err := loaded2.Profile("default")
	require.NoError(t, err)
	assert.Equal(t, "ws-round", p.WorkspaceID)
}

func TestLoad_InvalidYAML_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	cfgPath := configFileInDir(t, dir) // sets HOME / XDG_CONFIG_HOME for us

	// Create parent directories and write a broken YAML file.
	require.NoError(t, os.MkdirAll(filepath.Dir(cfgPath), 0o700))
	require.NoError(t, os.WriteFile(cfgPath, []byte(":\t:\nbad:\n  - :\t["), 0o600))

	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse config")
}

func TestLoad_MigratesLegacyJSON(t *testing.T) {
	dir := t.TempDir()
	setConfigDir(t, dir)

	// Write a legacy JSON config file.
	cfgDir, err := config.ConfigDir()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(cfgDir, 0o700))

	legacyPath := filepath.Join(cfgDir, "config.json")
	legacyCfg := config.New()
	legacyCfg.SetProfile(&config.Profile{
		Name:        "default",
		AuthMethod:  config.AuthMethodPersonalToken,
		WorkspaceID: "ws-legacy",
	})
	data, err := json.MarshalIndent(legacyCfg, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(legacyPath, data, 0o600))

	// Load should migrate the JSON to YAML automatically.
	loaded, err := config.Load()
	require.NoError(t, err)
	p, err := loaded.Profile("default")
	require.NoError(t, err)
	assert.Equal(t, "ws-legacy", p.WorkspaceID)

	// The legacy JSON file should have been removed.
	_, err = os.Stat(legacyPath)
	assert.True(t, os.IsNotExist(err), "legacy config.json should be removed after migration")

	// The new YAML file should exist.
	yamlPath, err := config.ConfigFilePath()
	require.NoError(t, err)
	_, err = os.Stat(yamlPath)
	assert.NoError(t, err, "config.yaml should exist after migration")
}

func TestLoad_NoMigrationWhenYAMLExists(t *testing.T) {
	dir := t.TempDir()
	setConfigDir(t, dir)

	cfgDir, err := config.ConfigDir()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(cfgDir, 0o700))

	// Write both a legacy JSON and a current YAML config.
	legacyPath := filepath.Join(cfgDir, "config.json")
	require.NoError(t, os.WriteFile(legacyPath, []byte(`{"active_profile":"default","profiles":{"default":{"name":"default","auth_method":"personal_token","workspace_id":"ws-json"}}}`), 0o600))

	yamlPath := filepath.Join(cfgDir, "config.yaml")
	yamlCfg := config.New()
	yamlCfg.SetProfile(&config.Profile{
		Name:        "default",
		AuthMethod:  config.AuthMethodPersonalToken,
		WorkspaceID: "ws-yaml",
	})
	yamlData, err := yaml.Marshal(yamlCfg)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(yamlPath, yamlData, 0o600))

	// Load should use the YAML file, not the JSON.
	loaded, err := config.Load()
	require.NoError(t, err)
	p, err := loaded.Profile("default")
	require.NoError(t, err)
	assert.Equal(t, "ws-yaml", p.WorkspaceID, "should load from YAML, not legacy JSON")

	// Legacy JSON should still be present (not touched when YAML exists).
	_, err = os.Stat(legacyPath)
	assert.NoError(t, err, "legacy config.json should remain untouched when config.yaml exists")
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
	assert.Equal(t, filepath.Join(dir, "config.yaml"), path)
}

func TestNerdFont_YAMLRoundTrip(t *testing.T) {
	cases := []struct {
		name     string
		nerdFont bool
	}{
		{"nerd_font_true", true},
		{"nerd_font_false", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.New()
			cfg.NerdFont = tc.nerdFont
			cfg.SetProfile(&config.Profile{
				Name:       "default",
				AuthMethod: config.AuthMethodPersonalToken,
			})

			data, err := yaml.Marshal(cfg)
			require.NoError(t, err)

			var loaded config.Config
			require.NoError(t, yaml.Unmarshal(data, &loaded))

			assert.Equal(t, tc.nerdFont, loaded.NerdFont, "NerdFont(%v) yaml round-trip mismatch", tc.nerdFont)
		})
	}
}

func TestNerdFontEnabled_EnvOverride(t *testing.T) {
	cases := []struct {
		name      string
		envValue  string
		configVal bool
		want      bool
	}{
		{"env_1_overrides_false", "1", false, true},
		{"env_true_overrides_false", "true", false, true},
		{"env_yes_overrides_false", "yes", false, true},
		{"env_TRUE_case_insensitive", "TRUE", false, true},
		{"env_YES_case_insensitive", "YES", false, true},
		{"env_0_overrides_true", "0", true, false},
		{"env_false_overrides_true", "false", true, false},
		{"env_no_overrides_true", "no", true, false},
		{"env_FALSE_case_insensitive", "FALSE", true, false},
		{"env_NO_case_insensitive", "NO", true, false},
		{"env_unknown_falls_back_to_config_true", "maybe", true, true},
		{"env_unknown_falls_back_to_config_false", "maybe", false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("CLICKTUI_NERD_FONTS", tc.envValue)
			cfg := &config.Config{NerdFont: tc.configVal}
			got := cfg.NerdFontEnabled()
			assert.Equal(t, tc.want, got, "NerdFontEnabled() mismatch (env=%q, config=%v)", tc.envValue, tc.configVal)
		})
	}
}

func TestNerdFontEnabled_FallsBackToConfig(t *testing.T) {
	cases := []struct {
		name      string
		configVal bool
	}{
		{"config_true_no_env", true},
		{"config_false_no_env", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Ensure env var is unset for this test.
			t.Setenv("CLICKTUI_NERD_FONTS", "")
			cfg := &config.Config{NerdFont: tc.configVal}
			got := cfg.NerdFontEnabled()
			assert.Equal(t, tc.configVal, got, "NerdFontEnabled() mismatch with no env set")
		})
	}
}

func TestDataDir_FallsBackToConfigDir(t *testing.T) {
	setConfigDir(t, t.TempDir())
	t.Setenv("XDG_DATA_HOME", "")

	dataDir, err := config.DataDir()
	require.NoError(t, err)

	configDir, err := config.ConfigDir()
	require.NoError(t, err)

	assert.Equal(t, configDir, dataDir)
}
