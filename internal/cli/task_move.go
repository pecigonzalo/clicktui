// Package cli — task move sub-command.
package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/pecigonzalo/clicktui/internal/app"
)

// taskMover is the subset of app.TaskService required by the move command.
// Extracted as an interface so tests can inject a fake without building
// the full service.
type taskMover interface {
	MoveTaskToList(ctx context.Context, workspaceID, taskID, listID string) (*app.TaskDetail, error)
}

// newTaskMoveCmd returns the "task move" (alias: "mv") sub-command.
// It moves a task to a different list in the same workspace.
//
// --to-list is required. --workspace is optional if workspace_id is configured
// in the active profile; otherwise the command returns an error.
func newTaskMoveCmd() *cobra.Command {
	var (
		toListFlag    string
		workspaceFlag string
	)

	cmd := &cobra.Command{
		Use:     "move TASK_ID",
		Aliases: []string{"mv"},
		Short:   "Move a task to a different list",
		Long: `Move a ClickUp task to a different list within the same workspace.

The workspace ID can be supplied via --workspace or configured as workspace_id
in your profile.

Examples:
  clicktui task move abc123 --to-list 9012345
  clicktui task move abc123 --to-list 9012345 --workspace 1234567`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]

			profileChanged := cmd.Root().PersistentFlags().Changed("profile")
			defaults := ResolveProfile(profileFlag, profileChanged)

			// Resolve workspace ID: flag overrides profile default.
			workspaceID := defaults.WorkspaceID
			if cmd.Flags().Changed("workspace") {
				workspaceID = workspaceFlag
			} else if workspaceFlag != "" && workspaceID == "" {
				workspaceID = workspaceFlag
			}
			if workspaceID == "" {
				return fmt.Errorf("no workspace ID: use --workspace or configure workspace_id in your profile")
			}

			rt := BuildRuntime(defaults.Profile)
			return runTaskMove(cmd.Context(), rt.TaskSvc, cmd, workspaceID, taskID, toListFlag)
		},
	}

	cmd.Flags().StringVar(&toListFlag, "to-list", "", "destination list ID (required)")
	cmd.Flags().StringVar(&workspaceFlag, "workspace", "", "workspace ID (overrides profile default)")

	if err := cmd.MarkFlagRequired("to-list"); err != nil {
		// MarkFlagRequired only fails for unknown flag names; this is safe.
		panic(fmt.Sprintf("task move: unexpected MarkFlagRequired error: %v", err))
	}

	return cmd
}

// runTaskMove moves a task to a new list and writes a confirmation to
// cmd.OutOrStdout().  Extracted so tests can inject a fake taskMover without
// wiring the full auth stack.
func runTaskMove(
	ctx context.Context,
	svc taskMover,
	cmd *cobra.Command,
	workspaceID, taskID, listID string,
) error {
	mode, err := resolveOutputMode(cmd)
	if err != nil {
		return err
	}

	if _, err := svc.MoveTaskToList(ctx, workspaceID, taskID, listID); err != nil {
		return fmt.Errorf("move task: %w", err)
	}
	return renderTaskMutation(cmd, mode, fmt.Sprintf("Task %s moved to list %s.", taskID, listID), taskID)
}
