// Package tui — task detail pane and status picker.
package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/pecigonzalo/clicktui/internal/app"
)

const pageStatusPicker = "status_picker"

// TaskDetailPane shows detailed information about a selected task.
type TaskDetailPane struct {
	*tview.TextView
	tuiApp   *App
	taskID   string
	listID   string
	taskName string
}

// NewTaskDetailPane creates an empty task detail pane.
func NewTaskDetailPane(a *App) *TaskDetailPane {
	tv := tview.NewTextView().
		SetDynamicColors(true).
		SetWordWrap(true).
		SetScrollable(true)
	tv.SetBorder(true).SetTitle(" Task Details ").SetBorderColor(tcell.ColorDarkCyan)
	tv.SetText("[darkgray]Select a task to view details")

	tdp := &TaskDetailPane{TextView: tv, tuiApp: a}
	tv.SetInputCapture(tdp.inputHandler)
	return tdp
}

func (td *TaskDetailPane) inputHandler(event *tcell.EventKey) *tcell.EventKey {
	if event.Key() == tcell.KeyRune && event.Rune() == 's' {
		if td.taskID != "" {
			td.openStatusPicker()
			return nil
		}
	}
	return event
}

// LoadDetail fetches and renders a task's full details.
func (td *TaskDetailPane) LoadDetail(taskID string) {
	td.SetText("[yellow]Loading task details...")
	td.tuiApp.setStatus("Loading task %s...", taskID)

	ctx := context.Background()
	go func() {
		detail, err := td.tuiApp.tasks.LoadTaskDetail(ctx, taskID)
		td.tuiApp.tviewApp.QueueUpdateDraw(func() {
			if err != nil {
				td.tuiApp.logger.Error("load task detail", "task", taskID, "error", err)
				td.tuiApp.setError("load task detail: %v", err)
				td.SetText(fmt.Sprintf("[red]Error: %v", err))
				return
			}
			td.taskID = detail.ID
			td.listID = detail.ListID
			td.taskName = detail.Name
			td.render(detail)
			td.tuiApp.setStatus("Viewing task %s — press s to update status", detail.ID)
		})
	}()
}

// openStatusPicker loads available statuses from the list and displays a modal
// selection list.  Must be called from the UI goroutine.
func (td *TaskDetailPane) openStatusPicker() {
	td.tuiApp.setStatus("Loading statuses...")
	taskID := td.taskID
	listID := td.listID

	ctx := context.Background()
	go func() {
		statuses, err := td.tuiApp.tasks.LoadListStatuses(ctx, listID)
		td.tuiApp.tviewApp.QueueUpdateDraw(func() {
			if err != nil {
				td.tuiApp.logger.Error("load list statuses", "list", listID, "error", err)
				td.tuiApp.setError("load statuses: %v", err)
				return
			}
			if len(statuses) == 0 {
				td.tuiApp.setError("no statuses available for this list")
				return
			}
			td.showStatusModal(taskID, statuses)
		})
	}()
}

// showStatusModal renders the status picker modal over the main layout.
// Must be called from the UI goroutine (inside QueueUpdateDraw).
func (td *TaskDetailPane) showStatusModal(taskID string, statuses []app.StatusOption) {
	list := tview.NewList()
	list.SetBorder(true).
		SetTitle(" Choose Status (Esc to cancel) ").
		SetBorderColor(tcell.ColorDarkCyan)
	list.ShowSecondaryText(false)

	for _, s := range statuses {
		// Capture loop variable for closure.
		chosen := s.Name
		list.AddItem(s.Name, "", 0, func() {
			td.tuiApp.pages.RemovePage(pageStatusPicker)
			td.tuiApp.tviewApp.SetFocus(td.tuiApp.taskDetail.TextView)
			td.applyStatusUpdate(taskID, chosen)
		})
	}

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			td.tuiApp.pages.RemovePage(pageStatusPicker)
			td.tuiApp.tviewApp.SetFocus(td.tuiApp.taskDetail.TextView)
			td.tuiApp.setStatus("Status update cancelled")
			return nil
		}
		return event
	})

	// Centre the modal: fixed width/height over the main content.
	modal := centreModal(list, 40, len(statuses)+4)
	td.tuiApp.pages.AddPage(pageStatusPicker, modal, true, true)
	td.tuiApp.tviewApp.SetFocus(list)
}

// centreModal wraps p in a Flex that centres it with the given dimensions.
func centreModal(p tview.Primitive, width, height int) tview.Primitive {
	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(p, height, 1, true).
			AddItem(nil, 0, 1, false),
			width, 1, true).
		AddItem(nil, 0, 1, false)
}

// applyStatusUpdate calls the service and refreshes the panes.
func (td *TaskDetailPane) applyStatusUpdate(taskID, status string) {
	td.tuiApp.setStatus("Updating status to %q...", status)

	ctx := context.Background()
	go func() {
		detail, err := td.tuiApp.tasks.UpdateTaskStatus(ctx, taskID, status)
		td.tuiApp.tviewApp.QueueUpdateDraw(func() {
			if err != nil {
				td.tuiApp.logger.Error("update task status", "task", taskID, "status", status, "error", err)
				td.tuiApp.setError("update status: %v", err)
				return
			}
			// Refresh detail pane.
			td.taskID = detail.ID
			td.listID = detail.ListID
			td.taskName = detail.Name
			td.render(detail)
			td.tuiApp.setStatus("Status updated to %q", status)

			// Refresh the task list so the status column reflects the change.
			td.tuiApp.taskList.refreshCurrentTask(taskID, detail.Status)
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

	fmt.Fprintf(&b, "\n[darkgray]Press s to update status[-]")

	td.SetText(b.String())
	td.ScrollToBeginning()
}
