// Package cli — browse sub-command (launches the TUI).
package cli

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/pecigonzalo/clicktui/internal/app"
	"github.com/pecigonzalo/clicktui/internal/auth"
	"github.com/pecigonzalo/clicktui/internal/clickup"
	"github.com/pecigonzalo/clicktui/internal/config"
	"github.com/pecigonzalo/clicktui/internal/tui"
)

func newBrowseCmd() *cobra.Command {
	var workspaceFlag, spaceFlag, listFlag string

	cmd := &cobra.Command{
		Use:   "browse",
		Short: "Launch the TUI to browse your ClickUp workspace",
		RunE: func(cmd *cobra.Command, _ []string) error {
			store := auth.NewKeyringStore()
			provider := auth.NewPersonalTokenProvider(profileFlag, store)
			client := clickup.New(provider)

			logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
				Level: slog.LevelWarn,
			}))

			hierarchySvc := app.NewHierarchyService(client)
			taskSvc := app.NewTaskService(client)
			uiStateSvc := app.NewUIStateService()

			resolvedProfile, opts := resolveLaunchOptions(profileFlag, workspaceFlag, spaceFlag, listFlag)

			// Initialise icon preset before building any TUI components.
			if cfg, err := config.Load(); err == nil {
				tui.InitIcons(cfg.NerdFontEnabled())
			}

			tuiApp := tui.New(hierarchySvc, taskSvc, uiStateSvc, resolvedProfile, logger, opts)
			return tuiApp.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVar(&workspaceFlag, "workspace", "", "workspace (team) ID to navigate to on launch")
	cmd.Flags().StringVar(&spaceFlag, "space", "", "space ID to navigate to on launch (requires --workspace)")
	cmd.Flags().StringVar(&listFlag, "list", "", "list ID to load tasks for on launch")

	return cmd
}

// resolveLaunchOptions determines the active profile name and merges CLI flags
// over profile config values. The --profile flag takes precedence over
// active_profile in the config file. CLI flags take precedence over profile
// defaults for workspace/space/list IDs.
//
// It returns the resolved profile name alongside the resolved LaunchOptions.
func resolveLaunchOptions(profile, workspaceFlag, spaceFlag, listFlag string) (string, tui.LaunchOptions) {
	var opts tui.LaunchOptions

	// Resolve the effective profile name: start with the provided name (which
	// is already the --profile flag value, defaulting to "default"). Then
	// check whether the config has an active_profile that should override it
	// only when the flag was not explicitly set (i.e. still "default").
	resolvedProfile := profile
	if cfg, err := config.Load(); err == nil {
		// If the caller supplied the default sentinel value, let active_profile
		// in the config take precedence.
		if resolvedProfile == config.DefaultProfile() && cfg.ActiveProfile != "" {
			resolvedProfile = cfg.ActiveProfile
		}
		// Load workspace/space/list defaults from the resolved profile.
		if p, err := cfg.Profile(resolvedProfile); err == nil {
			opts.WorkspaceID = p.WorkspaceID
			opts.SpaceID = p.SpaceID
			opts.ListID = p.ListID
		}
	}

	// CLI flags override profile values.
	if workspaceFlag != "" {
		opts.WorkspaceID = workspaceFlag
	}
	if spaceFlag != "" {
		opts.SpaceID = spaceFlag
	}
	if listFlag != "" {
		opts.ListID = listFlag
	}

	// SpaceID without WorkspaceID is meaningless — clear it.
	if opts.WorkspaceID == "" {
		opts.SpaceID = ""
	}

	return resolvedProfile, opts
}
