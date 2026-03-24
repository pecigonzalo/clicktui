// Package tui — unit tests for task list pane state logic.
package tui

import (
	"testing"

	"github.com/rivo/tview"

	"github.com/pecigonzalo/clicktui/internal/app"
)

// ── helper: build a minimal TaskListPane for testing ─────────────────────────

// newTestTaskListPane creates a TaskListPane with just enough wiring for
// state-level tests. It does NOT require a running tview.Application.
func newTestTaskListPane() *TaskListPane {
	table := tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0)
	table.SetBorder(true)

	return &TaskListPane{
		Table: table,
	}
}

// sampleTasks returns a small set of test tasks.
func sampleTasks() []app.TaskSummary {
	return []app.TaskSummary{
		{ID: "t1", Name: "Fix login bug", Status: "in progress", StatusColor: "#1e90ff", Priority: "high", Parent: ""},
		{ID: "t2", Name: "Add dark mode", Status: "todo", StatusColor: "#aaaaaa", Priority: "normal", Parent: ""},
		{ID: "t3", Name: "Write tests", Status: "in progress", StatusColor: "#1e90ff", Priority: "urgent", Parent: ""},
		{ID: "t4", Name: "Update docs", Status: "done", StatusColor: "#00ff00", Priority: "low", Parent: "t1"},
		{ID: "t5", Name: "Deploy to staging", Status: "todo", StatusColor: "#aaaaaa", Priority: "high", Parent: ""},
	}
}

// ── SelectedTaskID ──────────────────────────────────────────────────────────

func TestSelectedTaskID_ValidSelection(t *testing.T) {
	tlp := newTestTaskListPane()
	tlp.tasks = sampleTasks()
	tlp.render()

	// Select the second data row (row 1 is header, row 2 is first task).
	tlp.Select(2, 0)
	got := tlp.SelectedTaskID()
	if got != "t2" {
		t.Errorf("SelectedTaskID() = %q, want 't2'", got)
	}
}

func TestSelectedTaskID_HeaderSelected(t *testing.T) {
	tlp := newTestTaskListPane()
	tlp.tasks = sampleTasks()
	tlp.render()

	// Row 0 is the header — should return empty.
	tlp.Select(0, 0)
	got := tlp.SelectedTaskID()
	if got != "" {
		t.Errorf("SelectedTaskID() at header = %q, want empty", got)
	}
}

func TestSelectedTaskID_EmptyTaskList(t *testing.T) {
	tlp := newTestTaskListPane()
	tlp.tasks = nil
	got := tlp.SelectedTaskID()
	if got != "" {
		t.Errorf("SelectedTaskID() with no tasks = %q, want empty", got)
	}
}

// ── ApplyFilter / ClearFilter state transitions ─────────────────────────────

func TestApplyFilter_SetsFilteredTasks(t *testing.T) {
	tlp := newTestTaskListPane()
	all := sampleTasks()
	tlp.allTasks = all

	filtered := []app.TaskSummary{all[0], all[2]} // only "in progress" tasks
	query := app.TaskQuery{Fields: []app.FieldFilter{{Field: "status", Value: "in progress"}}}

	tlp.ApplyFilter(filtered, query)

	if len(tlp.tasks) != 2 {
		t.Fatalf("ApplyFilter: tasks count = %d, want 2", len(tlp.tasks))
	}
	if tlp.tasks[0].ID != "t1" || tlp.tasks[1].ID != "t3" {
		t.Errorf("ApplyFilter: tasks = %v, want [t1, t3]", taskIDs(tlp.tasks))
	}
	if tlp.activeQuery == nil {
		t.Error("ApplyFilter: activeQuery should be non-nil after applying filter")
	}
}

func TestApplyFilter_EmptyQuery_ShowsAll(t *testing.T) {
	tlp := newTestTaskListPane()
	all := sampleTasks()
	tlp.allTasks = all
	// Pre-set a filter.
	tlp.activeQuery = &app.TaskQuery{FreeText: "something"}
	tlp.tasks = all[:1]

	// Empty query clears the filter.
	tlp.ApplyFilter(nil, app.TaskQuery{})

	if len(tlp.tasks) != len(all) {
		t.Errorf("ApplyFilter(empty): tasks count = %d, want %d", len(tlp.tasks), len(all))
	}
	if tlp.activeQuery != nil {
		t.Error("ApplyFilter(empty): activeQuery should be nil")
	}
}

func TestClearFilter_RestoresAllTasks(t *testing.T) {
	tlp := newTestTaskListPane()
	all := sampleTasks()
	tlp.allTasks = all
	tlp.tasks = all[:2]
	tlp.activeQuery = &app.TaskQuery{FreeText: "fix"}

	tlp.ClearFilter()

	if len(tlp.tasks) != len(all) {
		t.Errorf("ClearFilter: tasks count = %d, want %d", len(tlp.tasks), len(all))
	}
	if tlp.activeQuery != nil {
		t.Error("ClearFilter: activeQuery should be nil")
	}
}

func TestApplyFilter_NilFiltered_SetsEmptySlice(t *testing.T) {
	tlp := newTestTaskListPane()
	tlp.allTasks = sampleTasks()

	query := app.TaskQuery{FreeText: "zzz-no-match"}
	tlp.ApplyFilter(nil, query)

	if tlp.tasks != nil {
		t.Errorf("ApplyFilter(nil, non-empty query): tasks should be nil, got %d items", len(tlp.tasks))
	}
	if tlp.activeQuery == nil {
		t.Error("ApplyFilter: activeQuery should be non-nil")
	}
}

// ── refreshCurrentTask state updates ────────────────────────────────────────

func TestRefreshCurrentTask_UpdatesAllTasks(t *testing.T) {
	tlp := newTestTaskListPane()
	all := sampleTasks()
	// Copy so we don't mutate the original.
	allCopy := make([]app.TaskSummary, len(all))
	copy(allCopy, all)
	tlp.allTasks = allCopy
	tlp.tasks = allCopy

	tlp.refreshCurrentTask("t2", "done", "#00ff00")

	// Verify allTasks was updated.
	for _, task := range tlp.allTasks {
		if task.ID == "t2" {
			if task.Status != "done" {
				t.Errorf("allTasks[t2].Status = %q, want 'done'", task.Status)
			}
			if task.StatusColor != "#00ff00" {
				t.Errorf("allTasks[t2].StatusColor = %q, want '#00ff00'", task.StatusColor)
			}
			return
		}
	}
	t.Error("task t2 not found in allTasks")
}

func TestRefreshCurrentTask_UpdatesDisplayedTasks(t *testing.T) {
	tlp := newTestTaskListPane()
	all := sampleTasks()
	allCopy := make([]app.TaskSummary, len(all))
	copy(allCopy, all)
	tlp.allTasks = allCopy
	tlp.tasks = allCopy
	// No active filter — should update displayed tasks directly.
	tlp.activeQuery = nil

	tlp.refreshCurrentTask("t1", "review", "#ff6600")

	found := false
	for _, task := range tlp.tasks {
		if task.ID == "t1" {
			found = true
			if task.Status != "review" {
				t.Errorf("tasks[t1].Status = %q, want 'review'", task.Status)
			}
			break
		}
	}
	if !found {
		t.Error("task t1 not found in displayed tasks")
	}
}

func TestRefreshCurrentTask_NonexistentTask(t *testing.T) {
	tlp := newTestTaskListPane()
	all := sampleTasks()
	tlp.allTasks = make([]app.TaskSummary, len(all))
	copy(tlp.allTasks, all)
	tlp.tasks = tlp.allTasks

	// Should not panic for a task ID that doesn't exist.
	tlp.refreshCurrentTask("nonexistent", "done", "#00ff00")

	// Verify nothing changed.
	for i, task := range tlp.allTasks {
		if task.Status != all[i].Status {
			t.Errorf("allTasks[%d].Status changed from %q to %q; should be unchanged",
				i, all[i].Status, task.Status)
		}
	}
}

// ── render ───────────────────────────────────────────────────────────────────

func TestRender_HeaderRow(t *testing.T) {
	tlp := newTestTaskListPane()
	tlp.tasks = sampleTasks()
	tlp.render()

	// Row 0 should be the header.
	cell := tlp.GetCell(0, 0)
	if cell == nil {
		t.Fatal("render: header cell (0,0) is nil")
	}
	text := cell.Text
	if text != "STATUS" {
		t.Errorf("render: header cell (0,0) = %q, want 'STATUS'", text)
	}

	nameCell := tlp.GetCell(0, 1)
	if nameCell == nil || nameCell.Text != "TASK NAME" {
		t.Errorf("render: header cell (0,1) = %v, want 'TASK NAME'", nameCell)
	}
}

func TestRender_TaskRowCount(t *testing.T) {
	tlp := newTestTaskListPane()
	tlp.tasks = sampleTasks()
	tlp.render()

	// Should have 1 header + 5 task rows = row indices 0..5.
	// Check that the last task row exists.
	cell := tlp.GetCell(5, 1) // 5th task (0-indexed row 5)
	if cell == nil {
		t.Error("render: expected 5 task rows (row 5 should exist)")
	}
}

func TestRender_SubtaskIndentation(t *testing.T) {
	tlp := newTestTaskListPane()
	tlp.tasks = sampleTasks() // t4 has Parent="t1"
	tlp.render()

	// t4 is at index 3 (4th task), so row 4 (row 0 is header).
	cell := tlp.GetCell(4, 1) // name column for t4
	if cell == nil {
		t.Fatal("render: subtask cell is nil")
	}
	text := stripTviewTags(cell.Text)
	// Subtask should be indented with the SubtaskPrefix icon.
	if text == "Update docs" {
		t.Error("render: subtask should have indentation/prefix, got bare name")
	}
}

func TestRender_EmptyTasks_NoPanic(t *testing.T) {
	tlp := newTestTaskListPane()
	tlp.tasks = nil
	// Rendering an empty task list should not panic.
	tlp.render()

	// Header row should still exist.
	cell := tlp.GetCell(0, 0)
	if cell == nil {
		t.Fatal("render with empty tasks: header cell should still exist")
	}
	if cell.Text != "STATUS" {
		t.Errorf("render with empty tasks: header = %q, want 'STATUS'", cell.Text)
	}
}

// ── helper ──────────────────────────────────────────────────────────────────

func taskIDs(tasks []app.TaskSummary) []string {
	ids := make([]string, len(tasks))
	for i, t := range tasks {
		ids[i] = t.ID
	}
	return ids
}
