// Package cli — task command group (direct task management without the TUI).
package cli

import (
	"github.com/spf13/cobra"
)

// newTaskCmd returns the "task" command group with list, view, status, create,
// move, and update subcommands registered.
func newTaskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "task",
		Aliases: []string{"tasks"},
		Short:   "Manage ClickUp tasks from the command line",
		Long: `Manage ClickUp tasks without launching the TUI.

Use the subcommands to list, view, and modify tasks directly from your terminal.

Examples:
  clicktui task list --list 9012345
  clicktui task view TASK_ID
  clicktui task create --name "Fix login bug" --list 9012345
  clicktui task move TASK_ID --to-list 9012345
  clicktui task update TASK_ID --name "New title"`,
	}

	cmd.AddCommand(
		newTaskListCmd(),
		newTaskViewCmd(),
		newTaskStatusCmd(),
		newTaskCreateCmd(),
		newTaskMoveCmd(),
		newTaskUpdateCmd(),
	)

	return cmd
}
