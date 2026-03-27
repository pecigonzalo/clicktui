// Package cli — output rendering helpers for task commands.
package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/pecigonzalo/clicktui/internal/app"
)

type outputMode string

const (
	outputModeText outputMode = "text"
	outputModeJSON outputMode = "json"
)

// taskMutationResult is the JSON representation of a mutation result.
type taskMutationResult struct {
	OK bool   `json:"ok"`
	ID string `json:"id"`
}

// resolveOutputMode reads the "output" flag from the command (or its parents).
// Returns an error for unrecognised values. Validate BEFORE any mutation call.
func resolveOutputMode(cmd *cobra.Command) (outputMode, error) {
	val, err := cmd.Flags().GetString("output")
	if err != nil {
		// Flag not defined locally; walk up to root via persistent flags.
		val, err = cmd.Root().PersistentFlags().GetString("output")
		if err != nil {
			return outputModeText, nil
		}
	}
	switch outputMode(val) {
	case outputModeText:
		return outputModeText, nil
	case outputModeJSON:
		return outputModeJSON, nil
	default:
		return "", fmt.Errorf("unsupported output format %q; supported: text, json", val)
	}
}

// renderTaskSummaries renders a list of task summaries to cmd.OutOrStdout().
func renderTaskSummaries(cmd *cobra.Command, mode outputMode, tasks []app.TaskSummary) error {
	switch mode {
	case outputModeJSON:
		out := tasks
		if out == nil {
			out = []app.TaskSummary{}
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(out)
	default:
		return renderTaskSummariesText(cmd, tasks)
	}
}

// renderTaskSummariesText renders tasks as a tab-aligned table to cmd.OutOrStdout().
func renderTaskSummariesText(cmd *cobra.Command, tasks []app.TaskSummary) error {
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

// renderTaskDetail renders a single task detail to cmd.OutOrStdout().
func renderTaskDetail(cmd *cobra.Command, mode outputMode, detail *app.TaskDetail) error {
	switch mode {
	case outputModeJSON:
		return json.NewEncoder(cmd.OutOrStdout()).Encode(detail)
	default:
		return renderTaskDetailText(cmd, detail)
	}
}

// renderTaskDetailText renders a task detail as a labelled key-value block to cmd.OutOrStdout().
func renderTaskDetailText(cmd *cobra.Command, detail *app.TaskDetail) error {
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

// renderTaskMutation renders a mutation result (ok + id) to cmd.OutOrStdout().
func renderTaskMutation(cmd *cobra.Command, mode outputMode, message string, id string) error {
	switch mode {
	case outputModeJSON:
		return json.NewEncoder(cmd.OutOrStdout()).Encode(taskMutationResult{OK: true, ID: id})
	default:
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), message); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
		return nil
	}
}
