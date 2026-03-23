// Package tui — task detail pane, status picker, and field selector.
package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/pecigonzalo/clicktui/internal/app"
)

const pageStatusPicker = "status_picker"

// ── Field selector types ─────────────────────────────────────────────────────

// fieldKind describes the action available on a selectable field.
type fieldKind int

const (
	fieldCopy     fieldKind = iota // y copies the value
	fieldOpen                      // y copies, o opens URL
	fieldNavigate                  // Enter navigates to task, y copies ID
)

// selectableField represents a single field in the selector overlay.
type selectableField struct {
	label string    // Display name (e.g., "Task ID", "URL", "Due Date")
	value string    // Raw copyable value
	kind  fieldKind // What kind of action is available
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
	td.tuiApp.footer.SetHelp("↑↓:field", "y:copy", "o:open", "Enter:go", "Esc:back")
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
		SetTitle(" " + icons.StatusDot + " Update Status ").
		SetBorderColor(ColorBorderFocused).
		SetTitleColor(ColorTitleFocused)
	list.ShowSecondaryText(true)
	list.SetMainTextStyle(tcell.StyleDefault.Foreground(ColorDetailValue))
	list.SetSecondaryTextStyle(tcell.StyleDefault.Foreground(ColorTextMuted))

	// dismissModal closes the picker, restores focus chrome, and resets footer
	// help to the default keybinding set.
	dismissModal := func() {
		td.tuiApp.pages.RemovePage(pageStatusPicker)
		td.tuiApp.setFocusPane(paneTaskDetail)
		td.tuiApp.footer.SetHelp(
			"Tab:next pane",
			"Shift+Tab:prev pane",
			"Enter:select",
			"[:toggle tree",
			"s:update status",
			"r:reload",
			"q:quit",
		)
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
			td.detail = detail
			td.buildFields(detail)
			td.render(detail)
			td.tuiApp.footer.SetStatusReady(fmt.Sprintf("Status → %q", status))

			// Refresh the task list so the status column reflects the change.
			td.tuiApp.taskList.refreshCurrentTask(taskID, detail.Status, detail.StatusColor)
		})
	}()
}

// ── Rendering helpers ─────────────────────────────────────────────────────────

// detailLabel returns a right-padded, coloured label for a field row.
func detailLabel(s string) string {
	const width = 10
	if len(s) < width {
		s += strings.Repeat(" ", width-len(s))
	}
	return tagColor(ColorDetailLabel) + s + "[-]"
}

// sectionHeader returns a tview-formatted section divider line.
// When icon is non-empty the format is:  "icon Title ────────────"
// When icon is empty:                    "── Title ──────────────"
func sectionHeader(title, icon string) string {
	var left string
	if icon != "" {
		left = icon + " "
	} else {
		left = "── "
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
		fmt.Fprintf(&b, "%s  %s%s[-]\n",
			detailLabel(""),
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

	// Value display — truncate long values for readability.
	val := truncateDisplay(f.value, 48)

	// Kind hint suffix.
	var hint string
	switch f.kind {
	case fieldOpen:
		hint = tagColor(ColorTextMuted) + "  [o:open]" + "[-]"
	case fieldNavigate:
		hint = tagColor(ColorTextMuted) + "  [↵:go]" + "[-]"
	default:
		hint = ""
	}

	if selected {
		fmt.Fprintf(b, "%s[::r]%s  %s[::- ]%s\n", cursor, label, tview.Escape(val), hint)
	} else {
		fmt.Fprintf(b, "%s%s%s[-]  %s%s[-]%s\n",
			cursor,
			tagColor(ColorDetailLabel), label,
			tagColor(ColorDetailValue), tview.Escape(val),
			hint)
	}
}

// ── Field registry ───────────────────────────────────────────────────────────

// buildFields populates the selectable field list from a TaskDetail.
// Fields with empty values are excluded.
func (td *TaskDetailPane) buildFields(d *app.TaskDetail) {
	if td.fields != nil {
		td.fields = td.fields[:0]
	}

	// Always include the task ID.
	td.fields = append(td.fields, selectableField{"Task ID", d.ID, fieldCopy})
	if d.CustomID != "" {
		td.fields = append(td.fields, selectableField{"Custom ID", d.CustomID, fieldCopy})
	}
	if d.URL != "" {
		td.fields = append(td.fields, selectableField{"URL", d.URL, fieldOpen})
	}
	if d.DueDate != "" {
		td.fields = append(td.fields, selectableField{"Due Date", d.DueDate, fieldCopy})
	}
	if d.StartDate != "" {
		td.fields = append(td.fields, selectableField{"Start Date", d.StartDate, fieldCopy})
	}
	if d.Parent != "" {
		td.fields = append(td.fields, selectableField{"Parent", d.Parent, fieldNavigate})
	}
	if d.Description != "" {
		td.fields = append(td.fields, selectableField{"Description", d.Description, fieldCopy})
	}
	for _, st := range d.Subtasks {
		td.fields = append(td.fields, selectableField{icons.SubtaskPrefix + " " + st.Name, st.ID, fieldNavigate})
	}
}
