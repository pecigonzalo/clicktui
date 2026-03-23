// Package tui — unit tests for task detail helper functions.
package tui

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
)

// ── detailLabel ───────────────────────────────────────────────────────────────

func TestDetailLabel_PadsToWidth(t *testing.T) {
	label := detailLabel("Due")
	// The label should be padded to at least 10 characters of visible text.
	// We strip tview tags before measuring.
	stripped := stripTviewTags(label)
	if len(stripped) < 10 {
		t.Errorf("detailLabel('Due') stripped = %q (len %d), want at least 10", stripped, len(stripped))
	}
}

func TestDetailLabel_LongString_NoTruncation(t *testing.T) {
	label := detailLabel("LongFieldName")
	stripped := stripTviewTags(label)
	if !strings.Contains(stripped, "LongFieldName") {
		t.Errorf("detailLabel('LongFieldName') stripped = %q, expected original text", stripped)
	}
}

// ── sectionHeader ─────────────────────────────────────────────────────────────

func TestSectionHeader_ContainsTitle(t *testing.T) {
	got := sectionHeader("Dates")
	stripped := stripTviewTags(got)
	if !strings.Contains(stripped, "Dates") {
		t.Errorf("sectionHeader('Dates') stripped = %q, expected 'Dates'", stripped)
	}
	if !strings.Contains(stripped, "─") {
		t.Errorf("sectionHeader stripped = %q, expected '─' divider chars", stripped)
	}
}

// ── statusBadge / priorityBadge ───────────────────────────────────────────────

func TestStatusBadge_ContainsStatus(t *testing.T) {
	got := statusBadge("in progress")
	stripped := stripTviewTags(got)
	if !strings.Contains(stripped, "in progress") {
		t.Errorf("statusBadge stripped = %q, expected 'in progress'", stripped)
	}
	if !strings.Contains(stripped, "●") {
		t.Errorf("statusBadge stripped = %q, expected dot '●'", stripped)
	}
}

func TestPriorityBadge_ContainsPriority(t *testing.T) {
	priorities := []string{"urgent", "high", "normal", "low", "unknown"}
	for _, p := range priorities {
		got := priorityBadge(p)
		stripped := stripTviewTags(got)
		if !strings.Contains(stripped, p) {
			t.Errorf("priorityBadge(%q) stripped = %q, expected priority name", p, stripped)
		}
	}
}

// ── statusDotColor ────────────────────────────────────────────────────────────

func TestStatusDotColor_ParsesHex(t *testing.T) {
	c := statusDotColor("#1e90ff", "custom")
	r, g, b := c.RGB()
	if r != 30 || g != 144 || b != 255 {
		t.Errorf("statusDotColor('#1e90ff') RGB = (%d,%d,%d), want (30,144,255)", r, g, b)
	}
}

func TestStatusDotColor_InvalidHex_FallsBack(t *testing.T) {
	c := statusDotColor("notacolor", "custom")
	if c == tcell.ColorDefault {
		t.Errorf("statusDotColor('notacolor') should not be ColorDefault")
	}
}

func TestStatusDotColor_EmptyColor_FallsBack(t *testing.T) {
	c := statusDotColor("", "open")
	if c == tcell.ColorDefault {
		t.Errorf("statusDotColor('') should not be ColorDefault")
	}
}

func TestStatusDotColor_ClosedType_Muted(t *testing.T) {
	c := statusDotColor("", "closed")
	if c != ColorTextMuted {
		t.Errorf("statusDotColor('', 'closed') = %v, want ColorTextMuted", c)
	}
}

// ── statusTypeLabel ───────────────────────────────────────────────────────────

func TestStatusTypeLabel(t *testing.T) {
	tests := []struct {
		t    string
		want string
	}{
		{"open", "open"},
		{"custom", "in-progress"},
		{"closed", "closed"},
		{"done", "closed"},
		{"other", "other"},
	}
	for _, tt := range tests {
		got := statusTypeLabel(tt.t)
		if got != tt.want {
			t.Errorf("statusTypeLabel(%q) = %q, want %q", tt.t, got, tt.want)
		}
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

// stripTviewTags removes tview colour/style tags (e.g. "[red]", "[#rrggbb]", "[-]")
// from a string so we can assert on visible text only.
func stripTviewTags(s string) string {
	var out strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '[':
			inTag = true
		case r == ']' && inTag:
			inTag = false
		case !inTag:
			out.WriteRune(r)
		}
	}
	return out.String()
}
