// Package cli — task update sub-command.
package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/pecigonzalo/clicktui/internal/app"
)

// taskUpdater is the subset of app.TaskService required by the update command.
// Extracted as an interface so tests can inject a fake without building
// the full service.
type taskUpdater interface {
	UpdateTask(ctx context.Context, taskID string, input app.UpdateTaskInput) error
}

// newTaskUpdateCmd returns the "task update" (alias: "edit") sub-command.
// It updates writable scalar fields of an existing task and prints a
// confirmation on success.
//
// At least one update flag must be specified; the command returns an error
// when no fields would change.  --clear-description and --description are
// mutually exclusive, as are --clear-due and --due.
func newTaskUpdateCmd() *cobra.Command {
	var (
		nameFlag      string
		descFlag      string
		clearDescFlag bool
		dueFlag       string
		clearDueFlag  bool
		priorityFlag  string
	)

	cmd := &cobra.Command{
		Use:     "update TASK_ID",
		Aliases: []string{"edit"},
		Short:   "Update fields of a ClickUp task",
		Long: `Update one or more writable fields of a ClickUp task.

At least one flag must be provided. Pointer fields (name, description, due,
priority) are only sent when the corresponding flag is present, leaving
unspecified fields unchanged.

Examples:
  clicktui task update abc123 --name "New title"
  clicktui task update abc123 --due 2026-06-01 --priority high
  clicktui task update abc123 --clear-due
  clicktui task update abc123 --description "Updated notes" --priority urgent`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]

			// Enforce mutual exclusions.
			if cmd.Flags().Changed("description") && cmd.Flags().Changed("clear-description") {
				return fmt.Errorf("--description and --clear-description are mutually exclusive")
			}
			if cmd.Flags().Changed("due") && cmd.Flags().Changed("clear-due") {
				return fmt.Errorf("--due and --clear-due are mutually exclusive")
			}

			var input app.UpdateTaskInput

			if cmd.Flags().Changed("name") {
				v := nameFlag
				input.Name = &v
			}

			switch {
			case cmd.Flags().Changed("description"):
				v := descFlag
				input.Description = &v
			case cmd.Flags().Changed("clear-description"):
				empty := ""
				input.Description = &empty
			}

			switch {
			case cmd.Flags().Changed("due"):
				ms, err := parseDueDate(dueFlag)
				if err != nil {
					return err
				}
				input.DueDate = &ms
			case cmd.Flags().Changed("clear-due"):
				empty := ""
				input.DueDate = &empty
			}

			if cmd.Flags().Changed("priority") {
				p, err := parsePriority(priorityFlag)
				if err != nil {
					return err
				}
				input.Priority = &p
			}

			// Require at least one field to be updated.
			if input.Name == nil && input.Description == nil &&
				input.DueDate == nil && input.Priority == nil {
				return fmt.Errorf("no update fields specified")
			}

			profileChanged := cmd.Root().PersistentFlags().Changed("profile")
			defaults := ResolveProfile(profileFlag, profileChanged)

			rt := BuildRuntime(defaults.Profile)
			return runTaskUpdate(cmd.Context(), rt.TaskSvc, cmd, taskID, input)
		},
	}

	cmd.Flags().StringVar(&nameFlag, "name", "", "new task name")
	cmd.Flags().StringVar(&descFlag, "description", "", "new task description")
	cmd.Flags().BoolVar(&clearDescFlag, "clear-description", false, "clear the task description")
	cmd.Flags().StringVar(&dueFlag, "due", "", "new due date in YYYY-MM-DD format")
	cmd.Flags().BoolVar(&clearDueFlag, "clear-due", false, "clear the due date")
	cmd.Flags().StringVar(&priorityFlag, "priority", "", "new priority: urgent, high, normal, low, or none")

	return cmd
}

// runTaskUpdate applies the given update input and writes a confirmation to
// cmd.OutOrStdout().  Extracted so tests can inject a fake taskUpdater without
// wiring the full auth stack.
func runTaskUpdate(
	ctx context.Context,
	svc taskUpdater,
	cmd *cobra.Command,
	taskID string,
	input app.UpdateTaskInput,
) error {
	mode, err := resolveOutputMode(cmd)
	if err != nil {
		return err
	}

	if err := svc.UpdateTask(ctx, taskID, input); err != nil {
		return fmt.Errorf("update task: %w", err)
	}
	return renderTaskMutation(cmd, mode, fmt.Sprintf("Task %s updated.", taskID), taskID)
}
