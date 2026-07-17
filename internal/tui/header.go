// Package tui — persistent header bar.
//
// Header is a one-line bar above the three panes that always shows which
// ClickUp profile is active. Profiles map to different tokens/accounts (see
// `clicktui --profile <name>`), so without a permanent on-screen indicator
// there is no way to tell — short of quitting and checking flags — whether
// you're looking at your personal workspace or a client's. K9s makes the
// same call for cluster/context/namespace; this is the ClickUp analogue.
package tui

import (
	"github.com/rivo/tview"
)

// Header is a one-line status bar rendered above the pane layout.
type Header struct {
	*tview.TextView
	profile string
}

// newHeader creates a Header TextView ready to be placed in a layout.
func newHeader() *Header {
	tv := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(false)
	tv.SetBackgroundColor(ColorBorderInactive)

	h := &Header{TextView: tv}
	h.refresh()
	return h
}

// SetProfile updates the displayed profile name.
func (h *Header) SetProfile(profile string) {
	h.profile = profile
	h.refresh()
}

func (h *Header) refresh() {
	title := tagColor(ColorTitleFocused) + "clicktui" + "[-]"
	profile := tagColor(ColorFooterStatus) + "profile: " + tview.Escape(h.profile) + "[-]"
	sep := tagColor(ColorFooterSep) + " │ " + "[-]"
	h.SetText(" " + title + sep + profile + " ")
}
