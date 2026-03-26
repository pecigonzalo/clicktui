// Package tui — bookmarks overlay.
//
// ShowBookmarksOverlay presents a centred modal listing all locally-bookmarked
// tasks for the active profile. The user can:
//   - Enter   — navigate to the bookmarked task (loads its list if needed)
//   - d/Delete — remove the selected bookmark
//   - Esc      — close the overlay
package tui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const pageBookmarksOverlay = "bookmarks_overlay"

// ShowBookmarksOverlay opens a modal listing all bookmarks for the active
// profile. It calls SetModalActive(true) on show and SetModalActive(false) on
// dismiss. Must be called from the UI goroutine.
func ShowBookmarksOverlay(a *App) {
	bookmarks := a.uiState.GetBookmarks(a.profile)
	if len(bookmarks) == 0 {
		a.footer.SetStatusReady("No bookmarks yet — press b on a task to add one")
		return
	}

	dismiss := func() {
		a.SetModalActive(false)
		a.pages.RemovePage(pageBookmarksOverlay)
		a.restoreDefaultHelp()
	}

	table := tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0)
	table.SetBorder(true).
		SetTitle(fmt.Sprintf(" %s Bookmarks (%d) ", icons.Bookmark, len(bookmarks))).
		SetBorderColor(ColorBorderFocused).
		SetTitleColor(ColorTitleFocused)
	table.SetSelectedStyle(tcell.StyleDefault.
		Background(ColorBorderFocused).
		Foreground(tcell.ColorWhite).
		Attributes(tcell.AttrBold))

	// renderTable rebuilds the table rows from the current bookmark slice.
	// Declared as a closure over bookmarks so removal can refresh in place.
	renderTable := func() {
		table.Clear()

		// Header row.
		headers := []struct {
			text      string
			expansion int
		}{
			{"TASK NAME", 5},
			{"LIST", 3},
			{"ADDED", 2},
		}
		for i, h := range headers {
			table.SetCell(0, i, tview.NewTableCell(h.text).
				SetTextColor(ColorTableHeader).
				SetSelectable(false).
				SetExpansion(h.expansion).
				SetAttributes(tcell.AttrBold))
		}

		for i, b := range bookmarks {
			row := i + 1

			// Task name with bookmark icon prefix.
			nameText := tagColor(ColorStatusLoading) + icons.Bookmark + "[-] " + tview.Escape(b.TaskName)
			table.SetCell(row, 0, tview.NewTableCell(nameText).
				SetTextColor(ColorText).
				SetExpansion(5))

			// List name, muted to de-emphasise.
			table.SetCell(row, 1, tview.NewTableCell(tview.Escape(b.ListName)).
				SetTextColor(ColorTextMuted).
				SetExpansion(3))

			// Date added — show only the date portion (not time).
			dateText := b.AddedAt.Format("2006-01-02")
			table.SetCell(row, 2, tview.NewTableCell(dateText).
				SetTextColor(ColorTextMuted).
				SetExpansion(2).
				SetMaxWidth(12))
		}

		if len(bookmarks) > 0 {
			table.Select(1, 0)
		}
	}
	renderTable()

	// removeBookmark removes the currently selected bookmark by index, refreshes
	// the table, and dismisses when the list becomes empty.
	removeBookmark := func() {
		row, _ := table.GetSelection()
		idx := row - 1 // row 0 is the header
		if idx < 0 || idx >= len(bookmarks) {
			return
		}
		b := bookmarks[idx]
		if err := a.uiState.RemoveBookmark(a.profile, b.TaskID); err != nil {
			a.logger.Error("remove bookmark from overlay", "task", b.TaskID, "error", err)
			a.footer.SetStatusError("remove bookmark: %v", err)
			return
		}

		// Remove from the in-memory slice so the overlay reflects the change.
		bookmarks = append(bookmarks[:idx], bookmarks[idx+1:]...)

		if len(bookmarks) == 0 {
			// Nothing left — close the overlay.
			dismiss()
			a.footer.SetStatusReady("All bookmarks removed")
			// Re-render the task list so bookmark icons are cleared.
			a.taskList.render()
			return
		}

		// Update the title to reflect the new count.
		table.SetTitle(fmt.Sprintf(" %s Bookmarks (%d) ", icons.Bookmark, len(bookmarks)))
		renderTable()
		a.footer.SetStatusReady("Bookmark removed: " + b.TaskName)
		// Re-render task list so the bookmark indicator updates.
		a.taskList.render()
	}

	// navigateToBookmark loads the list for the bookmark's task then opens the
	// task's detail view.
	navigateToBookmark := func() {
		row, _ := table.GetSelection()
		idx := row - 1
		if idx < 0 || idx >= len(bookmarks) {
			return
		}
		b := bookmarks[idx]

		dismiss()

		// Load the list in the task-list pane (cross-list navigation).
		// If the bookmark's list is already loaded we still call LoadTasks so
		// the list cache is used and we don't incur an unnecessary reload.
		a.taskList.LoadTasks(b.ListID, b.ListName)
		a.setFocusPane(paneTaskList)

		// Queue opening the task detail once the task list redraws.
		a.tviewApp.QueueUpdateDraw(func() {
			a.taskDetail.LoadDetail(b.TaskID)
			a.setFocusPane(paneTaskDetail)
		})

		a.footer.SetStatusReady("Navigating to: " + b.TaskName)
	}

	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			dismiss()
			return nil
		case tcell.KeyEnter:
			navigateToBookmark()
			return nil
		case tcell.KeyDelete:
			removeBookmark()
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'd':
				removeBookmark()
				return nil
			}
		}
		return event
	})

	a.footer.SetHelp("Enter:open", "d:remove", "Del:remove", "Esc:close")
	a.SetModalActive(true)

	// Size: cap height at 24; allow at least 8. Width covers typical task names.
	height := min(len(bookmarks)+1+4, 24) // +1 header, +4 borders/padding
	height = max(height, 8)
	modal := centreModal(table, 68, height)

	a.pages.AddPage(pageBookmarksOverlay, modal, true, true)
	a.tviewApp.SetFocus(table)
}

// ── Stale-bookmark description ────────────────────────────────────────────────
// A "stale" bookmark is one whose task no longer exists in ClickUp. The overlay
// has no way to proactively detect this without an API round-trip per bookmark,
// which is too expensive. Instead, staleness is handled lazily:
//
// 1. Navigating to a stale bookmark calls LoadTasks + LoadDetail as usual.
//    If LoadDetail returns an error, the detail pane shows the error text and
//    the footer shows an error message.  The user can then press d to remove
//    the bookmark from the overlay.
//
// 2. Removing a stale bookmark works identically to removing a live one: press
//    d or Delete while the row is selected.
//
// This approach avoids extra API calls while giving the user a clear path to
// clean up stale bookmarks.
