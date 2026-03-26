// Package cli — task view sub-command.
package cli

import (
	"context"
	"fmt"
	"strings"

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
	detail, err := svc.LoadTaskDetail(ctx, taskID)
	if err != nil {
		return fmt.Errorf("load task detail: %w", err)
	}

	out := cmd.OutOrStdout()

	writeField := func(label, value string) error {
		if value == "" {
			value = "-"
		}
		_, err := fmt.Fprintf(out, "%-12s %s\n", label+":", value)
		return err
	}

	if err := writeField("ID", detail.ID); err != nil {
		return err
	}
	if detail.CustomID != "" {
		if err := writeField("Custom ID", detail.CustomID); err != nil {
			return err
		}
	}
	if err := writeField("Name", detail.Name); err != nil {
		return err
	}
	if err := writeField("Status", detail.Status); err != nil {
		return err
	}
	if err := writeField("Priority", detail.Priority); err != nil {
		return err
	}
	if err := writeField("Due", detail.DueDate); err != nil {
		return err
	}
	if len(detail.Assignees) > 0 {
		if err := writeField("Assignees", strings.Join(detail.Assignees, ", ")); err != nil {
			return err
		}
	} else {
		if err := writeField("Assignees", ""); err != nil {
			return err
		}
	}
	if len(detail.Tags) > 0 {
		if err := writeField("Tags", strings.Join(detail.Tags, ", ")); err != nil {
			return err
		}
	} else {
		if err := writeField("Tags", ""); err != nil {
			return err
		}
	}
	if err := writeField("URL", detail.URL); err != nil {
		return err
	}
	if err := writeField("List", detail.List); err != nil {
		return err
	}
	if err := writeField("Folder", detail.Folder); err != nil {
		return err
	}
	if err := writeField("Space", detail.Space); err != nil {
		return err
	}

	// Description (may be multi-line).
	if _, err := fmt.Fprintln(out, "Description:"); err != nil {
		return err
	}
	if detail.Description == "" {
		if _, err := fmt.Fprintln(out, "  -"); err != nil {
			return err
		}
	} else {
		for line := range strings.SplitSeq(detail.Description, "\n") {
			if _, err := fmt.Fprintf(out, "  %s\n", line); err != nil {
				return err
			}
		}
	}

	// Subtasks.
	if len(detail.Subtasks) > 0 {
		if _, err := fmt.Fprintln(out, "Subtasks:"); err != nil {
			return err
		}
		for _, st := range detail.Subtasks {
			if _, err := fmt.Fprintf(out, "  ↳ [%s]  %s (%s)\n", st.Status, st.Name, st.ID); err != nil {
				return err
			}
		}
	}

	return nil
}

// taskViewer is the subset of app.TaskService used by the view command.
type taskViewer interface {
	LoadTaskDetail(ctx context.Context, taskID string) (*app.TaskDetail, error)
}
