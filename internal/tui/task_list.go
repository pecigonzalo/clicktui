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
	tuiApp    *App
	tasks     []app.TaskSummary
	currentID string
	listName  string
	page      int
	isLoading bool
}

// NewTaskListPane creates an empty task list table.
func NewTaskListPane(a *App) *TaskListPane {
	table := tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0)
	table.SetBorder(true)

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
	tlp.showLoading()

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
				tlp.tasks = tasks
			} else {
				tlp.tasks = append(tlp.tasks, tasks...)
			}
			tlp.render()
			if len(tasks) == 0 && tlp.page == 0 {
				tlp.tuiApp.footer.SetStatusReady(fmt.Sprintf("No tasks in %s", tlp.listName))
			} else {
				tlp.tuiApp.footer.SetStatusReady(fmt.Sprintf("%d tasks in %s", len(tlp.tasks), tlp.listName))
			}
		})
	}()
}

func (tlp *TaskListPane) render() {
	tlp.Clear()
	tlp.SetTitle(fmt.Sprintf(" %s ", tlp.listName))

	// Header row.
	headers := []struct {
		text      string
		expansion int
	}{
		{"STATUS", 2},
		{"TASK", 5},
		{"PRIORITY", 2},
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
		tlp.SetCell(row, 0, tview.NewTableCell(t.Status).
			SetTextColor(ColorBadgeStatus).
			SetExpansion(2))
		tlp.SetCell(row, 1, tview.NewTableCell(t.Name).
			SetTextColor(ColorText).
			SetExpansion(5))
		tlp.SetCell(row, 2, tview.NewTableCell(t.Priority).
			SetTextColor(priorityColor(t.Priority)).
			SetExpansion(2))
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

func (tlp *TaskListPane) showLoading() {
	tlp.Clear()
	tlp.SetCell(0, 0, tview.NewTableCell(loadingText("Loading tasks…")).
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
}

func (tlp *TaskListPane) inputHandler(event *tcell.EventKey) *tcell.EventKey {
	switch {
	case event.Key() == tcell.KeyRune && event.Rune() == 'n':
		// Load next page.
		tlp.page++
		tlp.fetchPage()
		return nil
	case event.Key() == tcell.KeyRune && event.Rune() == 's':
		// Trigger status picker for the currently selected task.
		row, _ := tlp.GetSelection()
		idx := row - 1
		if idx >= 0 && idx < len(tlp.tasks) {
			tlp.tuiApp.taskDetail.LoadDetail(tlp.tasks[idx].ID)
			// Open the detail pane and let the user press s there.
			tlp.tuiApp.tviewApp.SetFocus(tlp.tuiApp.taskDetail.TextView)
		}
		return nil
	}
	return event
}

// refreshCurrentTask updates the status column for a task in the list without
// a full reload.  Must be called from the UI goroutine.
func (tlp *TaskListPane) refreshCurrentTask(taskID, newStatus string) {
	for i, t := range tlp.tasks {
		if t.ID == taskID {
			tlp.tasks[i].Status = newStatus
			row := i + 1 // header occupies row 0
			tlp.SetCell(row, 0, tview.NewTableCell(newStatus).
				SetTextColor(ColorBadgeStatus).
				SetExpansion(2))
			return
		}
	}
}
