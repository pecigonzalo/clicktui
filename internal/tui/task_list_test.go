// Package tui — unit tests for task list pane state logic.
package tui

import (
	"strings"
	"testing"

	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pecigonzalo/clicktui/internal/app"
)

func newTestTaskListPane(t *testing.T) *TaskListPane {
	t.Helper()
	table := tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0)
	table.SetBorder(true)
	return &TaskListPane{Table: table}
}

func sampleTasks(t *testing.T) []app.TaskSummary {
	t.Helper()
	return []app.TaskSummary{
		{ID: "t1", Name: "Fix login bug", Status: "in progress", StatusColor: "#1e90ff", Priority: "high", Parent: ""},
		{ID: "t2", Name: "Add dark mode", Status: "todo", StatusColor: "#aaaaaa", Priority: "normal", Parent: ""},
		{ID: "t3", Name: "Write tests", Status: "in progress", StatusColor: "#1e90ff", Priority: "urgent", Parent: ""},
		{ID: "t4", Name: "Update docs", Status: "done", StatusColor: "#00ff00", Priority: "low", Parent: "t1"},
		{ID: "t5", Name: "Deploy to staging", Status: "todo", StatusColor: "#aaaaaa", Priority: "high", Parent: ""},
	}
}

func sortableTasks(t *testing.T) []app.TaskSummary {
	t.Helper()
	return []app.TaskSummary{
		{ID: "t1", Name: "Fix login bug", Status: "in progress", Priority: "high", PriorityOrder: 2, Parent: ""},
		{ID: "t2", Name: "Add dark mode", Status: "todo", Priority: "normal", PriorityOrder: 3, Parent: ""},
		{ID: "t3", Name: "Write tests", Status: "in progress", Priority: "urgent", PriorityOrder: 1, Parent: ""},
		{ID: "t4", Name: "Update docs", Status: "done", Priority: "low", PriorityOrder: 4, Parent: "t1"},
		{ID: "t5", Name: "Deploy to staging", Status: "todo", Priority: "high", PriorityOrder: 2, Parent: ""},
	}
}

func taskIDs(t *testing.T, tasks []app.TaskSummary) []string {
	t.Helper()
	ids := make([]string, len(tasks))
	for i, task := range tasks {
		ids[i] = task.ID
	}
	return ids
}

func TestSelectedTaskID_ValidSelection(t *testing.T) {
	tlp := newTestTaskListPane(t)
	tlp.tasks = sampleTasks(t)
	tlp.render()
	tlp.Select(2, 0)
	assert.Equal(t, "t2", tlp.SelectedTaskID())
}

func TestSelectedTaskID_HeaderSelected(t *testing.T) {
	tlp := newTestTaskListPane(t)
	tlp.tasks = sampleTasks(t)
	tlp.render()
	tlp.Select(0, 0)
	assert.Empty(t, tlp.SelectedTaskID())
}

func TestSelectedTaskID_EmptyTaskList(t *testing.T) {
	tlp := newTestTaskListPane(t)
	assert.Empty(t, tlp.SelectedTaskID())
}

func TestApplyFilter_SetsFilteredTasks(t *testing.T) {
	tlp := newTestTaskListPane(t)
	all := sampleTasks(t)
	tlp.allTasks = all

	filtered := []app.TaskSummary{all[0], all[2]}
	query := app.TaskQuery{Fields: []app.FieldFilter{{Field: "status", Value: "in progress"}}}
	tlp.ApplyFilter(filtered, query)

	require.Len(t, tlp.tasks, 2)
	assert.Equal(t, []string{"t1", "t3"}, taskIDs(t, tlp.tasks))
	assert.NotNil(t, tlp.activeQuery)
}

func TestApplyFilter_EmptyQuery_ShowsAll(t *testing.T) {
	tlp := newTestTaskListPane(t)
	all := sampleTasks(t)
	tlp.allTasks = all
	tlp.activeQuery = &app.TaskQuery{FreeText: "something"}
	tlp.tasks = all[:1]

	tlp.ApplyFilter(nil, app.TaskQuery{})

	assert.Len(t, tlp.tasks, len(all))
	assert.Nil(t, tlp.activeQuery)
}

func TestClearFilter_RestoresAllTasks(t *testing.T) {
	tlp := newTestTaskListPane(t)
	all := sampleTasks(t)
	tlp.allTasks = all
	tlp.tasks = all[:2]
	tlp.activeQuery = &app.TaskQuery{FreeText: "fix"}

	tlp.ClearFilter()

	assert.Len(t, tlp.tasks, len(all))
	assert.Nil(t, tlp.activeQuery)
}

func TestApplyFilter_NilFiltered_SetsEmptySlice(t *testing.T) {
	tlp := newTestTaskListPane(t)
	tlp.allTasks = sampleTasks(t)

	tlp.ApplyFilter(nil, app.TaskQuery{FreeText: "zzz-no-match"})

	assert.Nil(t, tlp.tasks)
	assert.NotNil(t, tlp.activeQuery)
}

func TestRefreshCurrentTask_UpdatesAllTasks(t *testing.T) {
	tlp := newTestTaskListPane(t)
	all := sampleTasks(t)
	allCopy := append([]app.TaskSummary(nil), all...)
	tlp.allTasks = allCopy
	tlp.tasks = allCopy

	tlp.refreshCurrentTask("t2", "done", "#00ff00")

	for _, task := range tlp.allTasks {
		if task.ID == "t2" {
			assert.Equal(t, "done", task.Status)
			assert.Equal(t, "#00ff00", task.StatusColor)
			return
		}
	}
	require.FailNow(t, "task t2 not found in allTasks")
}

func TestRefreshCurrentTask_UpdatesDisplayedTasks(t *testing.T) {
	tlp := newTestTaskListPane(t)
	allCopy := append([]app.TaskSummary(nil), sampleTasks(t)...)
	tlp.allTasks = allCopy
	tlp.tasks = allCopy

	tlp.refreshCurrentTask("t1", "review", "#ff6600")

	for _, task := range tlp.tasks {
		if task.ID == "t1" {
			assert.Equal(t, "review", task.Status)
			return
		}
	}
	require.FailNow(t, "task t1 not found in displayed tasks")
}

func TestRefreshCurrentTask_NonexistentTask(t *testing.T) {
	tlp := newTestTaskListPane(t)
	all := sampleTasks(t)
	tlp.allTasks = append([]app.TaskSummary(nil), all...)
	tlp.tasks = tlp.allTasks

	tlp.refreshCurrentTask("nonexistent", "done", "#00ff00")

	for i, task := range tlp.allTasks {
		assert.Equal(t, all[i].Status, task.Status)
	}
}

func TestRender_HeaderRow(t *testing.T) {
	tlp := newTestTaskListPane(t)
	tlp.tasks = sampleTasks(t)
	tlp.render()

	require.NotNil(t, tlp.GetCell(0, 0))
	assert.Equal(t, "STATUS", tlp.GetCell(0, 0).Text)
	require.NotNil(t, tlp.GetCell(0, 1))
	assert.Equal(t, "TASK NAME", tlp.GetCell(0, 1).Text)
}

func TestRender_TaskRowCount(t *testing.T) {
	tlp := newTestTaskListPane(t)
	tlp.tasks = sampleTasks(t)
	tlp.render()
	assert.NotNil(t, tlp.GetCell(5, 1))
}

func TestRender_SubtaskIndentation(t *testing.T) {
	tlp := newTestTaskListPane(t)
	tlp.tasks = sampleTasks(t)
	tlp.render()

	cell := tlp.GetCell(4, 1)
	require.NotNil(t, cell)
	assert.NotEqual(t, "Update docs", stripTviewTags(cell.Text))
}

func TestRender_EmptyTasks_NoPanic(t *testing.T) {
	tlp := newTestTaskListPane(t)
	tlp.render()
	require.NotNil(t, tlp.GetCell(0, 0))
	assert.Equal(t, "STATUS", tlp.GetCell(0, 0).Text)
}

func TestApplySortToTasks_NoSort(t *testing.T) {
	tlp := newTestTaskListPane(t)
	tasks := sortableTasks(t)
	tlp.tasks = append([]app.TaskSummary(nil), tasks...)

	tlp.applySortToTasks()

	assert.Equal(t, taskIDs(t, tasks), taskIDs(t, tlp.tasks))
}

func TestApplySortToTasks_ByPriority(t *testing.T) {
	tlp := newTestTaskListPane(t)
	tlp.tasks = append([]app.TaskSummary(nil), sortableTasks(t)...)
	tlp.sortField = app.SortFieldPriority
	tlp.sortAsc = true

	tlp.applySortToTasks()

	got := taskIDs(t, tlp.tasks)
	require.NotEmpty(t, got)
	assert.Equal(t, "t3", got[0])
	for i, id := range got {
		if id == "t1" && i+1 < len(got) {
			assert.Equal(t, "t4", got[i+1])
		}
	}
}

func TestApplySortToTasks_NilTasks(t *testing.T) {
	tlp := newTestTaskListPane(t)
	tlp.sortField = app.SortFieldName
	tlp.sortAsc = true
	tlp.applySortToTasks()
	assert.Nil(t, tlp.tasks)
}

func TestSortIndicator(t *testing.T) {
	tests := []struct {
		name      string
		sortField string
		sortAsc   bool
		want      string
	}{
		{name: "no_sort", sortField: "", sortAsc: true, want: ""},
		{name: "ascending_priority", sortField: app.SortFieldPriority, sortAsc: true, want: "↑priority"},
		{name: "descending_name", sortField: app.SortFieldName, sortAsc: false, want: "↓name"},
		{name: "ascending_status", sortField: app.SortFieldStatus, sortAsc: true, want: "↑status"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tlp := newTestTaskListPane(t)
			tlp.sortField = tc.sortField
			tlp.sortAsc = tc.sortAsc
			assert.Equal(t, tc.want, tlp.sortIndicator())
		})
	}
}

func TestCycleSortField_NoApp(t *testing.T) {
	tlp := newTestTaskListPane(t)
	tlp.allTasks = sortableTasks(t)
	tlp.tasks = append([]app.TaskSummary(nil), tlp.allTasks...)

	tlp.cycleSortField()

	assert.Equal(t, app.SortFieldStatus, tlp.sortField)
	assert.True(t, tlp.sortAsc)
}

func TestToggleSortDirection_NoApp(t *testing.T) {
	tlp := newTestTaskListPane(t)
	tlp.allTasks = sortableTasks(t)
	tlp.tasks = append([]app.TaskSummary(nil), tlp.allTasks...)
	tlp.sortField = app.SortFieldPriority
	tlp.sortAsc = true

	tlp.toggleSortDirection()

	assert.False(t, tlp.sortAsc)
}

func TestToggleSortDirection_NoOp_WhenNoSort(t *testing.T) {
	tlp := newTestTaskListPane(t)
	tlp.sortAsc = false

	tlp.toggleSortDirection()

	assert.False(t, tlp.sortAsc)
}

func TestReapplyFilter_WithSort(t *testing.T) {
	tlp := newTestTaskListPane(t)
	tlp.allTasks = append([]app.TaskSummary(nil), sortableTasks(t)...)
	tlp.sortField = app.SortFieldName
	tlp.sortAsc = true

	tlp.reapplyFilter()

	got := taskIDs(t, tlp.tasks)
	require.NotEmpty(t, got)
	assert.Equal(t, "t2", got[0])
}

func TestRestoreSelectionByID_Found(t *testing.T) {
	tlp := newTestTaskListPane(t)
	tlp.tasks = sortableTasks(t)
	tlp.render()

	tlp.restoreSelectionByID("t3")

	row, _ := tlp.GetSelection()
	assert.Equal(t, 3, row)
}

func TestRestoreSelectionByID_NotFound(t *testing.T) {
	tlp := newTestTaskListPane(t)
	tlp.tasks = sortableTasks(t)
	tlp.render()
	tlp.Select(4, 0)

	tlp.restoreSelectionByID("nonexistent")

	row, _ := tlp.GetSelection()
	assert.Equal(t, 1, row)
}

func TestRestoreSelectionByID_Empty(t *testing.T) {
	tlp := newTestTaskListPane(t)
	tlp.tasks = sortableTasks(t)
	tlp.render()
	tlp.Select(3, 0)

	tlp.restoreSelectionByID("")

	row, _ := tlp.GetSelection()
	assert.Equal(t, 3, row)
}

func TestSelectTaskByID_Found(t *testing.T) {
	tlp := newTestTaskListPane(t)
	tlp.tasks = sampleTasks(t)
	tlp.render()

	found := tlp.selectTaskByID("t3")

	assert.True(t, found)
	row, _ := tlp.GetSelection()
	assert.Equal(t, 3, row)
}

func TestSelectTaskByID_NotFound(t *testing.T) {
	tlp := newTestTaskListPane(t)
	tlp.tasks = sampleTasks(t)
	tlp.render()
	assert.False(t, tlp.selectTaskByID("does-not-exist"))
}

func TestSelectTaskByID_EmptyID(t *testing.T) {
	tlp := newTestTaskListPane(t)
	tlp.tasks = sampleTasks(t)
	tlp.render()
	assert.False(t, tlp.selectTaskByID(""))
}

func TestSelectTaskByID_EmptyTaskList(t *testing.T) {
	tlp := newTestTaskListPane(t)
	assert.False(t, tlp.selectTaskByID("t1"))
}

func TestErrTaskNameRequired_Sentinel(t *testing.T) {
	assert.NotNil(t, errTaskNameRequired)
}

func TestErrTaskNameRequired_BlankInputs(t *testing.T) {
	validate := func(s string) error {
		if strings.TrimSpace(s) == "" {
			return errTaskNameRequired
		}
		return nil
	}

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "empty", input: "", wantErr: true},
		{name: "spaces_only", input: "   ", wantErr: true},
		{name: "tab_only", input: "\t", wantErr: true},
		{name: "valid_name", input: "My Task", wantErr: false},
		{name: "trimmed_name", input: "  hello  ", wantErr: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validate(tc.input)
			if tc.wantErr {
				require.ErrorIs(t, err, errTaskNameRequired)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestPriorityOptions_Structure(t *testing.T) {
	require.NotEmpty(t, priorityOptions)
	require.Equal(t, "0", priorityOptions[0].Value)
	require.NotEmpty(t, priorityOptions[0].Label)

	wantValues := []string{"0", "1", "2", "3", "4"}
	require.Len(t, priorityOptions, len(wantValues))
	for i, want := range wantValues {
		assert.Equal(t, want, priorityOptions[i].Value)
		assert.NotEmpty(t, priorityOptions[i].Label)
	}
}
