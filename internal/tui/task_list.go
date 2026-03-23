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
	table.SetBorder(true).SetTitle(" Tasks ").SetBorderColor(tcell.ColorDarkCyan)

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
	tlp.tuiApp.setStatus("Loading tasks for %s (page %d)...", tlp.listName, tlp.page)

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
				tlp.tuiApp.setStatus("No tasks in %s", tlp.listName)
			} else {
				tlp.tuiApp.setStatus("%d tasks loaded for %s", len(tlp.tasks), tlp.listName)
			}
		})
	}()
}

func (tlp *TaskListPane) render() {
	tlp.Clear()
	tlp.SetTitle(fmt.Sprintf(" Tasks — %s ", tlp.listName))

	// Header row.
	headers := []string{"Status", "Name", "Priority"}
	for i, h := range headers {
		cell := tview.NewTableCell(h).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetExpansion(1)
		if i == 1 {
			cell.SetExpansion(3)
		}
		tlp.SetCell(0, i, cell)
	}

	for i, t := range tlp.tasks {
		row := i + 1
		tlp.SetCell(row, 0, tview.NewTableCell(t.Status).SetTextColor(tcell.ColorAqua).SetExpansion(1))
		tlp.SetCell(row, 1, tview.NewTableCell(t.Name).SetExpansion(3))
		tlp.SetCell(row, 2, tview.NewTableCell(t.Priority).SetExpansion(1))
	}

	if len(tlp.tasks) > 0 {
		tlp.Select(1, 0)
	}
}

func (tlp *TaskListPane) showEmpty(msg string) {
	tlp.Clear()
	tlp.SetCell(0, 0, tview.NewTableCell(msg).SetTextColor(tcell.ColorDarkGray).SetSelectable(false))
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
			// Brief delay is not needed; LoadDetail will set up taskID/listID,
			// but the status picker requires the detail pane to be populated.
			// Instead, open the detail pane and let the user press s there.
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
			tlp.SetCell(row, 0, tview.NewTableCell(newStatus).SetTextColor(tcell.ColorAqua).SetExpansion(1))
			return
		}
	}
}
