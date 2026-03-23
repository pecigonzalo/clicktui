// Package cli — browse sub-command (launches the TUI).
package cli

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/pecigonzalo/clicktui/internal/app"
	"github.com/pecigonzalo/clicktui/internal/auth"
	"github.com/pecigonzalo/clicktui/internal/clickup"
	"github.com/pecigonzalo/clicktui/internal/tui"
)

func newBrowseCmd() *cobra.Command {
	return &cobra.Command{
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

			tuiApp := tui.New(hierarchySvc, taskSvc, logger)
			return tuiApp.Run(cmd.Context())
		},
	}
}
