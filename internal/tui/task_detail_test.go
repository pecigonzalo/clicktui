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
	// The label should be padded to at least 12 characters of visible text.
	// We strip tview tags before measuring.
	stripped := stripTviewTags(label)
	if len(stripped) < 12 {
		t.Errorf("detailLabel('Due') stripped = %q (len %d), want at least 12", stripped, len(stripped))
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
	got := sectionHeader("Dates", "")
	stripped := stripTviewTags(got)
	if !strings.Contains(stripped, "Dates") {
		t.Errorf("sectionHeader('Dates') stripped = %q, expected 'Dates'", stripped)
	}
	if !strings.Contains(stripped, "─") {
		t.Errorf("sectionHeader stripped = %q, expected '─' divider chars", stripped)
	}
}

// ── statusBadge / statusBadgeColored / priorityBadge ─────────────────────────

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

func TestStatusBadgeColored_UsesHexColor(t *testing.T) {
	got := statusBadgeColored("in progress", "#1e90ff")
	// The result should contain the hex colour tag.
	if !strings.Contains(got, "#1e90ff") {
		t.Errorf("statusBadgeColored with hex color should contain hex tag; got = %q", got)
	}
	stripped := stripTviewTags(got)
	if !strings.Contains(stripped, "in progress") {
		t.Errorf("statusBadgeColored stripped = %q, expected 'in progress'", stripped)
	}
}

func TestStatusBadgeColored_FallsBackOnEmpty(t *testing.T) {
	got := statusBadgeColored("open", "")
	// Should fall back to ColorBadgeStatus (aqua) — not contain a hex color tag
	// for the empty input. Just verify it contains the status text.
	stripped := stripTviewTags(got)
	if !strings.Contains(stripped, "open") {
		t.Errorf("statusBadgeColored fallback stripped = %q, expected 'open'", stripped)
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

// ── render helpers ────────────────────────────────────────────────────────────

// newTestDetailPane creates a TaskDetailPane without a full App for render tests.
func newTestDetailPane() *TaskDetailPane {
	tv := tview.NewTextView().SetDynamicColors(true).SetWordWrap(true)
	return &TaskDetailPane{TextView: tv}
}

// ── render: output content ───────────────────────────────────────────────────

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

// ── sectionHeader with icon ───────────────────────────────────────────────────

func TestSectionHeader_WithIcon(t *testing.T) {
	got := sectionHeader("Dates", "CAL")
	stripped := stripTviewTags(got)
	if !strings.Contains(stripped, "CAL Dates") {
		t.Errorf("sectionHeader with icon stripped = %q, expected 'CAL Dates'", stripped)
	}
	// Should start with the SectionCorner prefix.
	if !strings.HasPrefix(stripped, icons.SectionCorner) {
		t.Errorf("sectionHeader with icon should start with SectionCorner %q; stripped = %q", icons.SectionCorner, stripped)
	}
}

func TestSectionHeader_WithoutIcon(t *testing.T) {
	got := sectionHeader("Dates", "")
	stripped := stripTviewTags(got)
	if !strings.HasPrefix(stripped, icons.SectionCorner) {
		t.Errorf("sectionHeader without icon should start with SectionCorner %q; stripped = %q", icons.SectionCorner, stripped)
	}
}

// ── render: location breadcrumb ──────────────────────────────────────────────

func TestRender_LocationBreadcrumb(t *testing.T) {
	tdp := newTestDetailPane()
	tdp.render(&app.TaskDetail{
		ID: "task1", Name: "Test", Status: "open", Priority: "normal",
		Space: "Engineering", Folder: "Sprint 23", List: "Current Sprint",
	})
	stripped := stripTviewTags(tdp.GetText(false))
	// Should contain breadcrumb with separator, not separate rows.
	if !strings.Contains(stripped, "Engineering") {
		t.Errorf("render() missing Space in breadcrumb; stripped = %q", stripped)
	}
	if !strings.Contains(stripped, "Sprint 23") {
		t.Errorf("render() missing Folder in breadcrumb; stripped = %q", stripped)
	}
	if !strings.Contains(stripped, "Current Sprint") {
		t.Errorf("render() missing List in breadcrumb; stripped = %q", stripped)
	}
	// Should NOT have separate "Space", "Folder", "List" labels.
	if strings.Contains(stripped, "Space ") && strings.Contains(stripped, "Folder ") {
		t.Errorf("render() should use breadcrumb, not separate rows; stripped = %q", stripped)
	}
}

func TestRender_LocationBreadcrumb_SkipsEmpty(t *testing.T) {
	tdp := newTestDetailPane()
	tdp.render(&app.TaskDetail{
		ID: "task1", Name: "Test", Status: "open", Priority: "normal",
		Space: "Engineering", Folder: "", List: "Current Sprint",
	})
	stripped := stripTviewTags(tdp.GetText(false))
	// Empty folder should be omitted from breadcrumb.
	if !strings.Contains(stripped, "Engineering") {
		t.Errorf("render() missing Space in breadcrumb; stripped = %q", stripped)
	}
	if !strings.Contains(stripped, "Current Sprint") {
		t.Errorf("render() missing List in breadcrumb; stripped = %q", stripped)
	}
}

// ── render: description gutter ───────────────────────────────────────────────

func TestRender_DescriptionGutter(t *testing.T) {
	tdp := newTestDetailPane()
	tdp.render(&app.TaskDetail{
		ID: "task1", Name: "Test", Status: "open", Priority: "normal",
		Space: "S", Folder: "F", List: "L",
		Description: "Line one\nLine two",
	})
	stripped := stripTviewTags(tdp.GetText(false))
	// Each description line should be prefixed with a gutter character.
	if !strings.Contains(stripped, "│ Line one") {
		t.Errorf("render() description missing gutter for line 1; stripped = %q", stripped)
	}
	if !strings.Contains(stripped, "│ Line two") {
		t.Errorf("render() description missing gutter for line 2; stripped = %q", stripped)
	}
}

// ── render: dates section ────────────────────────────────────────────────────

func TestRender_DatesSection_SecondaryLine(t *testing.T) {
	tdp := newTestDetailPane()
	tdp.render(&app.TaskDetail{
		ID: "task1", Name: "Test", Status: "open", Priority: "normal",
		Space: "S", Folder: "F", List: "L",
		DueDate: "2024-01-15", StartDate: "2024-01-10",
		DateCreated: "2024-01-05", DateUpdated: "2024-01-14",
	})
	stripped := stripTviewTags(tdp.GetText(false))
	// Primary dates should be on their own labeled rows.
	if !strings.Contains(stripped, "Due") || !strings.Contains(stripped, "2024-01-15") {
		t.Errorf("render() missing due date; stripped = %q", stripped)
	}
	if !strings.Contains(stripped, "Start") || !strings.Contains(stripped, "2024-01-10") {
		t.Errorf("render() missing start date; stripped = %q", stripped)
	}
	// Secondary dates should be on one muted line with · separator.
	if !strings.Contains(stripped, "Created 2024-01-05") {
		t.Errorf("render() missing created date on secondary line; stripped = %q", stripped)
	}
	if !strings.Contains(stripped, "Updated 2024-01-14") {
		t.Errorf("render() missing updated date on secondary line; stripped = %q", stripped)
	}
	if !strings.Contains(stripped, "·") {
		t.Errorf("render() missing · separator between created/updated; stripped = %q", stripped)
	}
}

// ── render: status uses StatusColor ──────────────────────────────────────────

func TestRender_StatusUsesColor(t *testing.T) {
	tdp := newTestDetailPane()
	tdp.render(&app.TaskDetail{
		ID: "task1", Name: "Test", Status: "in progress", StatusColor: "#ff6600",
		Priority: "normal", Space: "S", Folder: "F", List: "L",
	})
	text := tdp.GetText(false)
	// The rendered text should contain the hex colour from StatusColor.
	if !strings.Contains(text, "#ff6600") {
		t.Errorf("render() should use StatusColor hex in output; text = %q", text)
	}
}

// ── render: action hints removed from body ──────────────────────────────────

func TestRender_NoActionHints(t *testing.T) {
	tdp := newTestDetailPane()
	tdp.render(&app.TaskDetail{
		ID: "task1", Name: "Test", Status: "open", Priority: "normal",
		Space: "S", Folder: "F", List: "L",
	})
	stripped := stripTviewTags(tdp.GetText(false))
	// Action hints were moved to the global footer; they should not appear in
	// the rendered detail body.
	if strings.Contains(stripped, "update status") {
		t.Errorf("render() should not contain action hints in body; stripped = %q", stripped)
	}
}

// ── inputHandler ─────────────────────────────────────────────────────────────

func TestInputHandler_NonRuneKey_PassesThrough(t *testing.T) {
	tdp := newTestDetailPane()
	event := tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
	result := tdp.inputHandler(event)
	if result == nil {
		t.Error("inputHandler(Down) should pass through event, got nil")
	}
}

// ── buildFields ──────────────────────────────────────────────────────────────

func TestBuildFields_AllPopulated(t *testing.T) {
	tdp := newTestDetailPane()
	d := &app.TaskDetail{
		ID:          "task1",
		CustomID:    "PROJ-42",
		URL:         "https://app.clickup.com/t/task1",
		DueDate:     "2024-06-01",
		StartDate:   "2024-05-15",
		Parent:      "parent1",
		Description: "A task description.",
		Subtasks: []app.SubtaskSummary{
			{ID: "sub1", Name: "Subtask A", Status: "open"},
			{ID: "sub2", Name: "Subtask B", Status: "done"},
		},
	}
	tdp.buildFields(d)

	// Expected fields: Task ID, Custom ID, URL, Due Date, Start Date, Parent,
	// Description, + 2 subtasks = 9.
	if got := len(tdp.fields); got != 9 {
		t.Fatalf("buildFields() produced %d fields, want 9", got)
	}

	// Verify first field is always Task ID.
	if f := tdp.fields[0]; f.label != "Task ID" || f.value != "task1" || f.kind != fieldCopy {
		t.Errorf("fields[0] = %+v, want Task ID / task1 / fieldCopy", f)
	}
	// Verify URL field has fieldOpen kind.
	if f := tdp.fields[2]; f.label != "URL" || f.kind != fieldOpen {
		t.Errorf("fields[2] = %+v, want URL / fieldOpen", f)
	}
	// Verify Parent field has fieldNavigate kind.
	if f := tdp.fields[5]; f.label != "Parent" || f.kind != fieldNavigate {
		t.Errorf("fields[5] = %+v, want Parent / fieldNavigate", f)
	}
	// Verify subtask fields have fieldNavigate kind and use subtask ID as value.
	if f := tdp.fields[7]; f.kind != fieldNavigate || f.value != "sub1" {
		t.Errorf("fields[7] = %+v, want fieldNavigate with value sub1", f)
	}
	if f := tdp.fields[8]; f.kind != fieldNavigate || f.value != "sub2" {
		t.Errorf("fields[8] = %+v, want fieldNavigate with value sub2", f)
	}
}

func TestBuildFields_EmptyValuesExcluded(t *testing.T) {
	tdp := newTestDetailPane()
	d := &app.TaskDetail{
		ID:     "task1",
		Name:   "Minimal Task",
		Status: "open",
	}
	tdp.buildFields(d)

	// Only Task ID should be present — all optional fields are empty.
	if got := len(tdp.fields); got != 1 {
		t.Fatalf("buildFields() with minimal detail produced %d fields, want 1", got)
	}
	if f := tdp.fields[0]; f.label != "Task ID" {
		t.Errorf("fields[0].label = %q, want 'Task ID'", f.label)
	}
}

func TestBuildFields_SubtasksIncluded(t *testing.T) {
	tdp := newTestDetailPane()
	d := &app.TaskDetail{
		ID: "task1",
		Subtasks: []app.SubtaskSummary{
			{ID: "s1", Name: "Sub One", Status: "open"},
			{ID: "s2", Name: "Sub Two", Status: "done"},
			{ID: "s3", Name: "Sub Three", Status: "open"},
		},
	}
	tdp.buildFields(d)

	// 1 (Task ID) + 3 subtasks = 4.
	if got := len(tdp.fields); got != 4 {
		t.Fatalf("buildFields() with 3 subtasks produced %d fields, want 4", got)
	}
	// All subtask fields should be navigate kind.
	for i := 1; i < 4; i++ {
		if tdp.fields[i].kind != fieldNavigate {
			t.Errorf("fields[%d].kind = %d, want fieldNavigate", i, tdp.fields[i].kind)
		}
	}
}

func TestBuildFields_ResetsBetweenCalls(t *testing.T) {
	tdp := newTestDetailPane()
	d1 := &app.TaskDetail{
		ID:      "task1",
		DueDate: "2024-06-01",
		URL:     "https://example.com",
	}
	tdp.buildFields(d1)
	first := len(tdp.fields)

	d2 := &app.TaskDetail{ID: "task2"}
	tdp.buildFields(d2)
	second := len(tdp.fields)

	if first == second {
		t.Errorf("buildFields should reset: first call had %d fields, second had %d (same)", first, second)
	}
	if second != 1 {
		t.Errorf("buildFields after minimal detail: got %d fields, want 1", second)
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
