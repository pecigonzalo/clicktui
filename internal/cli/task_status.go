// Package cli — task status sub-command.
package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pecigonzalo/clicktui/internal/app"
)

// newTaskStatusCmd returns the "task status" (alias: "set-status") sub-command.
// It validates --to against the list's available statuses before mutating and
// prints a confirmation on success.
func newTaskStatusCmd() *cobra.Command {
	var toFlag string

	cmd := &cobra.Command{
		Use:     "status TASK_ID",
		Aliases: []string{"set-status"},
		Short:   "Update the status of a task",
		Long: `Validate and update the status of a ClickUp task.

The new status must be one of the statuses available in the task's list. If the
requested status is not valid, the command prints the available options and
returns an error.

Examples:
  clicktui task status abc123 --to "in progress"
  clicktui task set-status abc123 --to done`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]

			profileChanged := cmd.Root().PersistentFlags().Changed("profile")
			defaults := ResolveProfile(profileFlag, profileChanged)

			rt := BuildRuntime(defaults.Profile)
			return runTaskStatus(cmd.Context(), rt.TaskSvc, cmd, taskID, toFlag)
		},
	}

	cmd.Flags().StringVar(&toFlag, "to", "", "new status value (required)")
	if err := cmd.MarkFlagRequired("to"); err != nil {
		// MarkFlagRequired only fails for unknown flag names; this is safe.
		panic(fmt.Sprintf("task status: unexpected MarkFlagRequired error: %v", err))
	}

	return cmd
}

// taskStatusService is the subset of app.TaskService required by the status
// command.
type taskStatusService interface {
	LoadTaskDetail(ctx context.Context, taskID string) (*app.TaskDetail, error)
	LoadListStatuses(ctx context.Context, listID string) ([]app.StatusOption, error)
	UpdateTaskStatus(ctx context.Context, taskID, status string) (*app.TaskDetail, error)
}

// runTaskStatus validates the requested status and updates the task.
// Extracted so tests can inject a fake without the full auth stack.
func runTaskStatus(
	ctx context.Context,
	svc taskStatusService,
	cmd *cobra.Command,
	taskID, newStatus string,
) error {
	mode, err := resolveOutputMode(cmd)
	if err != nil {
		return err
	}

	// Fetch the task to discover its list ID.
	detail, err := svc.LoadTaskDetail(ctx, taskID)
	if err != nil {
		return fmt.Errorf("load task detail: %w", err)
	}

	// Validate the requested status against the list's available statuses.
	statuses, err := svc.LoadListStatuses(ctx, detail.ListID)
	if err != nil {
		return fmt.Errorf("load list statuses: %w", err)
	}

	valid := statusValid(statuses, newStatus)
	if !valid {
		names := statusNames(statuses)
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "available statuses: %s\n", strings.Join(names, ", ")); err != nil {
			// Log error but continue to return the validation error
			_ = err
		}
		return fmt.Errorf("invalid status %q for list %q", newStatus, detail.ListID)
	}

	if _, err := svc.UpdateTaskStatus(ctx, taskID, newStatus); err != nil {
		return fmt.Errorf("update task status: %w", err)
	}

	return renderTaskMutation(cmd, mode, fmt.Sprintf("Task %s status updated to %q.", taskID, newStatus), taskID)
}

// statusValid reports whether name matches one of the available statuses
// (case-insensitive comparison).
func statusValid(statuses []app.StatusOption, name string) bool {
	lower := strings.ToLower(name)
	for _, s := range statuses {
		if strings.ToLower(s.Name) == lower {
			return true
		}
	}
	return false
}

// statusNames returns the Name field of each StatusOption.
func statusNames(statuses []app.StatusOption) []string {
	names := make([]string, len(statuses))
	for i, s := range statuses {
		names[i] = s.Name
	}
	return names
}
