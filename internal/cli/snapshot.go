// Package cli — snapshot sub-command (headless TUI screen capture).
package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/pecigonzalo/clicktui/internal/app"
	"github.com/pecigonzalo/clicktui/internal/auth"
	"github.com/pecigonzalo/clicktui/internal/clickup"
	"github.com/pecigonzalo/clicktui/internal/config"
	"github.com/pecigonzalo/clicktui/internal/tui"
)

const defaultSnapshotTimeout = 30 * time.Second

func newSnapshotCmd() *cobra.Command {
	var (
		workspaceFlag string
		spaceFlag     string
		listFlag      string
		outputFlag    string
		widthFlag     int
		heightFlag    int
	)

	cmd := &cobra.Command{
		Use:   "snapshot",
		Short: "Capture a TUI snapshot to a file",
		Long: `Run the TUI headlessly on a simulated screen, wait for the initial data
load to complete, then dump the screen buffer to a plain-text file.

This is a development and debugging aid for environments without a real
terminal (e.g. CI or agent workflows).`,
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

			ctx, cancel := context.WithTimeout(cmd.Context(), defaultSnapshotTimeout)
			defer cancel()

			content, err := tuiApp.RunHeadless(ctx, widthFlag, heightFlag)
			if err != nil {
				return fmt.Errorf("snapshot: %w", err)
			}

			if err := os.WriteFile(outputFlag, []byte(content), 0o644); err != nil {
				return fmt.Errorf("write snapshot: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Snapshot written to %s\n", outputFlag)

			return nil
		},
	}

	cmd.Flags().StringVar(&workspaceFlag, "workspace", "", "workspace (team) ID to navigate to on launch")
	cmd.Flags().StringVar(&spaceFlag, "space", "", "space ID to navigate to on launch (requires --workspace)")
	cmd.Flags().StringVar(&listFlag, "list", "", "list ID to load tasks for on launch")
	cmd.Flags().StringVar(&outputFlag, "output", "snapshot.txt", "path to write the screen dump")
	cmd.Flags().IntVar(&widthFlag, "width", 220, "simulated screen width in columns")
	cmd.Flags().IntVar(&heightFlag, "height", 50, "simulated screen height in rows")

	return cmd
}
