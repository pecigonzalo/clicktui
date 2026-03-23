// Package tui — reusable chrome helpers.
//
// chrome.go provides building blocks for consistent pane styling:
//   - applyPaneStyle: focused / inactive border + title treatment
//   - newFooter: two-segment footer (left status, right keybindings)
//   - stateText helpers: loading / error / empty placeholder text
package tui

import (
	"fmt"
	"strings"

	"github.com/rivo/tview"
)

// ── Pane chrome ──────────────────────────────────────────────────────────────

// PaneStyler holds references needed to update a pane's chrome when focus
// changes.
type PaneStyler struct {
	box     *tview.Box
	title   string
	focused bool
}

// newPaneStyler creates a PaneStyler for the given box and title text.
// The title should NOT include the surrounding spaces; those are added here.
func newPaneStyler(box *tview.Box, title string) *PaneStyler {
	return &PaneStyler{box: box, title: title}
}

// SetFocused applies focused-pane chrome (bright border + white title).
func (ps *PaneStyler) SetFocused() {
	ps.focused = true
	ps.box.SetBorderColor(ColorBorderFocused).
		SetTitleColor(ColorTitleFocused).
		SetTitle(" " + ps.title + " ")
}

// SetInactive applies inactive-pane chrome (dim border + muted title).
func (ps *PaneStyler) SetInactive() {
	ps.focused = false
	ps.box.SetBorderColor(ColorBorderInactive).
		SetTitleColor(ColorTitleInactive).
		SetTitle(" " + ps.title + " ")
}

// reapply re-renders the current focus state with the current title.
// Call this after mutating ps.title to refresh the pane border.
func (ps *PaneStyler) reapply() {
	if ps.focused {
		ps.SetFocused()
	} else {
		ps.SetInactive()
	}
}

// ── Footer / status bar ───────────────────────────────────────────────────────

// Footer is a two-segment status bar: left side carries a dynamic status
// message; right side carries a static context-sensitive keybinding hint.
type Footer struct {
	*tview.TextView
	status string
	help   string
}

// newFooter creates a Footer TextView ready to be placed in a layout.
func newFooter() *Footer {
	tv := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(false)
	tv.SetBackgroundColor(ColorBorderInactive)

	f := &Footer{TextView: tv}
	f.refresh()
	return f
}

// SetStatus updates the left (status) segment.
func (f *Footer) SetStatus(format string, args ...any) {
	f.status = fmt.Sprintf(format, args...)
	f.refresh()
}

// SetStatusLoading shows a yellow "loading" message in the status segment.
func (f *Footer) SetStatusLoading(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	f.status = "[yellow]" + tview.Escape(msg) + "[-]"
	f.refresh()
}

// SetStatusError shows a red error message in the status segment.
func (f *Footer) SetStatusError(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	f.status = "[red]✗ " + tview.Escape(msg) + "[-]"
	f.refresh()
}

// SetStatusReady shows a muted ready message in the status segment.
func (f *Footer) SetStatusReady(msg string) {
	f.status = tagColor(ColorFooterStatus) + tview.Escape(msg) + "[-]"
	f.refresh()
}

// SetHelp updates the right (keybinding) segment.  Pass a list of
// "key:action" pairs; they are rendered as "[key] action  [key] action".
func (f *Footer) SetHelp(pairs ...string) {
	var sb strings.Builder
	for i, pair := range pairs {
		if i > 0 {
			sb.WriteString("  ")
		}
		k, action, found := strings.Cut(pair, ":")
		if !found {
			sb.WriteString(pair)
			continue
		}
		sb.WriteString("[yellow]" + k + "[-]")
		sb.WriteString(" " + action)
	}
	f.help = sb.String()
	f.refresh()
}

func (f *Footer) refresh() {
	// Pad status and right-align the help hint using a flexible separator.
	// tview TextViews don't support right-alignment, so we use a wide separator.
	if f.help == "" {
		f.SetText(" " + f.status)
		return
	}
	sep := tagColor(ColorFooterSep) + " │ " + "[-]"
	f.SetText(" " + f.status + sep + f.help + " ")
}

// ── Inline state helpers ──────────────────────────────────────────────────────

// loadingText returns a tview-formatted "loading" placeholder.
func loadingText(msg string) string {
	return "[yellow]⟳ " + tview.Escape(msg) + "[-]"
}

// errorText returns a tview-formatted error placeholder.
func errorText(msg string) string {
	return "[red]✗ " + tview.Escape(msg) + "[-]"
}

// emptyText returns a tview-formatted empty-state placeholder.
func emptyText(msg string) string {
	return tagColor(ColorTextSubtle) + tview.Escape(msg) + "[-]"
}
