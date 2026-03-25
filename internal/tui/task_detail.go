// Package tui — task detail pane, status picker, and field selector.
package tui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/pecigonzalo/clicktui/internal/app"
)

const pageStatusPicker = "status_picker"
const pageListPicker = "list_picker"

// ── Field selector types ─────────────────────────────────────────────────────

// fieldKind describes the action available on a selectable field.
type fieldKind int

const (
	fieldCopy     fieldKind = iota // y copies the value
	fieldOpen                      // y copies, o opens URL
	fieldNavigate                  // Enter navigates to task, y copies ID
	fieldEdit                      // e opens an editor modal
)

// editType identifies the type of editor modal to use for a fieldEdit field.
type editType int

const (
	editTypeText     editType = iota // single-line text input
	editTypeTextArea                 // multi-line text area
	editTypeDate                     // date input (YYYY-MM-DD)
	editTypeAssignee                 // multi-select assignee picker
)

// selectableField represents a single field in the selector overlay.
type selectableField struct {
	label    string    // Display name (e.g., "Task ID", "URL", "Due Date")
	value    string    // Raw copyable value (empty string for placeholder fields)
	kind     fieldKind // What kind of action is available
	edit     editType  // Editor type (only meaningful when kind == fieldEdit)
	hasValue bool      // Whether the field has a real value (vs. placeholder)
}

type moveListOption struct {
	WorkspaceID string
	ID          string
	Name        string
	Label       string
}

// TaskDetailPane shows detailed information about a selected task.
type TaskDetailPane struct {
	*tview.TextView
	tuiApp   *App
	taskID   string
	listID   string
	taskName string

	// Field selector state.
	detail       *app.TaskDetail   // last rendered detail for field access
	fields       []selectableField // current selectable fields
	selectedIdx  int               // currently selected field index
	selectorMode bool              // whether selector mode is active
}

// NewTaskDetailPane creates an empty task detail pane.
func NewTaskDetailPane(a *App) *TaskDetailPane {
	tv := tview.NewTextView().
		SetDynamicColors(true).
		SetWordWrap(true).
		SetScrollable(true)
	tv.SetBorder(true)
	tv.SetText(emptyText("Select a task to view details"))

	tdp := &TaskDetailPane{
		TextView:    tv,
		tuiApp:      a,
		selectedIdx: -1,
	}
	tv.SetInputCapture(tdp.inputHandler)
	return tdp
}

// CurrentTaskID returns the ID of the task currently displayed in the detail
// pane, or "" when no task has been loaded.
func (td *TaskDetailPane) CurrentTaskID() string {
	return td.taskID
}

func (td *TaskDetailPane) inputHandler(event *tcell.EventKey) *tcell.EventKey {
	if td.selectorMode {
		return td.selectorInputHandler(event)
	}
	switch event.Key() {
	case tcell.KeyEnter:
		if td.taskID != "" && len(td.fields) > 0 {
			td.enterSelectorMode()
			return nil
		}
	case tcell.KeyRune:
		switch event.Rune() {
		case 's':
			if td.taskID != "" {
				td.openStatusPicker()
				return nil
			}
		case 'm':
			if td.taskID != "" {
				td.openMoveListPicker()
				return nil
			}
		case 'r':
			if td.taskID != "" {
				td.tuiApp.tasks.InvalidateTaskDetail(td.taskID)
				td.LoadDetail(td.taskID)
				return nil
			}
		}
	}
	return event
}

// selectorInputHandler handles keys while the field selector is active.
func (td *TaskDetailPane) selectorInputHandler(event *tcell.EventKey) *tcell.EventKey {
	n := len(td.fields)
	if n == 0 {
		td.exitSelectorMode()
		return nil
	}

	switch event.Key() {
	case tcell.KeyEscape:
		td.exitSelectorMode()
		return nil
	case tcell.KeyUp:
		td.selectedIdx = (td.selectedIdx - 1 + n) % n
		td.renderWithSelector()
		return nil
	case tcell.KeyDown:
		td.selectedIdx = (td.selectedIdx + 1) % n
		td.renderWithSelector()
		return nil
	case tcell.KeyEnter:
		f := td.fields[td.selectedIdx]
		if f.kind == fieldNavigate {
			td.exitSelectorMode()
			td.LoadDetail(f.value)
			return nil
		}
		// For non-navigate fields, Enter copies the value (same as y).
		td.copyFieldAndExit(f)
		return nil
	case tcell.KeyRune:
		switch event.Rune() {
		case 'k':
			td.selectedIdx = (td.selectedIdx - 1 + n) % n
			td.renderWithSelector()
			return nil
		case 'j':
			td.selectedIdx = (td.selectedIdx + 1) % n
			td.renderWithSelector()
			return nil
		case 'y':
			td.copyFieldAndExit(td.fields[td.selectedIdx])
			return nil
		case 'e':
			f := td.fields[td.selectedIdx]
			if f.kind == fieldEdit {
				td.exitSelectorMode()
				td.dispatchEdit(f)
				return nil
			}
			// Ignore 'e' on non-editable fields.
			return nil
		case 'o':
			f := td.fields[td.selectedIdx]
			if f.kind == fieldOpen {
				td.exitSelectorMode()
				if err := openURL(f.value); err != nil {
					td.tuiApp.footer.SetStatusError("open url: %v", err)
				} else {
					td.tuiApp.footer.SetStatusReady("Opened: " + f.value)
				}
				return nil
			}
			// Ignore 'o' on non-openable fields.
			return nil
		}
	}
	return event
}

// enterSelectorMode activates the field selector overlay.
func (td *TaskDetailPane) enterSelectorMode() {
	td.selectorMode = true
	td.selectedIdx = 0
	td.renderWithSelector()
	td.tuiApp.footer.SetHelp("↑↓:field", "y:copy", "o:open", "e:edit", "Enter:go", "Esc:back")
}

// exitSelectorMode deactivates the field selector and re-renders clean.
func (td *TaskDetailPane) exitSelectorMode() {
	td.selectorMode = false
	td.selectedIdx = -1
	if td.detail != nil {
		td.render(td.detail)
	}
	td.tuiApp.restoreDefaultHelp()
}

// copyFieldAndExit copies the selected field value to the clipboard and exits
// selector mode.
func (td *TaskDetailPane) copyFieldAndExit(f selectableField) {
	td.exitSelectorMode()
	if err := writeClipboard(f.value); err != nil {
		td.tuiApp.footer.SetStatusError("copy failed: %v", err)
		return
	}
	td.tuiApp.footer.SetStatusReady(fmt.Sprintf("Copied %s: %s", f.label, truncateDisplay(f.value, 40)))
}

// truncateDisplay shortens s for display if longer than max runes.
func truncateDisplay(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

// LoadDetail fetches and renders a task's full details.
func (td *TaskDetailPane) LoadDetail(taskID string) {
	// Exit selector mode when loading a new task.
	td.selectorMode = false
	td.selectedIdx = -1

	td.tuiApp.setStatusLoading("Loading task %s…", taskID)

	go func() {
		detail, err := td.tuiApp.tasks.LoadTaskDetail(td.tuiApp.ctx, taskID)
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
			td.detail = detail
			td.buildFields(detail)
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

	go func() {
		statuses, err := td.tuiApp.tasks.LoadListStatuses(td.tuiApp.ctx, listID)
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
		SetTitle(" " + icons.StatusDot + " Update Status ").
		SetBorderColor(ColorBorderFocused).
		SetTitleColor(ColorTitleFocused)
	list.ShowSecondaryText(true)
	list.SetMainTextStyle(tcell.StyleDefault.Foreground(ColorDetailValue))
	list.SetSecondaryTextStyle(tcell.StyleDefault.Foreground(ColorTextMuted))

	// dismissModal closes the picker, restores focus chrome, and resets footer
	// help to the default keybinding set.
	dismissModal := func() {
		td.tuiApp.SetModalActive(false)
		td.tuiApp.pages.RemovePage(pageStatusPicker)
		td.tuiApp.setFocusPane(paneTaskDetail)
		td.tuiApp.restoreDefaultHelp()
	}

	for _, s := range statuses {
		// Capture loop variable for closure.
		chosen := s.Name

		// Build a coloured dot prefix using any color hint from the API.
		// Fall back to the generic badge colour when the API color is absent.
		dotColor := statusDotColor(s.Color, s.Type)
		dot := tagColor(dotColor) + icons.StatusDot + "[-]"
		typeLabel := statusTypeLabel(s.Type)
		main := dot + " " + tview.Escape(s.Name)

		list.AddItem(main, typeLabel, 0, func() {
			dismissModal()
			td.applyStatusUpdate(taskID, chosen)
		})
	}

	// Explicit cancel row — gives a visible affordance beyond just Esc.
	list.AddItem(
		tagColor(ColorTextMuted)+"✕ Cancel[-]",
		"press Esc or Enter to dismiss",
		0,
		func() {
			dismissModal()
			td.tuiApp.footer.SetStatusReady("Status update cancelled")
		},
	)

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			dismissModal()
			td.tuiApp.footer.SetStatusReady("Status update cancelled")
			return nil
		}
		return event
	})

	// Hint in footer while modal is open.
	td.tuiApp.footer.SetHelp("Enter:select", "Esc:cancel")

	// Mark modal as active so globalInputHandler suppresses global shortcuts.
	td.tuiApp.SetModalActive(true)

	// Height: status rows + cancel row + borders + secondary-text rows.
	// Each list item takes 2 rows when secondary text is shown; cap at 30.
	modalHeight := min(len(statuses)*2+1*2+4, 30)
	modal := centreModal(list, 48, modalHeight)
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

	go func() {
		detail, err := td.tuiApp.tasks.UpdateTaskStatus(td.tuiApp.ctx, taskID, status)
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
			td.detail = detail
			td.buildFields(detail)
			td.render(detail)
			td.tuiApp.footer.SetStatusReady(fmt.Sprintf("Status → %q", status))

			// Refresh the task list so the status column reflects the change.
			td.tuiApp.taskList.refreshCurrentTask(taskID, detail.Status, detail.StatusColor)
		})
	}()
}

// openMoveListPicker collects list targets from the loaded hierarchy and
// displays a modal to move the current task to another list.
func (td *TaskDetailPane) openMoveListPicker() {
	options := td.collectMoveListOptions()
	if len(options) == 0 {
		td.tuiApp.setError("no loaded destination lists; expand spaces in the tree first")
		return
	}
	td.showMoveListModal(td.taskID, options)
}

// collectMoveListOptions returns list destinations discovered in the currently
// loaded hierarchy tree, excluding the task's current list.
func (td *TaskDetailPane) collectMoveListOptions() []moveListOption {
	type localOption struct {
		workspaceID string
		id          string
		name        string
		label       string
	}
	var collected []localOption
	seen := make(map[string]struct{})

	var walk func(nodes []*app.HierarchyNode, workspaceID string, path []string)
	walk = func(nodes []*app.HierarchyNode, workspaceID string, path []string) {
		for _, n := range nodes {
			switch n.Kind {
			case app.NodeWorkspace:
				walk(n.Children, n.ID, append(path, n.Name))
			case app.NodeSpace, app.NodeFolder:
				walk(n.Children, workspaceID, append(path, n.Name))
			case app.NodeList:
				if workspaceID == "" || n.ID == "" || n.ID == td.listID {
					continue
				}
				if _, ok := seen[n.ID]; ok {
					continue
				}
				seen[n.ID] = struct{}{}
				labelParts := append(path, n.Name)
				collected = append(collected, localOption{
					workspaceID: workspaceID,
					id:          n.ID,
					name:        n.Name,
					label:       strings.Join(labelParts, " / "),
				})
			}
		}
	}

	walk(td.tuiApp.tree.cachedNodes, "", nil)
	sort.SliceStable(collected, func(i, j int) bool {
		return strings.ToLower(collected[i].label) < strings.ToLower(collected[j].label)
	})

	options := make([]moveListOption, len(collected))
	for i, o := range collected {
		options[i] = moveListOption{WorkspaceID: o.workspaceID, ID: o.id, Name: o.name, Label: o.label}
	}
	return options
}

// showMoveListModal renders the move-to-list picker modal over the main layout.
// Must be called from the UI goroutine.
func (td *TaskDetailPane) showMoveListModal(taskID string, options []moveListOption) {
	list := tview.NewList()
	list.SetBorder(true).
		SetTitle(" Move Task To List ").
		SetBorderColor(ColorBorderFocused).
		SetTitleColor(ColorTitleFocused)
	list.ShowSecondaryText(true)
	list.SetMainTextStyle(tcell.StyleDefault.Foreground(ColorDetailValue))
	list.SetSecondaryTextStyle(tcell.StyleDefault.Foreground(ColorTextMuted))

	dismissModal := func() {
		td.tuiApp.pages.RemovePage(pageListPicker)
		td.tuiApp.setFocusPane(paneTaskDetail)
		td.tuiApp.restoreDefaultHelp()
	}

	for _, opt := range options {
		chosenWorkspaceID := opt.WorkspaceID
		chosenID := opt.ID
		chosenName := opt.Name
		main := tview.Escape(opt.Label)
		secondary := "#" + opt.ID
		list.AddItem(main, secondary, 0, func() {
			dismissModal()
			td.applyMoveToList(chosenWorkspaceID, taskID, chosenID, chosenName)
		})
	}

	list.AddItem(
		tagColor(ColorTextMuted)+"Cancel[-]",
		"press Esc or Enter to dismiss",
		0,
		func() {
			dismissModal()
			td.tuiApp.footer.SetStatusReady("Move cancelled")
		},
	)

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			dismissModal()
			td.tuiApp.footer.SetStatusReady("Move cancelled")
			return nil
		}
		return event
	})

	td.tuiApp.footer.SetHelp("Enter:select", "Esc:cancel")

	modalHeight := min(len(options)*2+1*2+4, 30)
	modal := centreModal(list, 72, modalHeight)
	td.tuiApp.pages.AddPage(pageListPicker, modal, true, true)
	td.tuiApp.tviewApp.SetFocus(list)
}

// applyMoveToList calls the service and refreshes the detail + list panes.
func (td *TaskDetailPane) applyMoveToList(workspaceID, taskID, listID, listName string) {
	td.tuiApp.setStatusLoading("Moving task to %q…", listName)

	go func() {
		detail, err := td.tuiApp.tasks.MoveTaskToList(td.tuiApp.ctx, workspaceID, taskID, listID)
		td.tuiApp.tviewApp.QueueUpdateDraw(func() {
			if err != nil {
				td.tuiApp.logger.Error("move task to list", "workspace", workspaceID, "task", taskID, "list", listID, "error", err)
				td.tuiApp.setError("move task: %v", err)
				return
			}

			movedListID := detail.ListID
			if movedListID == "" {
				movedListID = listID
			}
			movedListName := detail.List
			if movedListName == "" {
				movedListName = listName
			}

			td.taskID = detail.ID
			td.listID = movedListID
			td.taskName = detail.Name
			detail.ListID = movedListID
			detail.List = movedListName
			td.detail = detail
			td.buildFields(detail)
			td.render(detail)

			// Reload destination list so the moved task appears in its new home.
			td.tuiApp.taskList.LoadTasks(movedListID, movedListName)
			td.tuiApp.footer.SetStatusReady(fmt.Sprintf("Moved to %q", movedListName))
		})
	}()
}

// ── Field editing ─────────────────────────────────────────────────────────────

// dispatchEdit opens the appropriate editor modal for f.
// Must be called from the UI goroutine.
func (td *TaskDetailPane) dispatchEdit(f selectableField) {
	switch f.edit {
	case editTypeDate:
		td.editDate(f)
	case editTypeTextArea:
		td.editDescription(f)
	case editTypeAssignee:
		td.editAssignees()
	}
}

// editDate opens a date modal for a Due Date or Start Date field.
func (td *TaskDetailPane) editDate(f selectableField) {
	taskID := td.taskID
	listID := td.listID
	isDueDate := f.label == "Due Date"

	ShowDateModal(td.tuiApp, DateModalConfig{
		Title:      " Edit " + f.label + " ",
		Initial:    f.value, // already in YYYY-MM-DD or empty
		AllowClear: true,
		OnSubmit: func(date string) {
			td.applyDateUpdate(taskID, listID, isDueDate, date)
		},
		OnCancel: func() {
			td.tuiApp.footer.SetStatusReady("Edit cancelled")
		},
	})
}

// applyDateUpdate calls UpdateTask with the new date value and refreshes panes.
// date is YYYY-MM-DD or "" to clear.
func (td *TaskDetailPane) applyDateUpdate(taskID, listID string, isDueDate bool, date string) {
	// Convert YYYY-MM-DD to epoch milliseconds string, or "" to clear.
	var epochMS string
	if date != "" {
		t, err := time.Parse(time.DateOnly, date)
		if err != nil {
			td.tuiApp.setError("invalid date: %v", err)
			return
		}
		epochMS = strconv.FormatInt(t.UnixMilli(), 10)
	}

	var input app.UpdateTaskInput
	if isDueDate {
		input.DueDate = &epochMS
		td.tuiApp.setStatusLoading("Updating due date…")
	} else {
		// Start date reuse DueDate field label but needs its own API field.
		// For now map to DueDate; when a StartDate API field exists swap here.
		// ClickUp v2 does not expose start_date in the update task endpoint
		// the same way — set DueDate only for due date, skip for start date.
		// TODO: add StartDate to UpdateTaskInput when the API supports it.
		td.tuiApp.footer.SetStatusReady("Start date editing not supported by the API yet")
		return
	}

	go func() {
		err := td.tuiApp.tasks.UpdateTask(td.tuiApp.ctx, taskID, input)
		td.tuiApp.tviewApp.QueueUpdateDraw(func() {
			if err != nil {
				td.tuiApp.logger.Error("update task date", "task", taskID, "error", err)
				td.tuiApp.setError("update date: %v", err)
				return
			}
			td.tuiApp.tasks.InvalidateTaskDetail(taskID)
			td.LoadDetail(taskID)
			td.tuiApp.tasks.InvalidateTaskList(listID)
			td.tuiApp.taskList.reapplyFilter()
			if date == "" {
				td.tuiApp.footer.SetStatusReady("Due date cleared")
			} else {
				td.tuiApp.footer.SetStatusReady("Due date → " + date)
			}
		})
	}()
}

// editDescription opens a textarea modal for the description field.
func (td *TaskDetailPane) editDescription(f selectableField) {
	taskID := td.taskID
	listID := td.listID

	ShowTextAreaModal(td.tuiApp, TextAreaModalConfig{
		Title:   " Edit Description ",
		Initial: f.value,
		OnSubmit: func(text string) {
			td.applyDescriptionUpdate(taskID, listID, text)
		},
		OnCancel: func() {
			td.tuiApp.footer.SetStatusReady("Edit cancelled")
		},
	})
}

// applyDescriptionUpdate calls UpdateTask with the new description and refreshes.
func (td *TaskDetailPane) applyDescriptionUpdate(taskID, listID, description string) {
	td.tuiApp.setStatusLoading("Updating description…")

	go func() {
		err := td.tuiApp.tasks.UpdateTask(td.tuiApp.ctx, taskID, app.UpdateTaskInput{
			Description: &description,
		})
		td.tuiApp.tviewApp.QueueUpdateDraw(func() {
			if err != nil {
				td.tuiApp.logger.Error("update task description", "task", taskID, "error", err)
				td.tuiApp.setError("update description: %v", err)
				return
			}
			td.tuiApp.tasks.InvalidateTaskDetail(taskID)
			td.LoadDetail(taskID)
			_ = listID // description doesn't affect task list display
			td.tuiApp.footer.SetStatusReady("Description updated")
		})
	}()
}

// editAssignees loads list members and opens a multi-select modal.
func (td *TaskDetailPane) editAssignees() {
	taskID := td.taskID
	listID := td.listID
	currentIDs := td.detail.AssigneeIDs

	td.tuiApp.setStatusLoading("Loading members…")

	go func() {
		members, err := td.tuiApp.tasks.LoadMembers(td.tuiApp.ctx, listID)
		td.tuiApp.tviewApp.QueueUpdateDraw(func() {
			if err != nil {
				td.tuiApp.logger.Error("load members for assignee edit", "list", listID, "error", err)
				td.tuiApp.setError("load members: %v", err)
				return
			}
			td.tuiApp.footer.SetStatusReady("")
			td.showAssigneeModal(taskID, listID, currentIDs, members)
		})
	}()
}

// showAssigneeModal renders the assignee multi-select modal.
// Must be called from the UI goroutine.
func (td *TaskDetailPane) showAssigneeModal(taskID, listID string, currentIDs []int, members []app.MemberSummary) {
	// Build option list — pre-select current assignees.
	currentSet := make(map[int]struct{}, len(currentIDs))
	for _, id := range currentIDs {
		currentSet[id] = struct{}{}
	}

	opts := make([]SelectOption, len(members))
	for i, m := range members {
		label := m.Username
		if m.Email != "" {
			label += " (" + m.Email + ")"
		}
		_, preSelected := currentSet[m.ID]
		opts[i] = SelectOption{
			Label:    label,
			Value:    strconv.Itoa(m.ID),
			Selected: preSelected,
		}
	}

	ShowSelectModal(td.tuiApp, SelectModalConfig{
		Title:   " Edit Assignees ",
		Options: opts,
		Multi:   true,
		OnSubmit: func(selected []string) {
			td.applyAssigneeUpdate(taskID, listID, currentIDs, selected)
		},
		OnCancel: func() {
			td.tuiApp.footer.SetStatusReady("Edit cancelled")
		},
	})
}

// applyAssigneeUpdate computes add/remove diffs and calls UpdateTask.
func (td *TaskDetailPane) applyAssigneeUpdate(taskID, listID string, currentIDs []int, selectedValues []string) {
	// Parse selected IDs.
	selectedSet := make(map[int]struct{}, len(selectedValues))
	for _, v := range selectedValues {
		id, err := strconv.Atoi(v)
		if err == nil {
			selectedSet[id] = struct{}{}
		}
	}

	// Compute add = selected - current.
	currentSet := make(map[int]struct{}, len(currentIDs))
	for _, id := range currentIDs {
		currentSet[id] = struct{}{}
	}

	var add, rem []int
	for id := range selectedSet {
		if _, ok := currentSet[id]; !ok {
			add = append(add, id)
		}
	}
	for id := range currentSet {
		if _, ok := selectedSet[id]; !ok {
			rem = append(rem, id)
		}
	}

	if len(add) == 0 && len(rem) == 0 {
		td.tuiApp.footer.SetStatusReady("No changes to assignees")
		return
	}

	td.tuiApp.setStatusLoading("Updating assignees…")

	go func() {
		err := td.tuiApp.tasks.UpdateTask(td.tuiApp.ctx, taskID, app.UpdateTaskInput{
			AssigneesAdd: add,
			AssigneesRem: rem,
		})
		td.tuiApp.tviewApp.QueueUpdateDraw(func() {
			if err != nil {
				td.tuiApp.logger.Error("update task assignees", "task", taskID, "error", err)
				td.tuiApp.setError("update assignees: %v", err)
				return
			}
			td.tuiApp.tasks.InvalidateTaskDetail(taskID)
			td.LoadDetail(taskID)
			td.tuiApp.tasks.InvalidateTaskList(listID)
			td.tuiApp.taskList.reapplyFilter()
			td.tuiApp.footer.SetStatusReady("Assignees updated")
		})
	}()
}

// ── Rendering helpers ─────────────────────────────────────────────────────────

// detailLabel returns a right-padded, coloured label for a field row.
// Uses rune count for width calculation so icon-prefixed labels align correctly.
func detailLabel(s string) string {
	const width = 12
	n := utf8.RuneCountInString(s)
	if n < width {
		s += strings.Repeat(" ", width-n)
	}
	return tagColor(ColorDetailLabel) + s + "[-]"
}

// sectionHeader returns a tview-formatted section divider line.
// When icon is non-empty the format is:  "╭─ icon Title ────────────"
// When icon is empty:                    "╭─ Title ──────────────"
// The leading corner character uses round box-drawing when Nerd Fonts are
// enabled and a plain dash for the unicode preset.
func sectionHeader(title, icon string) string {
	var left string
	if icon != "" {
		left = icons.SectionCorner + " " + icon + " "
	} else {
		left = icons.SectionCorner + " "
	}
	right := " " + strings.Repeat("─", 28)
	return tagColor(ColorTextMuted) + left + title + right + "[-]"
}

// statusBadge returns a styled inline badge for a status string.
func statusBadge(status string) string {
	return tagColor(ColorBadgeStatus) + icons.StatusDot + " " + tview.Escape(status) + "[-]"
}

// statusBadgeColored returns a styled inline badge using the API-provided hex
// color. Falls back to ColorBadgeStatus when hexColor is empty or invalid.
func statusBadgeColored(status, hexColor string) string {
	c := statusDotColor(hexColor, "")
	return tagColor(c) + icons.StatusDot + " " + tview.Escape(status) + "[-]"
}

// priorityBadge returns a styled inline badge for a priority string.
func priorityBadge(priority string) string {
	sym := prioritySymbol(priority)
	c := priorityColor(priority)
	return tagColor(c) + sym + " " + tview.Escape(priority) + "[-]"
}

// statusDotColor returns a tcell color for the status dot in the picker.
// It tries to parse an API-provided hex color string; falls back to
// ColorBadgeStatus when absent or unparseable.
func statusDotColor(apiColor, statusType string) tcell.Color {
	// ClickUp returns colors as "#rrggbb" strings.
	if len(apiColor) == 7 && apiColor[0] == '#' {
		var r, g, b uint8
		if n, _ := fmt.Sscanf(apiColor, "#%02x%02x%02x", &r, &g, &b); n == 3 {
			return tcell.NewRGBColor(int32(r), int32(g), int32(b))
		}
	}
	// Fall back to type-based colouring so closed/done statuses look muted.
	switch statusType {
	case "closed", "done":
		return ColorTextMuted
	default:
		return ColorBadgeStatus
	}
}

// statusTypeLabel returns a short human-friendly label for a status type string.
func statusTypeLabel(t string) string {
	switch t {
	case "open":
		return "open"
	case "custom":
		return "in-progress"
	case "closed", "done":
		return "closed"
	default:
		return t
	}
}

func (td *TaskDetailPane) render(d *app.TaskDetail) {
	td.renderBody(d, nil)
}

// renderWithSelector re-renders the detail body with the selector overlay
// appended at the bottom.
func (td *TaskDetailPane) renderWithSelector() {
	if td.detail == nil {
		return
	}
	sel := &selectorState{fields: td.fields, idx: td.selectedIdx}
	td.renderBody(td.detail, sel)
	td.ScrollToEnd()
}

// selectorState holds rendering parameters for the field selector overlay.
type selectorState struct {
	fields []selectableField
	idx    int
}

// renderBody renders the full detail pane text. When sel is non-nil, the
// field selector overlay is appended at the bottom.
func (td *TaskDetailPane) renderBody(d *app.TaskDetail, sel *selectorState) {
	var b strings.Builder

	// ── Title block ──────────────────────────────────────────────────────────
	// Task name in bold white, then ID + custom ID on a muted line.
	fmt.Fprintf(&b, "[white::b]%s[-:-:-]\n", tview.Escape(d.Name))

	idLine := tagColor(ColorTextSubtle) + tview.Escape(d.ID) + "[-]"
	if d.CustomID != "" {
		idLine += "  " + tagColor(ColorTextMuted) + tview.Escape(d.CustomID) + "[-]"
	}
	b.WriteString(idLine + "\n\n")

	// ── Status & Priority badges ─────────────────────────────────────────────
	b.WriteString(detailLabel("Status") + "  " + statusBadgeColored(d.Status, d.StatusColor) + "\n")
	b.WriteString(detailLabel("Priority") + "  " + priorityBadge(d.Priority) + "\n")

	// ── People & Tags ────────────────────────────────────────────────────────
	if len(d.Assignees) > 0 || len(d.Tags) > 0 {
		b.WriteString("\n")
		if len(d.Assignees) > 0 {
			label := "Assignees"
			if icons.Assignee != "" {
				label = icons.Assignee + " " + label
			}
			fmt.Fprintf(&b, "%s  %s%s[-]\n",
				detailLabel(label),
				tagColor(ColorDetailValue),
				tview.Escape(strings.Join(d.Assignees, ", ")))
		}
		if len(d.Tags) > 0 {
			label := "Tags"
			if icons.Tag != "" {
				label = icons.Tag + " " + label
			}
			fmt.Fprintf(&b, "%s  %s%s[-]\n",
				detailLabel(label),
				tagColor(ColorDetailValue),
				tview.Escape(strings.Join(d.Tags, ", ")))
		}
	}

	// ── Dates ────────────────────────────────────────────────────────────────
	hasDates := d.DueDate != "" || d.StartDate != "" || d.DateCreated != "" || d.DateUpdated != ""
	if hasDates {
		b.WriteString("\n" + sectionHeader("Dates", icons.Calendar) + "\n")

		// Primary dates: Due and Start on their own rows.
		if d.DueDate != "" {
			dueDateColor := tagColor(ColorDetailValue)
			if t, err := time.Parse(time.DateOnly, d.DueDate); err == nil {
				today := time.Now().Truncate(24 * time.Hour)
				switch {
				case t.Before(today):
					dueDateColor = tagColor(ColorStatusError) // overdue → red
				case t.Equal(today):
					dueDateColor = tagColor(ColorStatusLoading) // today → yellow
				}
			}
			fmt.Fprintf(&b, "%s  %s%s[-]\n", detailLabel("Due"), dueDateColor, d.DueDate)
		}
		if d.StartDate != "" {
			fmt.Fprintf(&b, "%s  %s%s[-]\n", detailLabel("Start"), tagColor(ColorDetailValue), d.StartDate)
		}

		// Secondary dates: Created/Updated on a single muted line.
		if d.DateCreated != "" || d.DateUpdated != "" {
			b.WriteString("\n")
			muted := tagColor(ColorTextMuted)
			var parts []string
			if d.DateCreated != "" {
				parts = append(parts, "Created "+d.DateCreated)
			}
			if d.DateUpdated != "" {
				parts = append(parts, "Updated "+d.DateUpdated)
			}
			fmt.Fprintf(&b, "%s%s[-]\n", muted, strings.Join(parts, "  ·  "))
		}
	}

	// ── Location ─────────────────────────────────────────────────────────────
	b.WriteString("\n" + sectionHeader("Location", icons.Location) + "\n")

	// Compact breadcrumb: skip empty segments (e.g. hidden folder).
	sep := " " + icons.Breadcrumb + " "
	var segments []string
	for _, seg := range []string{d.Space, d.Folder, d.List} {
		if seg != "" {
			segments = append(segments, tview.Escape(seg))
		}
	}
	if len(segments) > 0 {
		fmt.Fprintf(&b, "  %s%s[-]\n",
			tagColor(ColorDetailValue),
			strings.Join(segments, sep))
	}
	if d.Parent != "" {
		fmt.Fprintf(&b, "%s  %s%s %s[-]\n",
			detailLabel("Parent"),
			tagColor(ColorDetailValue),
			icons.ParentPrefix,
			tview.Escape(d.Parent))
	}

	// ── Subtasks ─────────────────────────────────────────────────────────────
	if len(d.Subtasks) > 0 {
		b.WriteString("\n" + sectionHeader(fmt.Sprintf("Subtasks (%d)", len(d.Subtasks)), icons.Subtask) + "\n")
		for _, st := range d.Subtasks {
			fmt.Fprintf(&b, "%s%s[-] %s%s[-]  %s  %s%s[-]\n",
				tagColor(ColorDetailValue),
				icons.SubtaskPrefix,
				tagColor(ColorTextSubtle),
				tview.Escape(st.ID),
				statusBadgeColored(st.Status, st.StatusColor),
				tagColor(ColorDetailValue),
				tview.Escape(st.Name))
		}
	}

	// ── URL ──────────────────────────────────────────────────────────────────
	if d.URL != "" {
		b.WriteString("\n")
		label := "URL"
		if icons.Link != "" {
			label = icons.Link + " " + label
		}
		fmt.Fprintf(&b, "%s  %s%s[-]\n", detailLabel(label), tagColor(ColorTextSubtle), d.URL)
	}

	// ── Description ──────────────────────────────────────────────────────────
	if d.Description != "" {
		b.WriteString("\n" + sectionHeader("Description", icons.Description) + "\n")
		gutter := tagColor(ColorTextMuted) + "│" + "[-] "
		for line := range strings.SplitSeq(tview.Escape(d.Description), "\n") {
			fmt.Fprintf(&b, "%s%s%s[-]\n", gutter, tagColor(ColorDetailValue), line)
		}
	}

	// ── Field selector overlay ──────────────────────────────────────────────
	if sel != nil && len(sel.fields) > 0 {
		b.WriteString("\n" + tagColor(ColorTextMuted) + "─── Select Field " + strings.Repeat("─", 24) + "[-]\n")
		for i, f := range sel.fields {
			td.renderSelectorRow(&b, f, i == sel.idx)
		}
	}

	td.SetText(b.String())
	td.ScrollToBeginning()
}

// renderSelectorRow writes a single field row into b.
// When selected is true, the row is rendered with a highlight and cursor.
func (td *TaskDetailPane) renderSelectorRow(b *strings.Builder, f selectableField, selected bool) {
	const labelWidth = 14 // Pad labels for alignment.

	// Cursor indicator.
	cursor := "  "
	if selected {
		cursor = tagColor(ColorSelectorHighlight) + "> " + "[-]"
	}

	// Padded label.
	label := tview.Escape(f.label)
	if len(f.label) < labelWidth {
		label += strings.Repeat(" ", labelWidth-len(f.label))
	}

	// Value display — use placeholder text for empty editable fields.
	var val string
	var valColor tcell.Color
	if !f.hasValue && f.kind == fieldEdit {
		switch f.edit {
		case editTypeDate:
			val = "(no date)"
		case editTypeTextArea:
			val = "(no description)"
		case editTypeAssignee:
			val = "(no assignees)"
		default:
			val = "(none)"
		}
		valColor = ColorTextMuted
	} else {
		val = truncateDisplay(f.value, 48)
		valColor = ColorDetailValue
	}

	// Kind hint suffix.
	var hint string
	switch f.kind {
	case fieldOpen:
		hint = tagColor(ColorTextMuted) + "  [o:open]" + "[-]"
	case fieldNavigate:
		hint = tagColor(ColorTextMuted) + "  [↵:go]" + "[-]"
	case fieldEdit:
		hint = tagColor(ColorTextMuted) + "  [e:edit]" + "[-]"
	default:
		hint = ""
	}

	if selected {
		fmt.Fprintf(b, "%s[::r]%s  %s[::- ]%s\n", cursor, label, tview.Escape(val), hint)
	} else {
		fmt.Fprintf(b, "%s%s%s[-]  %s%s[-]%s\n",
			cursor,
			tagColor(ColorDetailLabel), label,
			tagColor(valColor), tview.Escape(val),
			hint)
	}
}

// ── Field registry ───────────────────────────────────────────────────────────

// buildFields populates the selectable field list from a TaskDetail.
// Read-only fields with empty values are excluded.
// Editable fields always appear, using a placeholder when the value is absent.
func (td *TaskDetailPane) buildFields(d *app.TaskDetail) {
	if td.fields != nil {
		td.fields = td.fields[:0]
	}

	// Always include the task ID (read-only copy).
	td.fields = append(td.fields, selectableField{
		label:    "Task ID",
		value:    d.ID,
		kind:     fieldCopy,
		hasValue: true,
	})
	if d.CustomID != "" {
		td.fields = append(td.fields, selectableField{
			label:    "Custom ID",
			value:    d.CustomID,
			kind:     fieldCopy,
			hasValue: true,
		})
	}
	if d.URL != "" {
		td.fields = append(td.fields, selectableField{
			label:    "URL",
			value:    d.URL,
			kind:     fieldOpen,
			hasValue: true,
		})
	}

	// Editable: due date — always present so the user can set it when empty.
	td.fields = append(td.fields, selectableField{
		label:    "Due Date",
		value:    d.DueDate,
		kind:     fieldEdit,
		edit:     editTypeDate,
		hasValue: d.DueDate != "",
	})

	// Editable: start date — always present so the user can set it when empty.
	td.fields = append(td.fields, selectableField{
		label:    "Start Date",
		value:    d.StartDate,
		kind:     fieldEdit,
		edit:     editTypeDate,
		hasValue: d.StartDate != "",
	})

	if d.Parent != "" {
		td.fields = append(td.fields, selectableField{
			label:    "Parent",
			value:    d.Parent,
			kind:     fieldNavigate,
			hasValue: true,
		})
	}

	// Editable: description — always present so the user can set it when empty.
	td.fields = append(td.fields, selectableField{
		label:    "Description",
		value:    d.Description,
		kind:     fieldEdit,
		edit:     editTypeTextArea,
		hasValue: d.Description != "",
	})

	// Editable: assignees — always present so the user can set them when empty.
	assigneeDisplay := strings.Join(d.Assignees, ", ")
	td.fields = append(td.fields, selectableField{
		label:    "Assignees",
		value:    assigneeDisplay,
		kind:     fieldEdit,
		edit:     editTypeAssignee,
		hasValue: len(d.Assignees) > 0,
	})

	for _, st := range d.Subtasks {
		td.fields = append(td.fields, selectableField{
			label:    icons.SubtaskPrefix + " " + st.Name,
			value:    st.ID,
			kind:     fieldNavigate,
			hasValue: true,
		})
	}
}
