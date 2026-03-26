// Package cli — shared context and dependency helpers for CLI commands.
package cli

import (
	"github.com/pecigonzalo/clicktui/internal/app"
	"github.com/pecigonzalo/clicktui/internal/auth"
	"github.com/pecigonzalo/clicktui/internal/clickup"
	"github.com/pecigonzalo/clicktui/internal/config"
)

// ProfileDefaults holds the resolved profile name and its configured default
// workspace/space/list IDs.  These values come from the config file and may be
// further overridden by explicit CLI flags in each command.
type ProfileDefaults struct {
	// Profile is the effective profile name after resolving active_profile and
	// the --profile flag.
	Profile string
	// WorkspaceID is the default workspace ID from the resolved profile, or "".
	WorkspaceID string
	// SpaceID is the default space ID from the resolved profile, or "".
	SpaceID string
	// ListID is the default list ID from the resolved profile, or "".
	ListID string
}

// ResolveProfile resolves the active profile name and loads its defaults from
// the config file.
//
// profileFlagValue is the raw value of the --profile flag (defaults to
// config.DefaultProfile()).  profileChanged must be true when the caller
// explicitly passed --profile on the command line (i.e.
// cmd.Flags().Changed("profile") == true); otherwise active_profile in the
// config file is allowed to override the flag's default value.
//
// Callers should pass profileChanged = cmd.Flags().Changed("profile") (or
// cmd.Root().PersistentFlags().Changed("profile") for subcommands) so that
// --profile default is treated as an explicit flag, not as the default sentinel.
func ResolveProfile(profileFlagValue string, profileChanged bool) ProfileDefaults {
	resolved := profileFlagValue

	cfg, err := config.Load()
	if err != nil {
		// Config unreadable — fall back to raw flag value with empty defaults.
		return ProfileDefaults{Profile: resolved}
	}

	// When the caller did NOT explicitly pass --profile, let active_profile in
	// the config take precedence over the flag's default value.
	if !profileChanged && cfg.ActiveProfile != "" {
		resolved = cfg.ActiveProfile
	}

	d := ProfileDefaults{Profile: resolved}

	if p, err := cfg.Profile(resolved); err == nil {
		d.WorkspaceID = p.WorkspaceID
		d.SpaceID = p.SpaceID
		d.ListID = p.ListID
	}

	return d
}

// CLIRuntime bundles the app-layer services needed by direct CLI task commands.
// Construct it via BuildRuntime after resolving the active profile.
type CLIRuntime struct {
	// HierarchySvc provides workspace/space/list hierarchy queries.
	HierarchySvc *app.HierarchyService
	// TaskSvc provides task read/write operations.
	TaskSvc *app.TaskService
}

// BuildRuntime constructs a CLIRuntime for the given resolved profile name.
// It creates the keyring-backed credential store, personal-token provider,
// ClickUp HTTP client, and the two app-layer services used by task commands.
//
// profile should be the Profile field returned by ResolveProfile.
func BuildRuntime(profile string) CLIRuntime {
	store := auth.NewKeyringStore()
	provider := auth.NewPersonalTokenProvider(profile, store)
	client := clickup.New(provider)
	return CLIRuntime{
		HierarchySvc: app.NewHierarchyService(client),
		TaskSvc:      app.NewTaskService(client),
	}
}
