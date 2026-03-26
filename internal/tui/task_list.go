// Package tui — task list pane.
package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/pecigonzalo/clicktui/internal/app"
	"github.com/pecigonzalo/clicktui/internal/config"
)

// Sort direction arrows for the pane title indicator.
const (
	sortArrowAsc  = "↑"
	sortArrowDesc = "↓"
)

// TaskListPane displays tasks for a selected list in a table.
type TaskListPane struct {
	*tview.Table
	tuiApp      *App
	tasks       []app.TaskSummary // currently displayed tasks (may be filtered+sorted)
	allTasks    []app.TaskSummary // unfiltered full task set for the current list
	activeQuery *app.TaskQuery    // non-nil when a filter is active
	currentID   string
	listName    string
	page        int
	isLoading   bool
	styler      *PaneStyler // set by app.go after construction

	// Sort state.
	sortField   string         // current sort field ("" = no sort)
	sortAsc     bool           // sort direction
	statusOrder map[string]int // status name → position; populated from list statuses
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
	tlp.restoreSortPreference()
	tlp.loadStatusOrder(listID)
	tlp.fetchPage()
}

func (tlp *TaskListPane) fetchPage() {
	if tlp.isLoading {
		return
	}
	tlp.isLoading = true
	tlp.tuiApp.setStatusLoading("Loading tasks for %s (page %d)…", tlp.listName, tlp.page)

	go func() {
		tasks, err := tlp.tuiApp.tasks.LoadTasks(tlp.tuiApp.ctx, tlp.currentID, tlp.page)
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
	// Format: "ListName  #ListID  N tasks  ↑field" (sort indicator when active)
	if tlp.styler != nil {
		title := fmt.Sprintf("%s  %s#%s  %d tasks[-]",
			tview.Escape(tlp.listName),
			tagColor(ColorTextMuted),
			tlp.currentID,
			len(tlp.tasks),
		)
		if ind := tlp.sortIndicator(); ind != "" {
			title += "  " + tagColor(ColorFilterAccent) + ind + "[-]"
		}
		tlp.styler.title = title
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
		// Prepend bookmark indicator when the task is bookmarked.
		if tlp.isTaskBookmarked(t.ID) {
			nameText = tagColor(ColorStatusLoading) + icons.Bookmark + "[-] " + nameText
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

// restoreSelectionByID attempts to select the row for the task with the given
// ID after a re-sort or re-filter. If the ID is not found (or empty), falls
// back to selecting the first data row.
func (tlp *TaskListPane) restoreSelectionByID(taskID string) {
	if taskID == "" {
		return
	}
	for i, t := range tlp.tasks {
		if t.ID == taskID {
			tlp.Select(i+1, 0) // +1 for header row
			return
		}
	}
	// ID not in current visible set — stay on first row.
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
	case event.Key() == tcell.KeyRune && event.Rune() == 'c':
		// Initiate task creation flow for the current list.
		if tlp.currentID == "" {
			tlp.tuiApp.setError("select a list first")
			return nil
		}
		tlp.startCreateFlow()
		return nil
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
	case event.Key() == tcell.KeyRune && event.Rune() == 'S':
		// Cycle sort field: none → status → priority → due_date → assignee → name → none.
		prevID := tlp.SelectedTaskID()
		tlp.cycleSortField()
		tlp.restoreSelectionByID(prevID)
		return nil
	case event.Key() == tcell.KeyRune && event.Rune() == 'T':
		// Toggle sort direction (ascending ↔ descending).
		prevID := tlp.SelectedTaskID()
		tlp.toggleSortDirection()
		tlp.restoreSelectionByID(prevID)
		return nil
	case event.Key() == tcell.KeyRune && event.Rune() == 'r':
		if tlp.currentID != "" {
			tlp.tuiApp.tasks.InvalidateTaskList(tlp.currentID)
			tlp.allTasks = nil
			tlp.tasks = nil
			tlp.LoadTasks(tlp.currentID, tlp.listName)
			return nil
		}
	case event.Key() == tcell.KeyRune && event.Rune() == 'b':
		// Toggle bookmark for the currently selected task.
		row, _ := tlp.GetSelection()
		idx := row - 1
		if idx >= 0 && idx < len(tlp.tasks) {
			tlp.toggleBookmark(tlp.tasks[idx])
		}
		return nil
	}
	return event
}

// ── Task creation flow ───────────────────────────────────────────────────────

// startCreateFlow initiates the sequential task creation wizard.
// Step 1: prompt for task name (required).
// Step 2: prompt for status selection.
// Step 3: prompt for priority (optional; "skip" option available).
// Must be called from the UI goroutine.
func (tlp *TaskListPane) startCreateFlow() {
	listID := tlp.currentID

	ShowInputModal(tlp.tuiApp, InputModalConfig{
		Title:       "New Task Name",
		Placeholder: "Enter task name…",
		Validate: func(s string) error {
			if strings.TrimSpace(s) == "" {
				return errTaskNameRequired
			}
			return nil
		},
		OnSubmit: func(name string) {
			tlp.createFlowPickStatus(listID, strings.TrimSpace(name))
		},
		OnCancel: func() {
			tlp.tuiApp.footer.SetStatusReady("Create cancelled")
		},
	})
}

// createFlowPickStatus loads list statuses and shows the status picker.
// Called after the task name has been entered.
func (tlp *TaskListPane) createFlowPickStatus(listID, name string) {
	tlp.tuiApp.setStatusLoading("Loading statuses…")

	go func() {
		statuses, err := tlp.tuiApp.tasks.LoadListStatuses(tlp.tuiApp.ctx, listID)
		tlp.tuiApp.tviewApp.QueueUpdateDraw(func() {
			if err != nil {
				tlp.tuiApp.logger.Error("create task: load statuses", "list", listID, "error", err)
				tlp.tuiApp.setError("load statuses: %v", err)
				return
			}
			if len(statuses) == 0 {
				tlp.tuiApp.setError("no statuses available for this list")
				return
			}
			tlp.tuiApp.footer.SetStatusReady("")
			tlp.createFlowShowStatusModal(listID, name, statuses)
		})
	}()
}

// createFlowShowStatusModal presents the status selection modal.
// Must be called from the UI goroutine.
func (tlp *TaskListPane) createFlowShowStatusModal(listID, name string, statuses []app.StatusOption) {
	opts := make([]SelectOption, len(statuses))
	for i, s := range statuses {
		opts[i] = SelectOption{
			Label: s.Name,
			Value: s.Name,
		}
	}

	ShowSelectModal(tlp.tuiApp, SelectModalConfig{
		Title:   "Select Status",
		Options: opts,
		OnSubmit: func(selected []string) {
			if len(selected) == 0 {
				tlp.tuiApp.footer.SetStatusReady("Create cancelled")
				return
			}
			tlp.createFlowPickPriority(listID, name, selected[0])
		},
		OnCancel: func() {
			tlp.tuiApp.footer.SetStatusReady("Create cancelled")
		},
	})
}

// priorityOptions lists the ClickUp priority choices for the creation flow.
// "none" maps to priority int 0 (omitted by the API).
var priorityOptions = []SelectOption{
	{Label: "Skip (no priority)", Value: "0"},
	{Label: "Urgent", Value: "1"},
	{Label: "High", Value: "2"},
	{Label: "Normal", Value: "3"},
	{Label: "Low", Value: "4"},
}

// createFlowPickPriority shows an optional priority picker.
// Selecting "Skip" creates the task immediately with no priority.
func (tlp *TaskListPane) createFlowPickPriority(listID, name, status string) {
	ShowSelectModal(tlp.tuiApp, SelectModalConfig{
		Title:   "Select Priority (optional)",
		Options: priorityOptions,
		OnSubmit: func(selected []string) {
			priority := 0
			if len(selected) > 0 {
				switch selected[0] {
				case "1":
					priority = 1
				case "2":
					priority = 2
				case "3":
					priority = 3
				case "4":
					priority = 4
				}
			}
			tlp.executeCreate(listID, app.CreateTaskInput{
				Name:     name,
				Status:   status,
				Priority: priority,
			})
		},
		OnCancel: func() {
			// Cancel from priority picker still creates with name+status only.
			tlp.executeCreate(listID, app.CreateTaskInput{
				Name:   name,
				Status: status,
			})
		},
	})
}

// executeCreate calls the service to create the task, then reloads the list.
// Must be called from the UI goroutine.
func (tlp *TaskListPane) executeCreate(listID string, input app.CreateTaskInput) {
	tlp.tuiApp.setStatusLoading("Creating task %q…", input.Name)

	go func() {
		newID, err := tlp.tuiApp.tasks.CreateTask(tlp.tuiApp.ctx, listID, input)
		tlp.tuiApp.tviewApp.QueueUpdateDraw(func() {
			if err != nil {
				tlp.tuiApp.logger.Error("create task", "list", listID, "error", err)
				tlp.tuiApp.setError("create task: %v", err)
				return
			}
			taskName := input.Name

			// Invalidate list cache (CreateTask already does this, but be explicit
			// here so the reload always fetches fresh data).
			tlp.tuiApp.tasks.InvalidateTaskList(listID)
			tlp.allTasks = nil
			tlp.tasks = nil
			tlp.page = 0

			// Reload and, once done, try to select the new task.
			tlp.loadTasksAndSelect(listID, tlp.listName, newID, taskName)
		})
	}()
}

// loadTasksAndSelect reloads the task list and tries to select the task with
// targetID once loading completes. If the active filter hides the new task,
// a footer message is shown. Must be called from the UI goroutine.
func (tlp *TaskListPane) loadTasksAndSelect(listID, listName, targetID, taskName string) {
	tlp.currentID = listID
	tlp.listName = listName
	tlp.page = 0
	tlp.restoreSortPreference()
	tlp.loadStatusOrder(listID)
	// Use a wrapped fetch that selects the new task when loading completes.
	tlp.fetchPageThenSelect(targetID, taskName)
}

// fetchPageThenSelect fetches all pages (like fetchPage) and, when loading
// finishes, selects the task identified by targetID.
func (tlp *TaskListPane) fetchPageThenSelect(targetID, taskName string) {
	if tlp.isLoading {
		return
	}
	tlp.isLoading = true
	tlp.tuiApp.setStatusLoading("Loading tasks for %s (page %d)…", tlp.listName, tlp.page)

	go func() {
		tasks, err := tlp.tuiApp.tasks.LoadTasks(tlp.tuiApp.ctx, tlp.currentID, tlp.page)
		tlp.tuiApp.tviewApp.QueueUpdateDraw(func() {
			tlp.isLoading = false
			if err != nil {
				tlp.tuiApp.logger.Error("load tasks after create", "list", tlp.currentID, "error", err)
				tlp.tuiApp.setError("load tasks: %v", err)
				tlp.showEmpty(fmt.Sprintf("Error loading tasks: %v", err))
				return
			}
			if tlp.page == 0 {
				tlp.allTasks = tasks
			} else {
				tlp.allTasks = append(tlp.allTasks, tasks...)
			}

			if len(tasks) == 100 {
				// More pages — keep paginating, carry targetID through.
				tlp.page++
				tlp.isLoading = false
				tlp.fetchPageThenSelect(targetID, taskName)
				return
			}

			// All pages loaded — reapply filter+sort and select the new task.
			tlp.reapplyFilter()

			// Try to select the newly created task.
			found := tlp.selectTaskByID(targetID)
			if found {
				tlp.tuiApp.footer.SetStatusReady(fmt.Sprintf("Task created: %s", taskName))
			} else if tlp.activeQuery != nil {
				// Task exists but active filter is hiding it.
				tlp.tuiApp.footer.SetStatusReady(fmt.Sprintf("Task created (hidden by filter): %s", taskName))
			} else {
				tlp.tuiApp.footer.SetStatusReady(fmt.Sprintf("Task created: %s", taskName))
			}
		})
	}()
}

// selectTaskByID attempts to highlight the row for the given task ID.
// Returns true when the ID was found in the current visible task set.
func (tlp *TaskListPane) selectTaskByID(taskID string) bool {
	if taskID == "" {
		return false
	}
	for i, t := range tlp.tasks {
		if t.ID == taskID {
			tlp.Select(i+1, 0) // +1 for the header row
			return true
		}
	}
	return false
}

// errTaskNameRequired is returned by the name-field validator when the user
// tries to submit an empty task name.
var errTaskNameRequired = fmt.Errorf("task name is required")

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
	tlp.applySortToTasks()
	tlp.render()
}

// ClearFilter restores the full unfiltered task list. Must be called from the
// UI goroutine.
func (tlp *TaskListPane) ClearFilter() {
	tlp.activeQuery = nil
	tlp.tasks = tlp.allTasks
	tlp.applySortToTasks()
	tlp.render()
}

// reapplyFilter applies the active filter query to allTasks, then sorts, and
// re-renders. When no filter is active, it shows all tasks.
// Flow: allTasks → filter → sort → render.
// Must be called from the UI goroutine.
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
	tlp.applySortToTasks()
	tlp.render()
}

// applySortToTasks sorts tlp.tasks in place using the current sort preferences.
// No-op when sortField is empty or tasks is nil/empty.
func (tlp *TaskListPane) applySortToTasks() {
	if tlp.sortField == "" || len(tlp.tasks) == 0 {
		return
	}
	tlp.tasks = app.SortTasks(tlp.tasks, tlp.sortField, tlp.sortAsc, tlp.statusOrder)
}

// sortIndicator returns the sort indicator string for the pane title.
// Returns "" when no sort is active.
func (tlp *TaskListPane) sortIndicator() string {
	if tlp.sortField == "" {
		return ""
	}
	arrow := sortArrowAsc
	if !tlp.sortAsc {
		arrow = sortArrowDesc
	}
	return arrow + tlp.sortField
}

// cycleSortField advances to the next sort field, persists the preference,
// and re-renders the task list. Must be called from the UI goroutine.
func (tlp *TaskListPane) cycleSortField() {
	tlp.sortField = app.NextSortField(tlp.sortField)
	// Default to ascending when a new field is selected.
	if tlp.sortField != "" {
		tlp.sortAsc = true
	}
	tlp.persistSort()
	tlp.reapplyFilter()
}

// toggleSortDirection flips the sort direction and re-renders. If no sort
// field is active, this is a no-op. Must be called from the UI goroutine.
func (tlp *TaskListPane) toggleSortDirection() {
	if tlp.sortField == "" {
		return
	}
	tlp.sortAsc = !tlp.sortAsc
	tlp.persistSort()
	tlp.reapplyFilter()
}

// persistSort saves the current sort preference to the UI state service.
func (tlp *TaskListPane) persistSort() {
	if tlp.tuiApp == nil || tlp.tuiApp.uiState == nil {
		return
	}
	if err := tlp.tuiApp.uiState.SetSortPreference(tlp.tuiApp.profile, tlp.sortField, tlp.sortAsc); err != nil {
		tlp.tuiApp.logger.Error("persist sort preference", "error", err)
	}
}

// restoreSortPreference loads the saved sort preference from the UI state
// service into the pane's sort fields. Must be called from the UI goroutine.
func (tlp *TaskListPane) restoreSortPreference() {
	if tlp.tuiApp == nil || tlp.tuiApp.uiState == nil {
		return
	}
	tlp.sortField, tlp.sortAsc = tlp.tuiApp.uiState.GetSortPreference(tlp.tuiApp.profile)
}

// loadStatusOrder fetches list statuses and builds the statusOrder map for
// sort-by-status. Runs in a background goroutine to avoid blocking the UI.
func (tlp *TaskListPane) loadStatusOrder(listID string) {
	if tlp.tuiApp == nil {
		return
	}
	go func() {
		statuses, err := tlp.tuiApp.tasks.LoadListStatuses(tlp.tuiApp.ctx, listID)
		if err != nil {
			tlp.tuiApp.logger.Error("load list statuses for sort", "list", listID, "error", err)
			return
		}
		tlp.tuiApp.tviewApp.QueueUpdateDraw(func() {
			order := make(map[string]int, len(statuses))
			for i, s := range statuses {
				order[strings.ToLower(s.Name)] = i
			}
			tlp.statusOrder = order
			// Re-sort if currently sorting by status, now that we have the order.
			if tlp.sortField == app.SortFieldStatus {
				tlp.reapplyFilter()
			}
		})
	}()
}

// ── Bookmark helpers ─────────────────────────────────────────────────────────

// isTaskBookmarked reports whether the task with the given ID is bookmarked in
// the current profile. Returns false when uiState is not available.
func (tlp *TaskListPane) isTaskBookmarked(taskID string) bool {
	if tlp.tuiApp == nil || tlp.tuiApp.uiState == nil {
		return false
	}
	return tlp.tuiApp.uiState.IsBookmarked(tlp.tuiApp.profile, taskID)
}

// toggleBookmark adds or removes a bookmark for t and re-renders the list.
// Must be called from the UI goroutine.
func (tlp *TaskListPane) toggleBookmark(t app.TaskSummary) {
	if tlp.tuiApp == nil || tlp.tuiApp.uiState == nil {
		return
	}
	profile := tlp.tuiApp.profile
	if tlp.tuiApp.uiState.IsBookmarked(profile, t.ID) {
		if err := tlp.tuiApp.uiState.RemoveBookmark(profile, t.ID); err != nil {
			tlp.tuiApp.logger.Error("remove bookmark", "task", t.ID, "error", err)
			tlp.tuiApp.setError("remove bookmark: %v", err)
			return
		}
		tlp.tuiApp.footer.SetStatusReady("Bookmark removed: " + t.Name)
	} else {
		b := config.Bookmark{
			TaskID:   t.ID,
			TaskName: t.Name,
			ListID:   tlp.currentID,
			ListName: tlp.listName,
			AddedAt:  time.Now(),
		}
		if err := tlp.tuiApp.uiState.AddBookmark(profile, b); err != nil {
			tlp.tuiApp.logger.Error("add bookmark", "task", t.ID, "error", err)
			tlp.tuiApp.setError("add bookmark: %v", err)
			return
		}
		tlp.tuiApp.footer.SetStatusReady("Bookmarked: " + t.Name)
	}
	// Re-render so the bookmark indicator updates.
	tlp.render()
}
