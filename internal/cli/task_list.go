// Package cli — task list sub-command.
package cli

import (
	"context"
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/pecigonzalo/clicktui/internal/app"
)

// newTaskListCmd returns the "task list" (alias: "ls") sub-command.
// It resolves the list ID from --list or the active profile's list_id,
// then prints a tab-aligned table of tasks to cmd.OutOrStdout().
//
// When --all is set every page (0, 1, 2 …) is fetched until an empty page
// is returned; otherwise only the page specified by --page is printed.
func newTaskListCmd() *cobra.Command {
	var listFlag string
	var pageFlag int
	var allFlag bool

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List tasks in a ClickUp list",
		Long: `Print a table of tasks for a list.

The list ID can be supplied via --list or configured as list_id in your profile.
Use --all to fetch every page of results automatically.`,
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

			rt := BuildRuntime(defaults.Profile)
			return runTaskList(cmd.Context(), rt.TaskSvc, cmd, listID, pageFlag, allFlag)
		},
	}

	cmd.Flags().StringVar(&listFlag, "list", "", "list ID to fetch tasks from")
	cmd.Flags().IntVar(&pageFlag, "page", 0, "page of results to display (0-indexed)")
	cmd.Flags().BoolVar(&allFlag, "all", false, "fetch all pages and display all tasks")

	return cmd
}

// runTaskList fetches tasks and writes the table to cmd.OutOrStdout().
// It is extracted so that tests can inject a fake TaskService without wiring
// the full auth stack.
func runTaskList(
	ctx context.Context,
	svc taskLister,
	cmd *cobra.Command,
	listID string,
	page int,
	all bool,
) error {
	var tasks []app.TaskSummary

	if all {
		for p := 0; ; p++ {
			page, err := svc.LoadTasks(ctx, listID, p)
			if err != nil {
				return fmt.Errorf("load tasks (page %d): %w", p, err)
			}
			if len(page) == 0 {
				break
			}
			tasks = append(tasks, page...)
		}
	} else {
		loaded, err := svc.LoadTasks(ctx, listID, page)
		if err != nil {
			return fmt.Errorf("load tasks: %w", err)
		}
		tasks = loaded
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	defer func() {
		_ = w.Flush() // Flush errors are non-fatal; output is already written
	}()

	if _, err := fmt.Fprintln(w, "ID\tSTATUS\tPRIORITY\tDUE\tNAME"); err != nil {
		return err
	}
	for _, t := range tasks {
		name := t.Name
		if t.Parent != "" {
			name = "  ↳ " + name
		}
		due := t.DueDate
		if due == "" {
			due = "-"
		}
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			t.ID, t.Status, t.Priority, due, name,
		); err != nil {
			return err
		}
	}
	return nil
}

// taskLister is the subset of app.TaskService used by the list command.
// Extracted as an interface so tests can inject a fake without building
// the full service.
type taskLister interface {
	LoadTasks(ctx context.Context, listID string, page int) ([]app.TaskSummary, error)
}
