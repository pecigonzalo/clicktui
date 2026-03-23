// Package tui — task detail pane and status picker.
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

// CurrentTaskID returns the ID of the task currently displayed in the detail
// pane, or "" when no task has been loaded.
func (td *TaskDetailPane) CurrentTaskID() string {
	return td.taskID
}

func (td *TaskDetailPane) inputHandler(event *tcell.EventKey) *tcell.EventKey {
	if event.Key() != tcell.KeyRune {
		return event
	}
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
	return event
}

// LoadDetail fetches and renders a task's full details.
func (td *TaskDetailPane) LoadDetail(taskID string) {
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
			td.render(detail)
			td.tuiApp.footer.SetStatusReady(fmt.Sprintf("Status → %q", status))

			// Refresh the task list so the status column reflects the change.
			td.tuiApp.taskList.refreshCurrentTask(taskID, detail.Status)
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
				statusBadge(st.Status),
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

	td.SetText(b.String())
	td.ScrollToBeginning()
}
