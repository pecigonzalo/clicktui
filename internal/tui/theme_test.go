// Package tui — unit tests for pure theme/helper functions.
package tui

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"

	"github.com/pecigonzalo/clicktui/internal/app"
)

// ── tagColor / hexByte ────────────────────────────────────────────────────────

func TestTagColor_HexColor(t *testing.T) {
	c := tcell.NewHexColor(0xff8800)
	got := tagColor(c)
	if got != "[#ff8800]" {
		t.Errorf("tagColor(%v) = %q, want %q", c, got, "[#ff8800]")
	}
}

func TestTagColor_DefaultColor_FallsBack(t *testing.T) {
	c := tcell.ColorDefault // RGB returns (-1,-1,-1)
	got := tagColor(c)
	if got != "[white]" {
		t.Errorf("tagColor(Default) = %q, want %q", got, "[white]")
	}
}

func TestTagColor_NamedColorWithRGB(t *testing.T) {
	// DodgerBlue has well-known RGB values: (30, 144, 255)
	got := tagColor(tcell.ColorDodgerBlue)
	if !strings.HasPrefix(got, "[#") || !strings.HasSuffix(got, "]") {
		t.Errorf("tagColor(DodgerBlue) = %q, want hex tag", got)
	}
}

func TestHexByte(t *testing.T) {
	tests := []struct {
		v    int32
		want string
	}{
		{0x00, "00"},
		{0xff, "ff"},
		{0x1e, "1e"},
		{0xa0, "a0"},
	}
	for _, tt := range tests {
		got := hexByte(tt.v)
		if got != tt.want {
			t.Errorf("hexByte(%#x) = %q, want %q", tt.v, got, tt.want)
		}
	}
}

// ── priorityColor ─────────────────────────────────────────────────────────────

func TestPriorityColor(t *testing.T) {
	tests := []struct {
		priority string
		want     tcell.Color
	}{
		{"urgent", ColorBadgePriorityUrgent},
		{"high", ColorBadgePriorityHigh},
		{"normal", ColorBadgePriorityNormal},
		{"low", ColorBadgePriorityLow},
		{"unknown", ColorTextMuted},
		{"", ColorTextMuted},
	}
	for _, tt := range tests {
		got := priorityColor(tt.priority)
		if got != tt.want {
			t.Errorf("priorityColor(%q) = %v, want %v", tt.priority, got, tt.want)
		}
	}
}

// ── prioritySymbol ────────────────────────────────────────────────────────────

func TestPrioritySymbol(t *testing.T) {
	tests := []struct {
		priority  string
		wantRunes int
	}{
		{"urgent", 2},
		{"high", 2},
		{"normal", 2},
		{"low", 2},
		{"other", 2},
	}
	for _, tt := range tests {
		got := prioritySymbol(tt.priority)
		runeCount := len([]rune(got))
		if runeCount != tt.wantRunes {
			t.Errorf("prioritySymbol(%q) has %d runes, want %d (got %q)", tt.priority, runeCount, tt.wantRunes, got)
		}
	}
	// Spot-check values.
	if got := prioritySymbol("urgent"); !strings.Contains(got, "!") {
		t.Errorf("prioritySymbol(urgent) = %q, expected to contain '!'", got)
	}
	if got := prioritySymbol("low"); strings.Contains(got, "!") {
		t.Errorf("prioritySymbol(low) = %q, should not contain '!'", got)
	}
}

// ── nodeKindSymbol ────────────────────────────────────────────────────────────

func TestNodeKindSymbol(t *testing.T) {
	kinds := []app.NodeKind{
		app.NodeWorkspace,
		app.NodeSpace,
		app.NodeFolder,
		app.NodeList,
	}
	seen := make(map[string]bool)
	for _, k := range kinds {
		sym := nodeKindSymbol(k, false)
		if sym == "" {
			t.Errorf("nodeKindSymbol(%v, false) returned empty string", k)
		}
		if seen[sym] {
			t.Errorf("nodeKindSymbol(%v, false) = %q collides with another kind", k, sym)
		}
		seen[sym] = true
	}

	// Folder expanded vs collapsed should return different symbols.
	closedSym := nodeKindSymbol(app.NodeFolder, false)
	openSym := nodeKindSymbol(app.NodeFolder, true)
	if closedSym == "" || openSym == "" {
		t.Error("folder symbols must not be empty")
	}
	if closedSym == openSym {
		t.Errorf("folder open (%q) and closed (%q) symbols should differ", openSym, closedSym)
	}
}

// ── nodeColor ─────────────────────────────────────────────────────────────────

func TestNodeColor_AllKinds(t *testing.T) {
	kinds := []app.NodeKind{
		app.NodeWorkspace,
		app.NodeSpace,
		app.NodeFolder,
		app.NodeList,
	}
	for _, k := range kinds {
		c := nodeColor(k)
		if c == tcell.ColorDefault {
			t.Errorf("nodeColor(%v) returned ColorDefault — expected a real colour", k)
		}
	}
}
