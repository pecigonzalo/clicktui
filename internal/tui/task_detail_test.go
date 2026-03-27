// Package tui — unit tests for task detail helper functions.
package tui

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pecigonzalo/clicktui/internal/app"
)

func newTestDetailPane(t *testing.T) *TaskDetailPane {
	t.Helper()
	tv := tview.NewTextView().SetDynamicColors(true).SetWordWrap(true)
	return &TaskDetailPane{TextView: tv}
}

func minimalDetail() *app.TaskDetail {
	return &app.TaskDetail{
		ID:       "task1",
		Name:     "Test",
		Status:   "open",
		Priority: "normal",
		Space:    "S",
		Folder:   "F",
		List:     "L",
	}
}

func TestDetailLabel_PadsToWidth(t *testing.T) {
	stripped := stripTviewTags(detailLabel("Due"))
	assert.GreaterOrEqual(t, len(stripped), 12)
}

func TestDetailLabel_LongString_NoTruncation(t *testing.T) {
	stripped := stripTviewTags(detailLabel("LongFieldName"))
	assert.Contains(t, stripped, "LongFieldName")
}

func TestSectionHeader_ContainsTitle(t *testing.T) {
	stripped := stripTviewTags(sectionHeader("Dates", ""))
	assert.Contains(t, stripped, "Dates")
	assert.Contains(t, stripped, "─")
}

func TestStatusBadge_ContainsStatus(t *testing.T) {
	stripped := stripTviewTags(statusBadge("in progress"))
	assert.Contains(t, stripped, "in progress")
	assert.Contains(t, stripped, "●")
}

func TestStatusBadgeColored_UsesHexColor(t *testing.T) {
	got := statusBadgeColored("in progress", "#1e90ff")
	assert.Contains(t, got, "#1e90ff")
	assert.Contains(t, stripTviewTags(got), "in progress")
}

func TestStatusBadgeColored_FallsBackOnEmpty(t *testing.T) {
	assert.Contains(t, stripTviewTags(statusBadgeColored("open", "")), "open")
}

func TestPriorityBadge_ContainsPriority(t *testing.T) {
	for _, p := range []string{"urgent", "high", "normal", "low", "unknown"} {
		t.Run(p, func(t *testing.T) {
			t.Parallel()
			assert.Contains(t, stripTviewTags(priorityBadge(p)), p)
		})
	}
}

func TestStatusDotColor_ParsesHex(t *testing.T) {
	r, g, b := statusDotColor("#1e90ff", "custom").RGB()
	assert.Equal(t, int32(30), r)
	assert.Equal(t, int32(144), g)
	assert.Equal(t, int32(255), b)
}

func TestStatusDotColor_ClosedType_Muted(t *testing.T) {
	assert.Equal(t, ColorTextMuted, statusDotColor("", "closed"))
}

func TestStatusTypeLabel(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "open", in: "open", want: "open"},
		{name: "custom", in: "custom", want: "in-progress"},
		{name: "closed", in: "closed", want: "closed"},
		{name: "done", in: "done", want: "closed"},
		{name: "other", in: "other", want: "other"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, statusTypeLabel(tc.in))
		})
	}
}

func TestRender_SubtasksSection(t *testing.T) {
	tdp := newTestDetailPane(t)
	d := minimalDetail()
	d.Subtasks = []app.SubtaskSummary{{ID: "sub1", Name: "First sub", Status: "in progress"}, {ID: "sub2", Name: "Second sub", Status: "done"}}

	tdp.render(d)
	stripped := stripTviewTags(tdp.GetText(false))
	assert.Contains(t, stripped, "Subtasks (2)")
	assert.Contains(t, stripped, "sub1")
	assert.Contains(t, stripped, "First sub")
	assert.Contains(t, stripped, "sub2")
}

func TestRender_NoSubtasks_NoSection(t *testing.T) {
	tdp := newTestDetailPane(t)
	tdp.render(minimalDetail())
	assert.NotContains(t, stripTviewTags(tdp.GetText(false)), "Subtasks")
}

func TestSectionHeader_WithIcon(t *testing.T) {
	stripped := stripTviewTags(sectionHeader("Dates", "CAL"))
	assert.Contains(t, stripped, "CAL Dates")
	assert.True(t, strings.HasPrefix(stripped, icons.SectionCorner))
}

func TestRender_LocationBreadcrumb(t *testing.T) {
	tdp := newTestDetailPane(t)
	d := minimalDetail()
	d.Space, d.Folder, d.List = "Engineering", "Sprint 23", "Current Sprint"
	tdp.render(d)
	stripped := stripTviewTags(tdp.GetText(false))
	assert.Contains(t, stripped, "Engineering")
	assert.Contains(t, stripped, "Sprint 23")
	assert.Contains(t, stripped, "Current Sprint")
}

func TestRender_DescriptionGutter(t *testing.T) {
	tdp := newTestDetailPane(t)
	d := minimalDetail()
	d.Description = "Line one\nLine two"
	tdp.render(d)
	stripped := stripTviewTags(tdp.GetText(false))
	assert.Contains(t, stripped, "│ Line one")
	assert.Contains(t, stripped, "│ Line two")
}

func TestRender_DatesSection_SecondaryLine(t *testing.T) {
	tdp := newTestDetailPane(t)
	d := minimalDetail()
	d.DueDate, d.StartDate = "2024-01-15", "2024-01-10"
	d.DateCreated, d.DateUpdated = "2024-01-05", "2024-01-14"
	tdp.render(d)
	stripped := stripTviewTags(tdp.GetText(false))
	assert.Contains(t, stripped, "2024-01-15")
	assert.Contains(t, stripped, "2024-01-10")
	assert.Contains(t, stripped, "Created 2024-01-05")
	assert.Contains(t, stripped, "Updated 2024-01-14")
	assert.Contains(t, stripped, "·")
}

func TestRender_StatusUsesColor(t *testing.T) {
	tdp := newTestDetailPane(t)
	d := minimalDetail()
	d.Status = "in progress"
	d.StatusColor = "#ff6600"
	tdp.render(d)
	assert.Contains(t, tdp.GetText(false), "#ff6600")
}

func TestRender_NoActionHints(t *testing.T) {
	tdp := newTestDetailPane(t)
	tdp.render(minimalDetail())
	assert.NotContains(t, stripTviewTags(tdp.GetText(false)), "update status")
}

func TestRenderWithSelector_NoBottomSelectorBlock(t *testing.T) {
	tdp := newTestDetailPane(t)
	d := minimalDetail()
	tdp.detail = d
	tdp.buildFields(d)
	tdp.selectedIdx = 0
	tdp.renderWithSelector()
	assert.NotContains(t, stripTviewTags(tdp.GetText(false)), "Select Field")
}

func TestRenderWithSelector_InlineHighlightOnSelectedField(t *testing.T) {
	tdp := newTestDetailPane(t)
	d := minimalDetail()
	d.Description = "hello"
	tdp.detail = d
	tdp.buildFields(d)

	selected := -1
	for i, f := range tdp.fields {
		if f.label == "Description" {
			selected = i
			break
		}
	}
	require.GreaterOrEqual(t, selected, 0)

	tdp.selectedIdx = selected
	tdp.renderWithSelector()
	assert.Contains(t, stripTviewTags(tdp.GetText(false)), "> │ hello")
}

func TestInputHandler_NonRuneKey_PassesThrough(t *testing.T) {
	tdp := newTestDetailPane(t)
	event := tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
	assert.NotNil(t, tdp.inputHandler(event))
}

func TestInputHandler_SelectorMode_RuneJMovesSelection(t *testing.T) {
	tdp := newTestDetailPane(t)
	d := minimalDetail()
	tdp.detail = d
	tdp.buildFields(d)
	require.GreaterOrEqual(t, len(tdp.fields), 2)
	tdp.selectorMode = true
	tdp.selectedIdx = 0

	result := tdp.inputHandler(tcell.NewEventKey(tcell.KeyRune, 'j', tcell.ModNone))

	assert.Nil(t, result)
	assert.Equal(t, 1, tdp.selectedIdx)
}

func TestBuildFields_AllPopulated(t *testing.T) {
	tdp := newTestDetailPane(t)
	d := &app.TaskDetail{
		ID:          "task1",
		CustomID:    "PROJ-42",
		URL:         "https://app.clickup.com/t/task1",
		DueDate:     "2024-06-01",
		StartDate:   "2024-05-15",
		Parent:      "parent1",
		Description: "A task description.",
		Assignees:   []string{"alice"},
		AssigneeIDs: []int{101},
		Subtasks:    []app.SubtaskSummary{{ID: "sub1", Name: "Subtask A", Status: "open"}, {ID: "sub2", Name: "Subtask B", Status: "done"}},
	}
	tdp.buildFields(d)

	require.Len(t, tdp.fields, 10)
	// fields[0] = Task ID
	assert.Equal(t, "Task ID", tdp.fields[0].label)
	assert.Equal(t, fieldCopy, tdp.fields[0].kind)
	// fields[1] = Custom ID
	assert.Equal(t, "Custom ID", tdp.fields[1].label)
	assert.Equal(t, fieldCopy, tdp.fields[1].kind)
	// fields[2] = Assignees
	assert.Equal(t, "Assignees", tdp.fields[2].label)
	assert.Equal(t, fieldEdit, tdp.fields[2].kind)
	assert.Equal(t, editTypeAssignee, tdp.fields[2].edit)
	// fields[3] = Due Date
	assert.Equal(t, "Due Date", tdp.fields[3].label)
	assert.Equal(t, fieldEdit, tdp.fields[3].kind)
	assert.Equal(t, editTypeDate, tdp.fields[3].edit)
	assert.True(t, tdp.fields[3].hasValue)
	// fields[4] = Start Date
	assert.Equal(t, "Start Date", tdp.fields[4].label)
	assert.Equal(t, fieldEdit, tdp.fields[4].kind)
	assert.Equal(t, editTypeDate, tdp.fields[4].edit)
	assert.True(t, tdp.fields[4].hasValue)
	// fields[5] = Parent
	assert.Equal(t, "Parent", tdp.fields[5].label)
	assert.Equal(t, fieldNavigate, tdp.fields[5].kind)
	// fields[6] = sub1 subtask
	assert.Equal(t, fieldNavigate, tdp.fields[6].kind)
	// fields[7] = sub2 subtask
	assert.Equal(t, fieldNavigate, tdp.fields[7].kind)
	// fields[8] = URL
	assert.Equal(t, "URL", tdp.fields[8].label)
	assert.Equal(t, fieldOpen, tdp.fields[8].kind)
	// fields[9] = Description
	assert.Equal(t, "Description", tdp.fields[9].label)
	assert.Equal(t, fieldEdit, tdp.fields[9].kind)
	assert.Equal(t, editTypeTextArea, tdp.fields[9].edit)
}

func TestBuildFields_EditableFieldsAlwaysPresent(t *testing.T) {
	tdp := newTestDetailPane(t)
	d := &app.TaskDetail{ID: "task1", Name: "Minimal Task", Status: "open"}
	tdp.buildFields(d)

	require.Len(t, tdp.fields, 5)
	labels := map[string]bool{}
	for _, f := range tdp.fields {
		labels[f.label] = true
	}
	for _, want := range []string{"Due Date", "Start Date", "Description", "Assignees"} {
		assert.True(t, labels[want])
	}
	for _, f := range tdp.fields[1:] {
		if f.kind == fieldEdit {
			assert.False(t, f.hasValue)
		}
	}
}

func TestBuildFields_EmptyValuesExcluded_ReadOnly(t *testing.T) {
	tdp := newTestDetailPane(t)
	tdp.buildFields(&app.TaskDetail{ID: "task1", Name: "Minimal Task", Status: "open"})

	for _, f := range tdp.fields {
		assert.NotContains(t, []string{"Custom ID", "URL", "Parent"}, f.label)
	}
}

func TestBuildFields_SubtasksIncluded(t *testing.T) {
	tdp := newTestDetailPane(t)
	d := &app.TaskDetail{ID: "task1", Subtasks: []app.SubtaskSummary{{ID: "s1", Name: "Sub One"}, {ID: "s2", Name: "Sub Two"}, {ID: "s3", Name: "Sub Three"}}}
	tdp.buildFields(d)

	// Task ID + Assignees + Due Date + Start Date + 3 subtasks + Description = 8
	require.Len(t, tdp.fields, 8)
	// Subtasks occupy indices 4, 5, 6 (after Task ID, Assignees, Due Date, Start Date)
	for _, f := range tdp.fields[4:7] {
		assert.Equal(t, fieldNavigate, f.kind)
	}
}

func TestBuildFields_ResetsBetweenCalls(t *testing.T) {
	tdp := newTestDetailPane(t)
	tdp.buildFields(&app.TaskDetail{ID: "task1", DueDate: "2024-06-01", URL: "https://example.com"})
	first := len(tdp.fields)
	tdp.buildFields(&app.TaskDetail{ID: "task2"})
	second := len(tdp.fields)

	assert.NotEqual(t, first, second)
	assert.Equal(t, 5, second)
}
