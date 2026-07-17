// Package tui — full-screen help overlay.
//
// ShowHelpOverlay presents a K9s-style keybinding cheat sheet: every shortcut
// in the app, grouped by context, in a scrollable modal. Opened with '?' from
// anywhere global shortcuts are active; closed with Esc, '?', or 'q'.
package tui

import (
	"strings"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const pageHelpOverlay = "help_overlay"

// helpSection is a named group of keybindings shown together in the overlay.
type helpSection struct {
	title string
	binds [][2]string // [key, description] pairs
}

// helpSections is the full keybinding reference, grouped by where the binding
// applies. Keep this in sync with the actual input handlers — it is the single
// place a user can discover every shortcut in the app.
var helpSections = []helpSection{
	{
		title: "Global",
		binds: [][2]string{
			{"Tab / Shift+Tab", "Cycle focus between panes"},
			{":", "Jump to any list by name (command palette)"},
			{"/", "Filter the focused pane"},
			{"[", "Toggle the hierarchy pane"},
			{"y", "Copy focused entity's ID"},
			{"B", "Open bookmarks"},
			{"?", "Toggle this help"},
			{"q", "Quit"},
		},
	},
	{
		title: "Hierarchy pane",
		binds: [][2]string{
			{"↑↓ / j k", "Move selection"},
			{"Enter", "Expand node / select list"},
		},
	},
	{
		title: "Task list pane",
		binds: [][2]string{
			{"↑↓ / j k", "Move selection"},
			{"Enter", "Open task details"},
			{"c", "Create a task"},
			{"s", "Update status of selected task"},
			{"S", "Cycle sort field"},
			{"T", "Toggle sort direction"},
			{"b", "Toggle bookmark"},
			{"r", "Reload tasks"},
			{"(automatic)", "Silently refreshes in the background every 30s"},
		},
	},
	{
		title: "Task detail pane",
		binds: [][2]string{
			{"↑↓ / j k", "Scroll"},
			{"Enter", "Enter field selector"},
			{"s", "Update status"},
			{"m", "Move to another list"},
			{"b", "Toggle bookmark"},
			{"r", "Reload task"},
		},
	},
	{
		title: "Field selector (inside detail pane)",
		binds: [][2]string{
			{"↑↓ / j k", "Move between fields"},
			{"y / Enter", "Copy field value (or navigate for subtasks/parent)"},
			{"o", "Open field value as URL"},
			{"e", "Edit field"},
			{"Esc", "Exit field selector"},
		},
	},
	{
		title: "Bookmarks overlay",
		binds: [][2]string{
			{"Enter", "Navigate to bookmark"},
			{"d / Delete", "Remove bookmark"},
			{"Esc", "Close"},
		},
	},
	{
		title: "Filter bar",
		binds: [][2]string{
			{"Enter", "Apply filter"},
			{"Esc", "Clear and close filter"},
		},
	},
	{
		title: "Filter syntax (task list)",
		binds: [][2]string{
			{"status:<text>", "Match status (substring, case-insensitive)"},
			{"priority:<text>", "Match priority"},
			{"assignee:<text>", "Match first assignee"},
			{"due:<text>", "Match due date, e.g. due:2026-05 for a month"},
			{"<free text>", "Fuzzy-match task name; combine with fields, e.g. \"auth status:todo\""},
		},
	},
}

// ShowHelpOverlay opens a centred, scrollable modal listing every keybinding
// in the app. It calls SetModalActive(true) on show and SetModalActive(false)
// on dismiss. Must be called from the UI goroutine.
func ShowHelpOverlay(a *App) {
	view := tview.NewTextView().
		SetDynamicColors(true).
		SetWordWrap(false).
		SetScrollable(true)
	view.SetBorder(true).
		SetTitle(" Help ").
		SetBorderColor(ColorBorderFocused).
		SetTitleColor(ColorTitleFocused)

	view.SetText(renderHelpText())

	dismiss := func() {
		a.SetModalActive(false)
		a.pages.RemovePage(pageHelpOverlay)
		a.restoreDefaultHelp()
	}

	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			dismiss()
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case '?', 'q':
				dismiss()
				return nil
			}
		}
		return event
	})

	a.footer.SetHelp("↑↓:scroll", "Esc/?:close")
	a.SetModalActive(true)

	w, h := modalScreenSize(a)
	width := clamp(w-8, 40, 72)
	height := clamp(h-4, 12, 40)
	modal := centreModal(view, width, height)

	a.pages.AddPage(pageHelpOverlay, modal, true, true)
	a.tviewApp.SetFocus(view)
}

// renderHelpText formats helpSections into a tview-colour-tagged string with
// aligned key/description columns per section.
func renderHelpText() string {
	var sb strings.Builder
	for i, section := range helpSections {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(tagColor(ColorFilterAccent))
		sb.WriteString(strings.ToUpper(section.title))
		sb.WriteString("[-]\n")

		keyWidth := 0
		for _, b := range section.binds {
			if w := utf8.RuneCountInString(b[0]); w > keyWidth {
				keyWidth = w
			}
		}

		for _, b := range section.binds {
			sb.WriteString("  ")
			sb.WriteString(tagColor(ColorFooterKey))
			sb.WriteString(tview.Escape(padRight(b[0], keyWidth)))
			sb.WriteString("[-]  ")
			sb.WriteString(tview.Escape(b[1]))
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// padRight pads s with spaces to width runes.
func padRight(s string, width int) string {
	n := utf8.RuneCountInString(s)
	if n >= width {
		return s
	}
	return s + strings.Repeat(" ", width-n)
}
