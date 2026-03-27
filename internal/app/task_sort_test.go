package app_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pecigonzalo/clicktui/internal/app"
)

// ── helpers ──────────────────────────────────────────────────────────────────

func ids(tasks []app.TaskSummary) []string {
	out := make([]string, len(tasks))
	for i, t := range tasks {
		out[i] = t.ID
	}
	return out
}

func makeTasks() []app.TaskSummary {
	return []app.TaskSummary{
		{ID: "t1", Name: "Beta task", Status: "in progress", Priority: "high", PriorityOrder: 2, DueDate: "2025-03-20", Assignee: "carol"},
		{ID: "t2", Name: "Alpha task", Status: "todo", Priority: "urgent", PriorityOrder: 1, DueDate: "2025-03-15", Assignee: "alice"},
		{ID: "t3", Name: "Gamma task", Status: "done", Priority: "low", PriorityOrder: 4, DueDate: "", Assignee: ""},
		{ID: "t4", Name: "Delta task", Status: "todo", Priority: "normal", PriorityOrder: 3, DueDate: "2025-04-01", Assignee: "bob"},
		{ID: "t5", Name: "Epsilon task", Status: "in progress", Priority: "none", PriorityOrder: 5, DueDate: "", Assignee: ""},
	}
}

// ── SortTasks field sorting ─────────────────────────────────────────────────

func TestSortTasks_ByName(t *testing.T) {
	cases := []struct {
		name string
		asc  bool
		want []string
	}{
		{"ascending", true, []string{"t2", "t1", "t4", "t5", "t3"}},
		{"descending", false, []string{"t3", "t5", "t4", "t1", "t2"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := app.SortTasks(makeTasks(), "name", tc.asc, nil)
			assert.Equal(t, tc.want, ids(result))
		})
	}
}

func TestSortTasks_ByPriority(t *testing.T) {
	cases := []struct {
		name string
		asc  bool
		want []string
	}{
		// urgent(1) < high(2) < normal(3) < low(4) < none(5=empty,last)
		{"ascending", true, []string{"t2", "t1", "t4", "t3", "t5"}},
		{"descending", false, []string{"t3", "t4", "t1", "t2", "t5"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := app.SortTasks(makeTasks(), "priority", tc.asc, nil)
			assert.Equal(t, tc.want, ids(result))
		})
	}
}

func TestSortTasks_ByDueDate(t *testing.T) {
	cases := []struct {
		name string
		asc  bool
		want []string
	}{
		// "2025-03-15" < "2025-03-20" < "2025-04-01" < empty(last)
		{"ascending", true, []string{"t2", "t1", "t4", "t3", "t5"}},
		{"descending", false, []string{"t4", "t1", "t2", "t3", "t5"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := app.SortTasks(makeTasks(), "due_date", tc.asc, nil)
			assert.Equal(t, tc.want, ids(result))
		})
	}
}

func TestSortTasks_ByAssignee(t *testing.T) {
	cases := []struct {
		name string
		asc  bool
		want []string
	}{
		// alice < bob < carol < empty(last)
		{"ascending", true, []string{"t2", "t4", "t1", "t3", "t5"}},
		{"descending", false, []string{"t1", "t4", "t2", "t3", "t5"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := app.SortTasks(makeTasks(), "assignee", tc.asc, nil)
			assert.Equal(t, tc.want, ids(result))
		})
	}
}

func TestSortTasks_ByStatus_WithOrder(t *testing.T) {
	statusOrder := map[string]int{
		"todo":        0,
		"in progress": 1,
		"done":        2,
	}
	cases := []struct {
		name string
		asc  bool
		want []string
	}{
		// todo(0) < in progress(1) < done(2)
		{"ascending", true, []string{"t2", "t4", "t1", "t5", "t3"}},
		{"descending", false, []string{"t3", "t1", "t5", "t2", "t4"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := app.SortTasks(makeTasks(), "status", tc.asc, statusOrder)
			assert.Equal(t, tc.want, ids(result))
		})
	}
}

func TestSortTasks_ByStatus_AlphaFallback(t *testing.T) {
	// No statusOrder → alphabetical fallback.
	result := app.SortTasks(makeTasks(), "status", true, nil)
	// "done" < "in progress" < "todo"
	assert.Equal(t, []string{"t3", "t1", "t5", "t2", "t4"}, ids(result))
}

// ── Subtask adjacency preservation ──────────────────────────────────────────

func TestSortTasks_PreservesSubtaskAdjacency(t *testing.T) {
	tasks := []app.TaskSummary{
		{ID: "p1", Name: "Zebra", Status: "todo", PriorityOrder: 3},
		{ID: "c1", Name: "Child of Zebra", Status: "todo", PriorityOrder: 1, Parent: "p1"},
		{ID: "c2", Name: "Another child", Status: "done", PriorityOrder: 4, Parent: "p1"},
		{ID: "p2", Name: "Alpha", Status: "done", PriorityOrder: 2},
	}

	result := app.SortTasks(tasks, "name", true, nil)

	// Groups sorted by parent name: Alpha(p2) < Zebra(p1).
	// Children stay under their parent in original order.
	require.Len(t, result, 4)
	assert.Equal(t, "p2", result[0].ID, "Alpha parent first")
	assert.Equal(t, "p1", result[1].ID, "Zebra parent second")
	assert.Equal(t, "c1", result[2].ID, "first child of Zebra")
	assert.Equal(t, "c2", result[3].ID, "second child of Zebra")
}

func TestSortTasks_OrphanSubtasksSortAsTopLevel(t *testing.T) {
	tasks := []app.TaskSummary{
		{ID: "t1", Name: "Gamma", PriorityOrder: 5},
		{ID: "orphan", Name: "Alpha orphan", Parent: "missing", PriorityOrder: 5},
		{ID: "t2", Name: "Beta", PriorityOrder: 5},
	}

	result := app.SortTasks(tasks, "name", true, nil)

	// orphan parent is missing -> treated as top-level.
	assert.Equal(t, []string{"orphan", "t2", "t1"}, ids(result))
}

func TestSortTasks_MultipleParentsWithChildren(t *testing.T) {
	tasks := []app.TaskSummary{
		{ID: "p1", Name: "Zebra", PriorityOrder: 5},
		{ID: "c1a", Name: "Z-child-a", Parent: "p1", PriorityOrder: 5},
		{ID: "c1b", Name: "Z-child-b", Parent: "p1", PriorityOrder: 5},
		{ID: "p2", Name: "Alpha", PriorityOrder: 5},
		{ID: "c2a", Name: "A-child-a", Parent: "p2", PriorityOrder: 5},
	}

	result := app.SortTasks(tasks, "name", true, nil)

	assert.Equal(t, []string{"p2", "c2a", "p1", "c1a", "c1b"}, ids(result))
}

// ── Edge cases ──────────────────────────────────────────────────────────────

func TestSortTasks_Empty(t *testing.T) {
	result := app.SortTasks(nil, "name", true, nil)
	assert.Empty(t, result)
}

func TestSortTasks_SingleTask(t *testing.T) {
	tasks := []app.TaskSummary{{ID: "t1", Name: "Solo"}}
	result := app.SortTasks(tasks, "name", true, nil)
	require.Len(t, result, 1)
	assert.Equal(t, "t1", result[0].ID)
}

func TestSortTasks_EmptyField_NoSort(t *testing.T) {
	tasks := makeTasks()
	original := ids(tasks)
	result := app.SortTasks(tasks, "", true, nil)
	assert.Equal(t, original, ids(result), "empty field should preserve order")
}

func TestSortTasks_UnknownField_NoSort(t *testing.T) {
	tasks := makeTasks()
	original := ids(tasks)
	result := app.SortTasks(tasks, "nonexistent", true, nil)
	// Unknown field -> all keys are zero -> stable sort preserves order.
	assert.Equal(t, original, ids(result))
}

func TestSortTasks_DoesNotMutateInput(t *testing.T) {
	tasks := makeTasks()
	original := make([]app.TaskSummary, len(tasks))
	copy(original, tasks)

	_ = app.SortTasks(tasks, "name", true, nil)

	assert.Equal(t, ids(original), ids(tasks), "input slice must not be modified")
}

func TestSortTasks_EmptyValuesSortLast_Ascending(t *testing.T) {
	tasks := []app.TaskSummary{
		{ID: "t1", Name: "A", DueDate: ""},
		{ID: "t2", Name: "B", DueDate: "2025-01-01"},
		{ID: "t3", Name: "C", DueDate: ""},
		{ID: "t4", Name: "D", DueDate: "2025-06-01"},
	}

	result := app.SortTasks(tasks, "due_date", true, nil)

	// Non-empty sorted first, empty last.
	assert.Equal(t, []string{"t2", "t4", "t1", "t3"}, ids(result))
}

func TestSortTasks_EmptyValuesSortLast_Descending(t *testing.T) {
	tasks := []app.TaskSummary{
		{ID: "t1", Name: "A", DueDate: ""},
		{ID: "t2", Name: "B", DueDate: "2025-01-01"},
		{ID: "t3", Name: "C", DueDate: ""},
		{ID: "t4", Name: "D", DueDate: "2025-06-01"},
	}

	result := app.SortTasks(tasks, "due_date", false, nil)

	// Non-empty sorted descending, empty still last.
	assert.Equal(t, []string{"t4", "t2", "t1", "t3"}, ids(result))
}

// ── NextSortField ───────────────────────────────────────────────────────────

func TestNextSortField(t *testing.T) {
	cases := []struct {
		current string
		want    string
	}{
		{"", "status"},
		{"status", "priority"},
		{"priority", "due_date"},
		{"due_date", "assignee"},
		{"assignee", "name"},
		{"name", ""},
		{"unknown", "status"},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%q to %q", tc.current, tc.want), func(t *testing.T) {
			t.Parallel()
			got := app.NextSortField(tc.current)
			assert.Equal(t, tc.want, got)
		})
	}
}
