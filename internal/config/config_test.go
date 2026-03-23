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
