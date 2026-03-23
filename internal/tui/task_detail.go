// Package tui — task detail pane.
package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/pecigonzalo/clicktui/internal/app"
)

// TaskDetailPane shows detailed information about a selected task.
type TaskDetailPane struct {
	*tview.TextView
}

// NewTaskDetailPane creates an empty task detail pane.
func NewTaskDetailPane() *TaskDetailPane {
	tv := tview.NewTextView().
		SetDynamicColors(true).
		SetWordWrap(true).
		SetScrollable(true)
	tv.SetBorder(true).SetTitle(" Task Details ").SetBorderColor(tcell.ColorDarkCyan)
	tv.SetText("[darkgray]Select a task to view details")

	return &TaskDetailPane{TextView: tv}
}

// LoadDetail fetches and renders a task's full details.
func (td *TaskDetailPane) LoadDetail(a *App, taskID string) {
	td.SetText("[yellow]Loading task details...")
	a.setStatus("Loading task %s...", taskID)

	ctx := context.Background()
	go func() {
		detail, err := a.tasks.LoadTaskDetail(ctx, taskID)
		a.tviewApp.QueueUpdateDraw(func() {
			if err != nil {
				a.logger.Error("load task detail", "task", taskID, "error", err)
				a.setError("load task detail: %v", err)
				td.SetText(fmt.Sprintf("[red]Error: %v", err))
				return
			}
			td.render(detail)
			a.setStatus("Viewing task %s", detail.ID)
		})
	}()
}

func (td *TaskDetailPane) render(d *app.TaskDetail) {
	var b strings.Builder

	fmt.Fprintf(&b, "[yellow]%s[-]\n", tview.Escape(d.Name))
	fmt.Fprintf(&b, "[darkgray]ID: %s[-]", d.ID)
	if d.CustomID != "" {
		fmt.Fprintf(&b, "  [darkgray]Custom: %s[-]", d.CustomID)
	}
	b.WriteString("\n\n")

	fmt.Fprintf(&b, "[white]Status:    [aqua]%s[-]\n", tview.Escape(d.Status))
	fmt.Fprintf(&b, "[white]Priority:  [green]%s[-]\n", tview.Escape(d.Priority))

	if len(d.Assignees) > 0 {
		fmt.Fprintf(&b, "[white]Assignees: [white]%s[-]\n", tview.Escape(strings.Join(d.Assignees, ", ")))
	}
	if len(d.Tags) > 0 {
		fmt.Fprintf(&b, "[white]Tags:      [white]%s[-]\n", tview.Escape(strings.Join(d.Tags, ", ")))
	}

	b.WriteString("\n")
	if d.DueDate != "" {
		fmt.Fprintf(&b, "[white]Due:       %s\n", d.DueDate)
	}
	if d.StartDate != "" {
		fmt.Fprintf(&b, "[white]Start:     %s\n", d.StartDate)
	}
	if d.DateCreated != "" {
		fmt.Fprintf(&b, "[white]Created:   %s\n", d.DateCreated)
	}
	if d.DateUpdated != "" {
		fmt.Fprintf(&b, "[white]Updated:   %s\n", d.DateUpdated)
	}

	b.WriteString("\n")
	fmt.Fprintf(&b, "[white]Space:     %s\n", tview.Escape(d.Space))
	fmt.Fprintf(&b, "[white]Folder:    %s\n", tview.Escape(d.Folder))
	fmt.Fprintf(&b, "[white]List:      %s\n", tview.Escape(d.List))
	if d.Parent != "" {
		fmt.Fprintf(&b, "[white]Parent:    %s\n", d.Parent)
	}

	if d.URL != "" {
		fmt.Fprintf(&b, "\n[darkgray]%s[-]\n", d.URL)
	}

	if d.Description != "" {
		fmt.Fprintf(&b, "\n[white]─── Description ───[-]\n%s\n", tview.Escape(d.Description))
	}

	td.SetText(b.String())
	td.ScrollToBeginning()
}
