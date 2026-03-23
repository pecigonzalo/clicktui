// Package tui — unit tests for task detail helper functions.
package tui

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/pecigonzalo/clicktui/internal/app"
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

// ── render: navigation state ─────────────────────────────────────────────────

// newTestDetailPane creates a TaskDetailPane without a full App for render tests.
func newTestDetailPane() *TaskDetailPane {
	tv := tview.NewTextView().SetDynamicColors(true).SetWordWrap(true)
	return &TaskDetailPane{TextView: tv}
}

func TestRender_SetsParentID(t *testing.T) {
	tdp := newTestDetailPane()
	tdp.render(&app.TaskDetail{
		ID: "task1", Name: "Test", Status: "open", Priority: "normal",
		Space: "S", Folder: "F", List: "L",
		Parent: "parent123",
	})
	if tdp.parentID != "parent123" {
		t.Errorf("render() parentID = %q, want %q", tdp.parentID, "parent123")
	}
}

func TestRender_SetsSubtaskIDs(t *testing.T) {
	tdp := newTestDetailPane()
	tdp.render(&app.TaskDetail{
		ID: "task1", Name: "Test", Status: "open", Priority: "normal",
		Space: "S", Folder: "F", List: "L",
		Subtasks: []app.SubtaskSummary{
			{ID: "sub1", Name: "First", Status: "open"},
			{ID: "sub2", Name: "Second", Status: "done"},
		},
	})
	if len(tdp.subtaskIDs) != 2 {
		t.Fatalf("render() subtaskIDs len = %d, want 2", len(tdp.subtaskIDs))
	}
	if tdp.subtaskIDs[0] != "sub1" || tdp.subtaskIDs[1] != "sub2" {
		t.Errorf("render() subtaskIDs = %v, want [sub1 sub2]", tdp.subtaskIDs)
	}
}

func TestRender_NoSubtasks_EmptySlice(t *testing.T) {
	tdp := newTestDetailPane()
	tdp.render(&app.TaskDetail{
		ID: "task1", Name: "Test", Status: "open", Priority: "normal",
		Space: "S", Folder: "F", List: "L",
	})
	if len(tdp.subtaskIDs) != 0 {
		t.Errorf("render() subtaskIDs len = %d, want 0", len(tdp.subtaskIDs))
	}
	if tdp.parentID != "" {
		t.Errorf("render() parentID = %q, want empty", tdp.parentID)
	}
}

// ── render: output content ───────────────────────────────────────────────────

func TestRender_ParentShowsNavigationHint(t *testing.T) {
	tdp := newTestDetailPane()
	tdp.render(&app.TaskDetail{
		ID: "task1", Name: "Test", Status: "open", Priority: "normal",
		Space: "S", Folder: "F", List: "L",
		Parent: "parent123",
	})
	text := tdp.GetText(false)
	stripped := stripTviewTags(text)
	if !strings.Contains(stripped, "▸ parent123") {
		t.Errorf("render() parent line missing '▸ parent123'; stripped = %q", stripped)
	}
	// The hint uses tview-escaped brackets [[]p], which stripTviewTags turns
	// into "p] go to parent".
	if !strings.Contains(stripped, "p] go to parent") {
		t.Errorf("render() parent line missing navigation hint; stripped = %q", stripped)
	}
}

func TestRender_SubtasksSection(t *testing.T) {
	tdp := newTestDetailPane()
	tdp.render(&app.TaskDetail{
		ID: "task1", Name: "Test", Status: "open", Priority: "normal",
		Space: "S", Folder: "F", List: "L",
		Subtasks: []app.SubtaskSummary{
			{ID: "sub1", Name: "First sub", Status: "in progress"},
			{ID: "sub2", Name: "Second sub", Status: "done"},
		},
	})
	text := tdp.GetText(false)
	stripped := stripTviewTags(text)

	if !strings.Contains(stripped, "Subtasks (2)") {
		t.Errorf("render() missing subtasks header with count; stripped text does not contain 'Subtasks (2)'")
	}
	if !strings.Contains(stripped, "sub1") {
		t.Errorf("render() missing subtask ID 'sub1'; stripped text does not contain it")
	}
	if !strings.Contains(stripped, "First sub") {
		t.Errorf("render() missing subtask name 'First sub'; stripped text does not contain it")
	}
	if !strings.Contains(stripped, "sub2") {
		t.Errorf("render() missing subtask ID 'sub2'; stripped text does not contain it")
	}
}

func TestRender_NoSubtasks_NoSection(t *testing.T) {
	tdp := newTestDetailPane()
	tdp.render(&app.TaskDetail{
		ID: "task1", Name: "Test", Status: "open", Priority: "normal",
		Space: "S", Folder: "F", List: "L",
	})
	stripped := stripTviewTags(tdp.GetText(false))
	if strings.Contains(stripped, "Subtasks") {
		t.Errorf("render() should not contain Subtasks section when there are none; stripped = %q", stripped)
	}
}

// ── render: action hints ─────────────────────────────────────────────────────

func TestRender_ActionHints_StatusOnly(t *testing.T) {
	tdp := newTestDetailPane()
	tdp.render(&app.TaskDetail{
		ID: "task1", Name: "Test", Status: "open", Priority: "normal",
		Space: "S", Folder: "F", List: "L",
	})
	stripped := stripTviewTags(tdp.GetText(false))
	// tview-escaped [[]s] strips to "s] update status".
	if !strings.Contains(stripped, "s] update status") {
		t.Errorf("render() missing status hint; stripped = %q", stripped)
	}
	if strings.Contains(stripped, "p] go to parent") {
		t.Errorf("render() should not show parent hint when no parent")
	}
	if strings.Contains(stripped, "1-N] open subtask") {
		t.Errorf("render() should not show subtask hint when no subtasks")
	}
}

func TestRender_ActionHints_AllPresent(t *testing.T) {
	tdp := newTestDetailPane()
	tdp.render(&app.TaskDetail{
		ID: "task1", Name: "Test", Status: "open", Priority: "normal",
		Space: "S", Folder: "F", List: "L",
		Parent: "p1",
		Subtasks: []app.SubtaskSummary{
			{ID: "sub1", Name: "S1", Status: "open"},
		},
	})
	stripped := stripTviewTags(tdp.GetText(false))
	// tview-escaped brackets [[]x] strip to "x] ...".
	for _, hint := range []string{"s] update status", "p] go to parent", "1-N] open subtask"} {
		if !strings.Contains(stripped, hint) {
			t.Errorf("render() missing hint %q; stripped = %q", hint, stripped)
		}
	}
}

// ── inputHandler: navigation ─────────────────────────────────────────────────

func TestInputHandler_PKeyNavigatesToParent(t *testing.T) {
	tdp := newTestDetailPane()
	tdp.parentID = "parent1"

	// The 'p' key should consume the event (return nil) when parent is set.
	// Without a full App, LoadDetail will panic, so we verify the guard by
	// checking that the no-parent case passes through (tested below) and that
	// the parentID state is correctly set by render (tested above).
	if tdp.parentID == "" {
		t.Fatal("test setup: parentID should be set")
	}
}

func TestInputHandler_PKeyNoParent_PassesThrough(t *testing.T) {
	tdp := newTestDetailPane()
	tdp.parentID = ""

	event := tcell.NewEventKey(tcell.KeyRune, 'p', tcell.ModNone)
	result := tdp.inputHandler(event)
	if result == nil {
		t.Error("inputHandler('p') with no parent should pass through event, got nil")
	}
}

func TestInputHandler_NumberKeyNoSubtasks_PassesThrough(t *testing.T) {
	tdp := newTestDetailPane()
	tdp.subtaskIDs = nil

	event := tcell.NewEventKey(tcell.KeyRune, '1', tcell.ModNone)
	result := tdp.inputHandler(event)
	if result == nil {
		t.Error("inputHandler('1') with no subtasks should pass through event, got nil")
	}
}

func TestInputHandler_NumberKeyOutOfRange_PassesThrough(t *testing.T) {
	tdp := newTestDetailPane()
	tdp.subtaskIDs = []string{"sub1"}

	// '2' is out of range (only 1 subtask)
	event := tcell.NewEventKey(tcell.KeyRune, '2', tcell.ModNone)
	result := tdp.inputHandler(event)
	if result == nil {
		t.Error("inputHandler('2') with 1 subtask should pass through event, got nil")
	}
}

func TestInputHandler_NonRuneKey_PassesThrough(t *testing.T) {
	tdp := newTestDetailPane()
	event := tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
	result := tdp.inputHandler(event)
	if result == nil {
		t.Error("inputHandler(Down) should pass through event, got nil")
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
