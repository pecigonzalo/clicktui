// Package tui — unit tests for pure theme/helper functions.
package tui

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pecigonzalo/clicktui/internal/app"
)

// ── tagColor / hexByte ────────────────────────────────────────────────────────

func TestTagColor_HexColor(t *testing.T) {
	c := tcell.NewHexColor(0xff8800)
	got := tagColor(c)
	assert.Equal(t, "[#ff8800]", got)
}

func TestTagColor_DefaultColor_FallsBack(t *testing.T) {
	c := tcell.ColorDefault // RGB returns (-1,-1,-1)
	got := tagColor(c)
	assert.Equal(t, "[white]", got)
}

func TestTagColor_NamedColorWithRGB(t *testing.T) {
	// DodgerBlue has well-known RGB values: (30, 144, 255)
	got := tagColor(tcell.ColorDodgerBlue)
	assert.True(t, strings.HasPrefix(got, "[#") && strings.HasSuffix(got, "]"))
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
		assert.Equal(t, tt.want, got)
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
		assert.Equal(t, tt.want, got)
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
		assert.Equal(t, tt.wantRunes, runeCount)
	}
	// Spot-check values.
	assert.True(t, strings.Contains(prioritySymbol("urgent"), "!"))
	assert.False(t, strings.Contains(prioritySymbol("low"), "!"))
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
		require.NotEmpty(t, sym)
		assert.False(t, seen[sym])
		seen[sym] = true
	}

	// Folder expanded vs collapsed should return different symbols.
	closedSym := nodeKindSymbol(app.NodeFolder, false)
	openSym := nodeKindSymbol(app.NodeFolder, true)
	require.NotEmpty(t, closedSym)
	require.NotEmpty(t, openSym)
	assert.NotEqual(t, closedSym, openSym)
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
		assert.NotEqual(t, tcell.ColorDefault, c)
	}
}
