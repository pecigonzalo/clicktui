package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pecigonzalo/clicktui/internal/config"
)

// setTestConfigDir redirects the OS user-config directory for the duration of t.
// On Linux/BSD it sets XDG_CONFIG_HOME. On macOS it also sets HOME so that
// os.UserConfigDir returns <dir>/Library/Application Support.
// t.Setenv handles both cleanup and error checking automatically.
func setTestConfigDir(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)
}

// writeTestConfig writes a minimal YAML config file into a temp directory and
// redirects XDG_CONFIG_HOME/HOME so config.Load() reads it.
func writeTestConfig(t *testing.T, content string) {
	t.Helper()

	dir := t.TempDir()
	setTestConfigDir(t, dir)

	// Determine where config.Load() will look by calling the real path helper
	// after redirecting the env vars.
	cfgPath, err := config.ConfigFilePath()
	require.NoError(t, err)

	err = os.MkdirAll(filepath.Dir(cfgPath), 0o700)
	require.NoError(t, err)
	err = os.WriteFile(cfgPath, []byte(content), 0o600)
	require.NoError(t, err)
}

func TestResolveProfile(t *testing.T) {
	cases := []struct {
		name             string
		configYAML       string
		profileFlagValue string
		profileChanged   bool
		wantProfile      string
		wantWorkspaceID  string
		wantSpaceID      string
		wantListID       string
	}{
		{
			name: "no_config_returns_flag_default",
			// No config file — config.Load() returns a fresh Config with empty
			// active_profile and no profiles, so the flag value is used as-is.
			configYAML:       "",
			profileFlagValue: "default",
			profileChanged:   false,
			wantProfile:      "default",
		},
		{
			name: "active_profile_overrides_flag_default_when_flag_not_changed",
			configYAML: `active_profile: work
profiles:
  work:
    name: work
    auth_method: personal_token
    workspace_id: "ws-work"
    space_id: "sp-work"
    list_id: "li-work"
`,
			profileFlagValue: "default",
			profileChanged:   false,
			wantProfile:      "work",
			wantWorkspaceID:  "ws-work",
			wantSpaceID:      "sp-work",
			wantListID:       "li-work",
		},
		{
			name: "explicit_profile_flag_takes_precedence_over_active_profile",
			configYAML: `active_profile: work
profiles:
  work:
    name: work
    auth_method: personal_token
    workspace_id: "ws-work"
  staging:
    name: staging
    auth_method: personal_token
    workspace_id: "ws-staging"
`,
			profileFlagValue: "staging",
			profileChanged:   true,
			wantProfile:      "staging",
			wantWorkspaceID:  "ws-staging",
		},
		{
			name: "explicit_profile_flag_default_string_is_honoured",
			// --profile default was explicitly passed; active_profile must NOT
			// override it even though it points elsewhere.
			configYAML: `active_profile: work
profiles:
  work:
    name: work
    auth_method: personal_token
    workspace_id: "ws-work"
  default:
    name: default
    auth_method: personal_token
    workspace_id: "ws-default"
`,
			profileFlagValue: "default",
			profileChanged:   true,
			wantProfile:      "default",
			wantWorkspaceID:  "ws-default",
		},
		{
			name: "profile_defaults_loaded_from_resolved_profile",
			configYAML: `active_profile: ""
profiles:
  default:
    name: default
    auth_method: personal_token
    workspace_id: "ws-123"
    space_id: "sp-456"
    list_id: "li-789"
`,
			profileFlagValue: "default",
			profileChanged:   false,
			wantProfile:      "default",
			wantWorkspaceID:  "ws-123",
			wantSpaceID:      "sp-456",
			wantListID:       "li-789",
		},
		{
			name: "missing_profile_in_config_returns_empty_defaults",
			configYAML: `active_profile: ""
profiles: {}
`,
			profileFlagValue: "ghost",
			profileChanged:   true,
			wantProfile:      "ghost",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.configYAML != "" {
				writeTestConfig(t, tc.configYAML)
			} else {
				// Point at an empty temp dir so no config file exists.
				setTestConfigDir(t, t.TempDir())
			}

			got := ResolveProfile(tc.profileFlagValue, tc.profileChanged)

			assert.Equal(t, tc.wantProfile, got.Profile)
			assert.Equal(t, tc.wantWorkspaceID, got.WorkspaceID)
			assert.Equal(t, tc.wantSpaceID, got.SpaceID)
			assert.Equal(t, tc.wantListID, got.ListID)
		})
	}
}

// TestNewTaskCmd_Registration verifies that task command and alias register on root.
func TestNewTaskCmd_Registration(t *testing.T) {
	root := New()
	root.SetOut(os.Stdout)

	// Walk the registered commands to find "task".
	var taskCmdFound bool
	var aliasFound bool
	for _, cmd := range root.Commands() {
		if cmd.Name() == "task" {
			taskCmdFound = true
			aliasFound = cmd.HasAlias("tasks")
			break
		}
	}
	assert.True(t, taskCmdFound, "expected 'task' command to be registered on root")
	assert.True(t, aliasFound, "expected 'task' command to have alias 'tasks'")
}
