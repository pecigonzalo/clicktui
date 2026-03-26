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

			profileChanged := cmd.Root().PersistentFlags().Changed("profile")
			resolvedProfile, opts := resolveLaunchOptions(profileFlag, profileChanged, workspaceFlag, spaceFlag, listFlag)

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
// over profile config values. It is a thin wrapper around ResolveProfile that
// also merges workspace/space/list flags into the TUI LaunchOptions.
//
// profileChanged must be true when --profile was explicitly provided on the
// command line (use cmd.Root().PersistentFlags().Changed("profile")).
func resolveLaunchOptions(profile string, profileChanged bool, workspaceFlag, spaceFlag, listFlag string) (string, tui.LaunchOptions) {
	defaults := ResolveProfile(profile, profileChanged)

	opts := tui.LaunchOptions{
		WorkspaceID: defaults.WorkspaceID,
		SpaceID:     defaults.SpaceID,
		ListID:      defaults.ListID,
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

	return defaults.Profile, opts
}
