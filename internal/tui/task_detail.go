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
	tv.SetBorder(true)
	tv.SetText(emptyText("Select a task to view details"))

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
	td.SetText(loadingText("Loading task details…"))
	td.tuiApp.setStatusLoading("Loading task %s…", taskID)

	ctx := context.Background()
	go func() {
		detail, err := td.tuiApp.tasks.LoadTaskDetail(ctx, taskID)
		td.tuiApp.tviewApp.QueueUpdateDraw(func() {
			if err != nil {
				td.tuiApp.logger.Error("load task detail", "task", taskID, "error", err)
				td.tuiApp.setError("load task detail: %v", err)
				td.SetText(errorText(fmt.Sprintf("load task detail: %v", err)))
				return
			}
			td.taskID = detail.ID
			td.listID = detail.ListID
			td.taskName = detail.Name
			td.render(detail)
			td.tuiApp.footer.SetStatusReady(fmt.Sprintf("Viewing: %s", detail.Name))
		})
	}()
}

// openStatusPicker loads available statuses from the list and displays a modal
// selection list.  Must be called from the UI goroutine.
func (td *TaskDetailPane) openStatusPicker() {
	td.tuiApp.setStatusLoading("Loading statuses…")
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
		SetTitle(" Update Status ").
		SetBorderColor(ColorBorderFocused).
		SetTitleColor(ColorTitleFocused)
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
			td.tuiApp.footer.SetStatusReady("Status update cancelled")
			return nil
		}
		return event
	})

	// Hint in footer while modal is open.
	td.tuiApp.footer.SetHelp("Enter:select", "Esc:cancel")

	// Centre the modal: fixed width/height over the main content.
	modal := centreModal(list, 44, len(statuses)+4)
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
	td.tuiApp.setStatusLoading("Updating status to %q…", status)

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
			td.tuiApp.footer.SetStatusReady(fmt.Sprintf("Status → %q", status))

			// Refresh the task list so the status column reflects the change.
			td.tuiApp.taskList.refreshCurrentTask(taskID, detail.Status)

			// Restore default help keys.
			td.tuiApp.footer.SetHelp(
				"Tab:next pane",
				"Shift+Tab:prev pane",
				"Enter:select",
				"s:update status",
				"q:quit",
			)
		})
	}()
}

// label returns a right-padded label for the detail pane field grid.
func label(s string) string {
	const width = 10
	if len(s) < width {
		s += strings.Repeat(" ", width-len(s))
	}
	return s
}

func (td *TaskDetailPane) render(d *app.TaskDetail) {
	var b strings.Builder

	// ── Title block ─────────────────────────────────────────────────────────
	fmt.Fprintf(&b, "[white::b]%s[-:-:-]\n", tview.Escape(d.Name))
	fmt.Fprintf(&b, "%s%s[-]", tagColor(ColorTextSubtle), tview.Escape(d.ID))
	if d.CustomID != "" {
		fmt.Fprintf(&b, "  %s%s[-]", tagColor(ColorTextSubtle), tview.Escape(d.CustomID))
	}
	b.WriteString("\n\n")

	// ── Core fields ──────────────────────────────────────────────────────────
	fmt.Fprintf(&b, "%s%s[-] %s%s[-]\n",
		tagColor(ColorDetailLabel), label("Status"),
		tagColor(ColorBadgeStatus), tview.Escape(d.Status))
	fmt.Fprintf(&b, "%s%s[-] %s%s[-]\n",
		tagColor(ColorDetailLabel), label("Priority"),
		tagColor(priorityColor(d.Priority)), tview.Escape(d.Priority))

	if len(d.Assignees) > 0 {
		fmt.Fprintf(&b, "%s%s[-] %s%s[-]\n",
			tagColor(ColorDetailLabel), label("Assignees"),
			tagColor(ColorDetailValue), tview.Escape(strings.Join(d.Assignees, ", ")))
	}
	if len(d.Tags) > 0 {
		fmt.Fprintf(&b, "%s%s[-] %s%s[-]\n",
			tagColor(ColorDetailLabel), label("Tags"),
			tagColor(ColorDetailValue), tview.Escape(strings.Join(d.Tags, ", ")))
	}

	// ── Dates ────────────────────────────────────────────────────────────────
	hasDates := d.DueDate != "" || d.StartDate != "" || d.DateCreated != "" || d.DateUpdated != ""
	if hasDates {
		b.WriteString("\n")
		if d.DueDate != "" {
			fmt.Fprintf(&b, "%s%s[-] %s%s[-]\n",
				tagColor(ColorDetailLabel), label("Due"),
				tagColor(ColorDetailValue), d.DueDate)
		}
		if d.StartDate != "" {
			fmt.Fprintf(&b, "%s%s[-] %s%s[-]\n",
				tagColor(ColorDetailLabel), label("Start"),
				tagColor(ColorDetailValue), d.StartDate)
		}
		if d.DateCreated != "" {
			fmt.Fprintf(&b, "%s%s[-] %s%s[-]\n",
				tagColor(ColorDetailLabel), label("Created"),
				tagColor(ColorDetailValue), d.DateCreated)
		}
		if d.DateUpdated != "" {
			fmt.Fprintf(&b, "%s%s[-] %s%s[-]\n",
				tagColor(ColorDetailLabel), label("Updated"),
				tagColor(ColorDetailValue), d.DateUpdated)
		}
	}

	// ── Location ─────────────────────────────────────────────────────────────
	b.WriteString("\n")
	fmt.Fprintf(&b, "%s%s[-] %s%s[-]\n",
		tagColor(ColorDetailLabel), label("Space"),
		tagColor(ColorDetailValue), tview.Escape(d.Space))
	fmt.Fprintf(&b, "%s%s[-] %s%s[-]\n",
		tagColor(ColorDetailLabel), label("Folder"),
		tagColor(ColorDetailValue), tview.Escape(d.Folder))
	fmt.Fprintf(&b, "%s%s[-] %s%s[-]\n",
		tagColor(ColorDetailLabel), label("List"),
		tagColor(ColorDetailValue), tview.Escape(d.List))
	if d.Parent != "" {
		fmt.Fprintf(&b, "%s%s[-] %s%s[-]\n",
			tagColor(ColorDetailLabel), label("Parent"),
			tagColor(ColorDetailValue), d.Parent)
	}

	// ── URL ──────────────────────────────────────────────────────────────────
	if d.URL != "" {
		fmt.Fprintf(&b, "\n%s%s[-]\n", tagColor(ColorTextSubtle), d.URL)
	}

	// ── Description ──────────────────────────────────────────────────────────
	if d.Description != "" {
		fmt.Fprintf(&b, "\n%s── Description ──[-]\n%s%s[-]\n",
			tagColor(ColorDetailLabel),
			tagColor(ColorDetailValue),
			tview.Escape(d.Description))
	}

	// ── Footer hint ──────────────────────────────────────────────────────────
	fmt.Fprintf(&b, "\n%s[s] update status[-]", tagColor(ColorTextSubtle))

	td.SetText(b.String())
	td.ScrollToBeginning()
}
