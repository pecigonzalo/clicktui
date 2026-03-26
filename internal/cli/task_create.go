// Package cli — task create sub-command.
package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/pecigonzalo/clicktui/internal/app"
)

// taskCreator is the subset of app.TaskService required by the create command.
// Extracted as an interface so tests can inject a fake without building
// the full service.
type taskCreator interface {
	CreateTask(ctx context.Context, listID string, input app.CreateTaskInput) (string, error)
}

// newTaskCreateCmd returns the "task create" (alias: "new") sub-command.
// It creates a new task in the given list and prints the resulting task ID.
//
// --name is required. --list is optional if list_id is configured in the
// active profile; otherwise the command returns an error.
func newTaskCreateCmd() *cobra.Command {
	var (
		nameFlag     string
		listFlag     string
		statusFlag   string
		descFlag     string
		dueFlag      string
		priorityFlag string
	)

	cmd := &cobra.Command{
		Use:     "create",
		Aliases: []string{"new"},
		Short:   "Create a new ClickUp task",
		Long: `Create a new task in a ClickUp list.

--name is required. The list ID can be supplied via --list or configured as
list_id in your profile.

Examples:
  clicktui task create --name "Fix login bug" --list 9012345
  clicktui task create --name "Write tests" --list 9012345 --priority high --due 2026-05-01`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			profileChanged := cmd.Root().PersistentFlags().Changed("profile")
			defaults := ResolveProfile(profileFlag, profileChanged)

			// Resolve list ID: flag overrides profile default.
			listID := defaults.ListID
			if cmd.Flags().Changed("list") {
				listID = listFlag
			} else if listFlag != "" && listID == "" {
				listID = listFlag
			}
			if listID == "" {
				return fmt.Errorf("no list ID: use --list or configure list_id in your profile")
			}

			dueMS, err := parseDueDate(dueFlag)
			if err != nil {
				return err
			}
			priority, err := parsePriority(priorityFlag)
			if err != nil {
				return err
			}

			rt := BuildRuntime(defaults.Profile)
			return runTaskCreate(cmd.Context(), rt.TaskSvc, cmd, listID, app.CreateTaskInput{
				Name:        nameFlag,
				Status:      statusFlag,
				Description: descFlag,
				DueDate:     dueMS,
				Priority:    priority,
			})
		},
	}

	cmd.Flags().StringVar(&nameFlag, "name", "", "task name (required)")
	cmd.Flags().StringVar(&listFlag, "list", "", "list ID to create the task in")
	cmd.Flags().StringVar(&statusFlag, "status", "", "initial status")
	cmd.Flags().StringVar(&descFlag, "description", "", "task description")
	cmd.Flags().StringVar(&dueFlag, "due", "", "due date in YYYY-MM-DD format")
	cmd.Flags().StringVar(&priorityFlag, "priority", "", "priority: urgent, high, normal, low, or none")

	if err := cmd.MarkFlagRequired("name"); err != nil {
		// MarkFlagRequired only fails for unknown flag names; this is safe.
		panic(fmt.Sprintf("task create: unexpected MarkFlagRequired error: %v", err))
	}

	return cmd
}

// runTaskCreate creates a task and writes a confirmation to cmd.OutOrStdout().
// Extracted so tests can inject a fake taskCreator without wiring the full
// auth stack.
func runTaskCreate(
	ctx context.Context,
	svc taskCreator,
	cmd *cobra.Command,
	listID string,
	input app.CreateTaskInput,
) error {
	taskID, err := svc.CreateTask(ctx, listID, input)
	if err != nil {
		return fmt.Errorf("create task: %w", err)
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Created task %s.\n", taskID); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}
