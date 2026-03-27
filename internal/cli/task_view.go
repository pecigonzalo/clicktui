// Package cli — task view sub-command.
package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/pecigonzalo/clicktui/internal/app"
)

// newTaskViewCmd returns the "task view" (aliases: "get", "show") sub-command.
// It takes exactly one positional argument (the task ID) and prints a
// labelled key-value block to cmd.OutOrStdout().
func newTaskViewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "view TASK_ID",
		Aliases: []string{"get", "show"},
		Short:   "Display full details for a task",
		Long: `Print a labelled detail block for a single ClickUp task.

Examples:
  clicktui task view abc123
  clicktui task get  abc123
  clicktui task show abc123`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]

			profileChanged := cmd.Root().PersistentFlags().Changed("profile")
			defaults := ResolveProfile(profileFlag, profileChanged)

			rt := BuildRuntime(defaults.Profile)
			return runTaskView(cmd.Context(), rt.TaskSvc, cmd, taskID)
		},
	}
	return cmd
}

// runTaskView fetches and renders the detail block for taskID.
// Extracted so tests can inject a fake without the full auth stack.
func runTaskView(ctx context.Context, svc taskViewer, cmd *cobra.Command, taskID string) error {
	mode, err := resolveOutputMode(cmd)
	if err != nil {
		return err
	}

	detail, err := svc.LoadTaskDetail(ctx, taskID)
	if err != nil {
		return fmt.Errorf("load task detail: %w", err)
	}

	return renderTaskDetail(cmd, mode, detail)
}

// taskViewer is the subset of app.TaskService used by the view command.
type taskViewer interface {
	LoadTaskDetail(ctx context.Context, taskID string) (*app.TaskDetail, error)
}
