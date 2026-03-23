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

			opts := resolveLaunchOptions(workspaceFlag, spaceFlag, listFlag)

			// Initialise icon preset before building any TUI components.
			if cfg, err := config.Load(); err == nil {
				if p, err := cfg.Active(); err == nil {
					tui.InitIcons(p.NerdFontEnabled())
				}
			}

			tuiApp := tui.New(hierarchySvc, taskSvc, logger, opts)
			return tuiApp.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVar(&workspaceFlag, "workspace", "", "workspace (team) ID to navigate to on launch")
	cmd.Flags().StringVar(&spaceFlag, "space", "", "space ID to navigate to on launch (requires --workspace)")
	cmd.Flags().StringVar(&listFlag, "list", "", "list ID to load tasks for on launch (requires --workspace and --space)")

	return cmd
}

// resolveLaunchOptions merges CLI flags over profile config values. Flags take
// precedence; when absent, values from the active profile are used.
func resolveLaunchOptions(workspaceFlag, spaceFlag, listFlag string) tui.LaunchOptions {
	var opts tui.LaunchOptions

	// Load profile defaults — errors are non-fatal; the TUI works without them.
	if cfg, err := config.Load(); err == nil {
		if p, err := cfg.Active(); err == nil {
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

	// SpaceID without WorkspaceID is meaningless — clear it and any dependent IDs.
	if opts.WorkspaceID == "" {
		opts.SpaceID = ""
		opts.ListID = ""
	}

	// ListID without SpaceID is meaningless — clear it.
	if opts.SpaceID == "" {
		opts.ListID = ""
	}

	return opts
}
