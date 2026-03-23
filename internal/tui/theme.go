// Package tui — shared design tokens and styling helpers.
//
// All colour and style decisions live here. The rest of the TUI package
// imports only from this file; no raw tcell.Color or tview colour strings
// should appear in pane files.
package tui

import (
	"github.com/gdamore/tcell/v2"

	"github.com/pecigonzalo/clicktui/internal/app"
)

// ── Semantic colour tokens ────────────────────────────────────────────────────

// Foreground / text
var (
	// ColorText is the default body text colour.
	ColorText = tcell.ColorWhite
	// ColorTextMuted is for secondary / dimmed content.
	ColorTextMuted = tcell.NewHexColor(0x888888)
	// ColorTextSubtle is for placeholder or hint text.
	ColorTextSubtle = tcell.ColorDarkGray
)

// Chrome / borders
var (
	// ColorBorderFocused is the border colour for the currently-focused pane.
	ColorBorderFocused = tcell.ColorDodgerBlue
	// ColorBorderInactive is the border colour for unfocused panes.
	ColorBorderInactive = tcell.NewHexColor(0x444444)
	// ColorTitleFocused is the title colour for the focused pane.
	ColorTitleFocused = tcell.ColorWhite
	// ColorTitleInactive is the title colour for unfocused panes.
	ColorTitleInactive = tcell.NewHexColor(0x888888)
)

// Status / state
var (
	// ColorStatusOK is used for ready/success status text.
	ColorStatusOK = tcell.NewHexColor(0x888888)
	// ColorStatusLoading is used for in-progress status text.
	ColorStatusLoading = tcell.ColorYellow
	// ColorStatusError is used for error status text.
	ColorStatusError = tcell.ColorRed
)

// Data badges
var (
	// ColorBadgeStatus is the foreground for task status badges.
	ColorBadgeStatus = tcell.ColorAqua
	// ColorBadgePriorityUrgent is the foreground for "urgent" priority.
	ColorBadgePriorityUrgent = tcell.ColorRed
	// ColorBadgePriorityHigh is the foreground for "high" priority.
	ColorBadgePriorityHigh = tcell.ColorOrange
	// ColorBadgePriorityNormal is the foreground for "normal" priority.
	ColorBadgePriorityNormal = tcell.ColorYellow
	// ColorBadgePriorityLow is the foreground for "low" priority.
	ColorBadgePriorityLow = tcell.NewHexColor(0x888888)
)

// Selection / affordances
var (
	// ColorPaginationHint is the foreground for the "load more" row.
	ColorPaginationHint = tcell.NewHexColor(0x666666)
)

// Hierarchy tree node colours
var (
	ColorNodeWorkspace = tcell.ColorGold
	ColorNodeSpace     = tcell.ColorDodgerBlue
	ColorNodeFolder    = tcell.ColorMediumAquamarine
	ColorNodeList      = tcell.ColorWhite
)

// Table header / label colours
var (
	// ColorTableHeader is the foreground for table header cells.
	ColorTableHeader = tcell.NewHexColor(0xaaaaaa)
	// ColorDetailLabel is the foreground for detail field labels.
	ColorDetailLabel = tcell.NewHexColor(0xaaaaaa)
	// ColorDetailValue is the foreground for detail field values.
	ColorDetailValue = tcell.ColorWhite
)

// Filter
var (
	// ColorFilterAccent is the foreground for filter indicators in pane titles.
	ColorFilterAccent = tcell.ColorMediumOrchid
	// ColorFilterPrompt is the foreground for the filter input prompt.
	ColorFilterPrompt = tcell.ColorMediumOrchid
)

// Footer
var (
	// ColorFooterStatus is the foreground for the left status segment.
	ColorFooterStatus = tcell.NewHexColor(0x888888)
	// ColorFooterKey is the foreground for keybinding labels in the help segment.
	ColorFooterKey = tcell.ColorYellow
	// ColorFooterSep is the foreground for separator characters in the footer.
	ColorFooterSep = tcell.NewHexColor(0x444444)
)

// ── tview colour-tag helpers ──────────────────────────────────────────────────
// These return tview dynamic-colour tag strings so callers never need to
// hand-roll "[colourname]" format strings.

// tagColor returns a tview foreground colour tag for a tcell.Color.
func tagColor(c tcell.Color) string {
	r, g, b := c.RGB()
	if r < 0 || g < 0 || b < 0 {
		// Named colour without RGB info — fall back to a safe name.
		return "[white]"
	}
	return colorToTag(r, g, b)
}

// colorToTag formats r,g,b into a tview hex colour tag.
func colorToTag(r, g, b int32) string {
	return "[#" + hexByte(r) + hexByte(g) + hexByte(b) + "]"
}

func hexByte(v int32) string {
	const digits = "0123456789abcdef"
	return string([]byte{digits[v>>4], digits[v&0xf]})
}

// priorityColor returns the semantic badge colour for a priority string.
func priorityColor(priority string) tcell.Color {
	switch priority {
	case "urgent":
		return ColorBadgePriorityUrgent
	case "high":
		return ColorBadgePriorityHigh
	case "normal":
		return ColorBadgePriorityNormal
	case "low":
		return ColorBadgePriorityLow
	default:
		return ColorTextMuted
	}
}

// prioritySymbol returns a compact single-character indicator for a priority.
// This is used alongside the full name to give a quick visual scan affordance.
func prioritySymbol(priority string) string {
	switch priority {
	case "urgent":
		return "!!"
	case "high":
		return " !"
	case "normal":
		return " -"
	case "low":
		return " ·"
	default:
		return "  "
	}
}

// nodeKindSymbol returns a compact unicode symbol for the node kind.
func nodeKindSymbol(k app.NodeKind) string {
	switch k {
	case app.NodeWorkspace:
		return "◉"
	case app.NodeSpace:
		return "◈"
	case app.NodeFolder:
		return "▸"
	case app.NodeList:
		return "≡"
	default:
		return "·"
	}
}
