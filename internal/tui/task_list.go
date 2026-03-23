// Package tui — task list pane.
package tui

import (
	"context"
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/pecigonzalo/clicktui/internal/app"
)

// TaskListPane displays tasks for a selected list in a table.
type TaskListPane struct {
	*tview.Table
	tuiApp      *App
	tasks       []app.TaskSummary // currently displayed tasks (may be filtered)
	allTasks    []app.TaskSummary // unfiltered full task set for the current list
	activeQuery *app.TaskQuery    // non-nil when a filter is active
	currentID   string
	listName    string
	page        int
	isLoading   bool
	styler      *PaneStyler // set by app.go after construction
}

// NewTaskListPane creates an empty task list table.
func NewTaskListPane(a *App) *TaskListPane {
	table := tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0)
	table.SetBorder(true)

	// Selection bar: white text on blue background for clarity.
	table.SetSelectedStyle(tcell.StyleDefault.
		Foreground(tcell.ColorBlack).
		Background(ColorBorderFocused).
		Attributes(tcell.AttrBold))

	tlp := &TaskListPane{
		Table:  table,
		tuiApp: a,
	}

	tlp.showEmpty("Select a list to view tasks")
	table.SetSelectedFunc(tlp.onSelected)
	table.SetInputCapture(tlp.inputHandler)
	return tlp
}

// LoadTasks fetches and displays tasks for the given list.
func (tlp *TaskListPane) LoadTasks(listID, listName string) {
	tlp.currentID = listID
	tlp.listName = listName
	tlp.page = 0
	tlp.fetchPage()
}

func (tlp *TaskListPane) fetchPage() {
	if tlp.isLoading {
		return
	}
	tlp.isLoading = true
	tlp.tuiApp.setStatusLoading("Loading tasks for %s (page %d)…", tlp.listName, tlp.page)

	ctx := context.Background()
	go func() {
		tasks, err := tlp.tuiApp.tasks.LoadTasks(ctx, tlp.currentID, tlp.page)
		tlp.tuiApp.tviewApp.QueueUpdateDraw(func() {
			tlp.isLoading = false
			if err != nil {
				tlp.tuiApp.logger.Error("load tasks", "list", tlp.currentID, "error", err)
				tlp.tuiApp.setError("load tasks: %v", err)
				tlp.showEmpty(fmt.Sprintf("Error loading tasks: %v", err))
				return
			}
			if tlp.page == 0 {
				tlp.allTasks = tasks
			} else {
				// Each page is already ordered by orderByParent in LoadTasks.
				// Cross-page parent/child splits are not re-ordered here — a
				// rare edge case since the ClickUp API returns subtasks adjacent
				// to their parents in practice. Known limitation for v1.
				tlp.allTasks = append(tlp.allTasks, tasks...)
			}
			tlp.reapplyFilter()

			// Auto-load next page if this page was full (100 = ClickUp API page size).
			if len(tasks) == 100 {
				tlp.page++
				tlp.isLoading = false // reset so fetchPage() can proceed
				tlp.fetchPage()
				return
			}

			// All pages loaded — show final status.
			if len(tlp.allTasks) == 0 {
				tlp.tuiApp.footer.SetStatusReady(fmt.Sprintf("No tasks in %s", tlp.listName))
			} else {
				tlp.tuiApp.footer.SetStatusReady(fmt.Sprintf("%d tasks in %s", len(tlp.tasks), tlp.listName))
			}
		})
	}()
}

// SelectedTaskID returns the ID of the currently selected task, or "" when no
// task row is selected.
func (tlp *TaskListPane) SelectedTaskID() string {
	row, _ := tlp.GetSelection()
	idx := row - 1 // row 0 is the header
	if idx < 0 || idx >= len(tlp.tasks) {
		return ""
	}
	return tlp.tasks[idx].ID
}

func (tlp *TaskListPane) render() {
	tlp.Clear()

	// Update the pane title via styler so focus state is preserved.
	// Format: "ListName  #ListID  N tasks"
	if tlp.styler != nil {
		tlp.styler.title = fmt.Sprintf("%s  %s#%s  %d tasks[-]",
			tview.Escape(tlp.listName),
			tagColor(ColorTextMuted),
			tlp.currentID,
			len(tlp.tasks),
		)
		tlp.styler.reapply()
	}

	// Header row — narrow status, wide task name, compact priority.
	headers := []struct {
		text      string
		expansion int
	}{
		{"STATUS", 3},
		{"TASK NAME", 8},
		{"PRI", 2},
	}
	for i, h := range headers {
		tlp.SetCell(0, i, tview.NewTableCell(h.text).
			SetTextColor(ColorTableHeader).
			SetSelectable(false).
			SetExpansion(h.expansion).
			SetAttributes(tcell.AttrBold))
	}

	for i, t := range tlp.tasks {
		row := i + 1

		// Status: dot prefix adds visual weight without requiring colour columns.
		// Use the API-provided hex color when available; fall back to aqua.
		statusText := icons.StatusDot + " " + t.Status
		statusColor := statusDotColor(t.StatusColor, "")
		tlp.SetCell(row, 0, tview.NewTableCell(statusText).
			SetTextColor(statusColor).
			SetExpansion(3).
			SetMaxWidth(18))

		// Subtasks are indented and dimmed to show hierarchy at a glance.
		nameText := tview.Escape(t.Name)
		nameColor := ColorText
		if t.Parent != "" {
			nameText = "  " + icons.SubtaskPrefix + " " + nameText
			nameColor = ColorTextMuted
		}
		tlp.SetCell(row, 1, tview.NewTableCell(nameText).
			SetTextColor(nameColor).
			SetExpansion(8))

		// Priority: symbol prefix for compact at-a-glance scanning.
		priSymbol := prioritySymbol(t.Priority)
		priText := priSymbol + " " + t.Priority
		tlp.SetCell(row, 2, tview.NewTableCell(priText).
			SetTextColor(priorityColor(t.Priority)).
			SetExpansion(2).
			SetMaxWidth(12))
	}

	if len(tlp.tasks) > 0 {
		tlp.Select(1, 0)
	}
}

func (tlp *TaskListPane) showEmpty(msg string) {
	tlp.Clear()
	tlp.SetCell(0, 0, tview.NewTableCell(emptyText(msg)).
		SetSelectable(false).
		SetExpansion(1))
}

func (tlp *TaskListPane) onSelected(row, _ int) {
	idx := row - 1 // header at row 0
	if idx < 0 || idx >= len(tlp.tasks) {
		return
	}
	task := tlp.tasks[idx]
	tlp.tuiApp.taskDetail.LoadDetail(task.ID)
	tlp.tuiApp.setFocusPane(paneTaskDetail)
}

func (tlp *TaskListPane) inputHandler(event *tcell.EventKey) *tcell.EventKey {
	switch {
	case event.Key() == tcell.KeyRune && event.Rune() == 's':
		// Trigger status picker for the currently selected task.
		row, _ := tlp.GetSelection()
		idx := row - 1
		if idx >= 0 && idx < len(tlp.tasks) {
			tlp.tuiApp.taskDetail.LoadDetail(tlp.tasks[idx].ID)
			// Move focus to the detail pane so the user can press s there
			// once the detail finishes loading.
			tlp.tuiApp.setFocusPane(paneTaskDetail)
		}
		return nil
	case event.Key() == tcell.KeyRune && event.Rune() == 'r':
		if tlp.currentID != "" {
			tlp.tuiApp.tasks.InvalidateTaskList(tlp.currentID)
			tlp.allTasks = nil
			tlp.tasks = nil
			tlp.LoadTasks(tlp.currentID, tlp.listName)
			return nil
		}
	}
	return event
}

// refreshCurrentTask updates the status column for a task in the list without
// a full reload.  Must be called from the UI goroutine.
func (tlp *TaskListPane) refreshCurrentTask(taskID, newStatus, newStatusColor string) {
	// Update allTasks (the canonical unfiltered set) so restoring from filter
	// picks up the new status.
	for i, t := range tlp.allTasks {
		if t.ID == taskID {
			tlp.allTasks[i].Status = newStatus
			tlp.allTasks[i].StatusColor = newStatusColor
			break
		}
	}

	// If a filter is active, reapply it — the task may no longer match the
	// filter criteria after the status change.
	if tlp.activeQuery != nil {
		tlp.reapplyFilter()
		return
	}

	// No filter active: update the displayed tasks and the visible table cell
	// in place for a snappy update without a full re-render.
	for i, t := range tlp.tasks {
		if t.ID == taskID {
			tlp.tasks[i].Status = newStatus
			tlp.tasks[i].StatusColor = newStatusColor
			row := i + 1 // header occupies row 0
			statusText := icons.StatusDot + " " + newStatus
			statusColor := statusDotColor(newStatusColor, "")
			tlp.SetCell(row, 0, tview.NewTableCell(statusText).
				SetTextColor(statusColor).
				SetExpansion(3).
				SetMaxWidth(18))
			return
		}
	}
}

// ApplyFilter replaces the displayed tasks with the given filtered subset and
// stores the query so it can be reapplied after pagination or status changes.
// Pass nil/empty query to show all tasks (equivalent to ClearFilter). Must be
// called from the UI goroutine.
func (tlp *TaskListPane) ApplyFilter(filtered []app.TaskSummary, query app.TaskQuery) {
	if query.Empty() {
		tlp.activeQuery = nil
		tlp.tasks = tlp.allTasks
	} else {
		tlp.activeQuery = &query
		if filtered == nil {
			tlp.tasks = nil
		} else {
			tlp.tasks = filtered
		}
	}
	tlp.render()
}

// ClearFilter restores the full unfiltered task list. Must be called from the
// UI goroutine.
func (tlp *TaskListPane) ClearFilter() {
	tlp.activeQuery = nil
	tlp.tasks = tlp.allTasks
	tlp.render()
}

// reapplyFilter applies the active filter query to allTasks and re-renders.
// When no filter is active, it shows all tasks. Must be called from the UI
// goroutine.
func (tlp *TaskListPane) reapplyFilter() {
	if tlp.activeQuery == nil {
		tlp.tasks = tlp.allTasks
	} else {
		filtered := app.FilterTasks(tlp.allTasks, *tlp.activeQuery)
		if filtered == nil {
			// Query produced no matches — show empty filtered set.
			tlp.tasks = nil
		} else {
			tlp.tasks = filtered
		}
	}
	tlp.render()
}
