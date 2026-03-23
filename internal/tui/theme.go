// Package tui — shared design tokens and styling helpers.
//
// All colour and style decisions live here. The rest of the TUI package
// imports only from this file; no raw tcell.Color or tview colour strings
// should appear in pane files.
package tui

import (
	"strings"

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

// Selector
var (
	// ColorSelectorHighlight is the background colour for the selected row in
	// the field selector overlay.
	ColorSelectorHighlight = tcell.ColorDodgerBlue
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

// ── Icon registry ─────────────────────────────────────────────────────────────

// Icons holds all symbolic glyphs used across the TUI.
type Icons struct {
	// Tree node kinds
	Workspace    string
	Space        string
	Folder       string
	FolderOpen   string // expanded folder (▾ or nf-fa-folder_open)
	FolderClosed string // collapsed folder (▸ or nf-fa-folder)
	List         string

	// Status & priority
	StatusDot      string
	PriorityUrgent string
	PriorityHigh   string
	PriorityNormal string
	PriorityLow    string

	// Detail pane sections
	Calendar      string
	Location      string
	Description   string
	Subtask       string
	Link          string
	Assignee      string
	Tag           string
	SectionCorner string // "╭─" for nerd fonts, "──" for unicode

	// Misc
	SubtaskPrefix string // ↳ or arrow
	ParentPrefix  string // ▸ or arrow
	Breadcrumb    string // › separator
}

var unicodeIcons = Icons{
	Workspace:      "◉",
	Space:          "◈",
	Folder:         "▸",
	FolderOpen:     "▾",
	FolderClosed:   "▸",
	List:           "≡",
	StatusDot:      "●",
	PriorityUrgent: "!!",
	PriorityHigh:   " !",
	PriorityNormal: " -",
	PriorityLow:    " ·",
	Calendar:       "",
	Location:       "",
	Description:    "",
	Subtask:        "",
	Link:           "",
	Assignee:       "",
	Tag:            "",
	SectionCorner:  "──",
	SubtaskPrefix:  "↳",
	ParentPrefix:   "▸",
	Breadcrumb:     "›",
}

var nerdFontIcons = Icons{
	Workspace:      "\U000f0c21", // 󰰡
	Space:          "\uf0ac",     //  nf-fa-globe
	Folder:         "\uf07c",     //  nf-fa-folder_open (legacy default)
	FolderOpen:     "\uf07c",     //  nf-fa-folder_open
	FolderClosed:   "\uf07b",     //  nf-fa-folder
	List:           "\uf03a",     //  nf-fa-list
	StatusDot:      "\uf111",     //  nf-fa-circle
	PriorityUrgent: "\uf06a",     //  nf-fa-exclamation_circle
	PriorityHigh:   "\uf062",     //  nf-fa-arrow_up
	PriorityNormal: "\uf068",     //  nf-fa-minus
	PriorityLow:    "\uf063",     //  nf-fa-arrow_down
	Calendar:       "\uf073",     //  nf-fa-calendar
	Location:       "\uf041",     //  nf-fa-map_marker
	Description:    "\uf15c",     //  nf-fa-file_text
	Subtask:        "\uf0ae",     //  nf-fa-tasks
	Link:           "\uf0c1",     //  nf-fa-link
	Assignee:       "\uf007",     //  nf-fa-user
	Tag:            "\uf02c",     //  nf-fa-tags
	SectionCorner:  "╭─",
	SubtaskPrefix:  "\uf178", //  nf-fa-long_arrow_right
	ParentPrefix:   "\uf062", //  nf-fa-arrow_up
	Breadcrumb:     "\ue0b1", //  powerline right arrow
}

// icons is the active icon set. All rendering code reads from this variable.
var icons = unicodeIcons

// InitIcons selects the icon preset. Call once at startup before any rendering.
func InitIcons(nerdFont bool) {
	if nerdFont {
		icons = nerdFontIcons
	} else {
		icons = unicodeIcons
	}
}

// prioritySymbol returns a compact single-character indicator for a priority.
// This is used alongside the full name to give a quick visual scan affordance.
func prioritySymbol(priority string) string {
	switch strings.ToLower(priority) {
	case "urgent":
		return icons.PriorityUrgent
	case "high":
		return icons.PriorityHigh
	case "normal":
		return icons.PriorityNormal
	case "low":
		return icons.PriorityLow
	default:
		return "  "
	}
}

// nodeKindSymbol returns a compact symbol for the node kind.
// For folder nodes, the symbol changes based on the expanded state.
func nodeKindSymbol(k app.NodeKind, expanded bool) string {
	switch k {
	case app.NodeWorkspace:
		return icons.Workspace
	case app.NodeSpace:
		return icons.Space
	case app.NodeFolder:
		if expanded {
			return icons.FolderOpen
		}
		return icons.FolderClosed
	case app.NodeList:
		return icons.List
	default:
		return "?"
	}
}
