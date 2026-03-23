package app_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pecigonzalo/clicktui/internal/app"
	"github.com/pecigonzalo/clicktui/internal/clickup"
)

func TestTaskService_LoadTasks(t *testing.T) {
	api := newFakeAPI()
	api.tasks["l1"] = []clickup.Task{
		{
			ID:   "t1",
			Name: "Fix login",
			Status: clickup.Status{
				Status: "in progress",
			},
			Priority: &clickup.Priority{Name: "high"},
		},
		{
			ID:   "t2",
			Name: "Update docs",
			Status: clickup.Status{
				Status: "open",
			},
		},
	}

	svc := app.NewTaskService(api)
	summaries, err := svc.LoadTasks(context.Background(), "l1", 0)
	require.NoError(t, err)
	require.Len(t, summaries, 2)

	assert.Equal(t, "t1", summaries[0].ID)
	assert.Equal(t, "Fix login", summaries[0].Name)
	assert.Equal(t, "in progress", summaries[0].Status)
	assert.Equal(t, "high", summaries[0].Priority)
	assert.Empty(t, summaries[0].Parent, "top-level task should have empty Parent")

	assert.Equal(t, "none", summaries[1].Priority)
}

func TestTaskService_LoadTasks_Error(t *testing.T) {
	api := newFakeAPI()
	api.tasksErr = errors.New("api failure")

	svc := app.NewTaskService(api)
	_, err := svc.LoadTasks(context.Background(), "l1", 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "load tasks")
}

func TestTaskService_LoadTaskDetail(t *testing.T) {
	api := newFakeAPI()
	api.tasksByID["t1"] = &clickup.Task{
		ID:   "t1",
		Name: "Fix login",
		Status: clickup.Status{
			Status: "in progress",
			Color:  "#ff0000",
		},
		Priority: &clickup.Priority{Name: "high"},
		Assignees: []clickup.Assignee{
			{ID: 1, Username: "alice"},
		},
		Tags: []clickup.Tag{
			{Name: "bug"},
		},
		DueDate:     "1700000000000",
		DateCreated: "1699900000000",
		URL:         "https://app.clickup.com/t/t1",
		List:        clickup.TaskRef{ID: "l1", Name: "Sprint 42"},
		Folder:      clickup.TaskRef{ID: "f1", Name: "Backend"},
		Space:       clickup.TaskRef{ID: "s1", Name: "Engineering"},
	}

	svc := app.NewTaskService(api)
	detail, err := svc.LoadTaskDetail(context.Background(), "t1")
	require.NoError(t, err)

	assert.Equal(t, "t1", detail.ID)
	assert.Equal(t, "Fix login", detail.Name)
	assert.Equal(t, "in progress", detail.Status)
	assert.Equal(t, "#ff0000", detail.StatusColor)
	assert.Equal(t, "high", detail.Priority)
	assert.Equal(t, []string{"alice"}, detail.Assignees)
	assert.Equal(t, []string{"bug"}, detail.Tags)
	assert.Equal(t, "2023-11-14", detail.DueDate)
	assert.Equal(t, "Sprint 42", detail.List)
	assert.Equal(t, "l1", detail.ListID)
	assert.Equal(t, "Backend", detail.Folder)
	assert.Equal(t, "Engineering", detail.Space)
	assert.Equal(t, "https://app.clickup.com/t/t1", detail.URL)
}

func TestTaskService_LoadTaskDetail_Error(t *testing.T) {
	api := newFakeAPI()
	api.taskErr = errors.New("not found")

	svc := app.NewTaskService(api)
	_, err := svc.LoadTaskDetail(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "load task detail")
}

func TestTaskService_LoadTasks_NilPriority(t *testing.T) {
	api := newFakeAPI()
	api.tasks["l1"] = []clickup.Task{
		{
			ID:   "t1",
			Name: "No priority",
			Status: clickup.Status{
				Status: "open",
			},
			// Priority is nil.
		},
	}

	svc := app.NewTaskService(api)
	summaries, err := svc.LoadTasks(context.Background(), "l1", 0)
	require.NoError(t, err)
	require.Len(t, summaries, 1)
	assert.Equal(t, "none", summaries[0].Priority)
}

func TestTaskService_LoadTaskDetail_EmptyDates(t *testing.T) {
	api := newFakeAPI()
	api.tasksByID["t1"] = &clickup.Task{
		ID:   "t1",
		Name: "No dates",
		Status: clickup.Status{
			Status: "open",
		},
		List:   clickup.TaskRef{ID: "l1"},
		Folder: clickup.TaskRef{ID: "f1"},
		Space:  clickup.TaskRef{ID: "s1"},
	}

	svc := app.NewTaskService(api)
	detail, err := svc.LoadTaskDetail(context.Background(), "t1")
	require.NoError(t, err)
	assert.Empty(t, detail.DueDate)
	assert.Empty(t, detail.StartDate)
}

func TestTaskService_LoadListStatuses(t *testing.T) {
	api := newFakeAPI()
	api.statusesByListID["l1"] = []clickup.Status{
		{Status: "open", Color: "#d3d3d3", Type: "open"},
		{Status: "in progress", Color: "#4169e1", Type: "custom"},
		{Status: "done", Color: "#00ff00", Type: "closed"},
	}

	svc := app.NewTaskService(api)
	opts, err := svc.LoadListStatuses(context.Background(), "l1")
	require.NoError(t, err)
	require.Len(t, opts, 3)

	assert.Equal(t, "open", opts[0].Name)
	assert.Equal(t, "#d3d3d3", opts[0].Color)
	assert.Equal(t, "open", opts[0].Type)
	assert.Equal(t, "in progress", opts[1].Name)
	assert.Equal(t, "done", opts[2].Name)
}

func TestTaskService_LoadListStatuses_Empty(t *testing.T) {
	api := newFakeAPI()
	// No statuses configured for the list — returns empty slice, not an error.

	svc := app.NewTaskService(api)
	opts, err := svc.LoadListStatuses(context.Background(), "l1")
	require.NoError(t, err)
	assert.Empty(t, opts)
}

func TestTaskService_LoadListStatuses_Error(t *testing.T) {
	api := newFakeAPI()
	api.listStatusesErr = errors.New("list not found")

	svc := app.NewTaskService(api)
	_, err := svc.LoadListStatuses(context.Background(), "l1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "load list statuses")
}

func TestTaskService_UpdateTaskStatus(t *testing.T) {
	api := newFakeAPI()
	api.tasksByID["t1"] = &clickup.Task{
		ID:   "t1",
		Name: "Fix login",
		Status: clickup.Status{
			Status: "open",
			Color:  "#d3d3d3",
		},
		List: clickup.TaskRef{ID: "l1", Name: "Sprint 42"},
	}

	svc := app.NewTaskService(api)
	detail, err := svc.UpdateTaskStatus(context.Background(), "t1", "in progress")
	require.NoError(t, err)
	require.NotNil(t, detail)

	assert.Equal(t, "t1", detail.ID)
	assert.Equal(t, "in progress", detail.Status)
}

func TestTaskService_UpdateTaskStatus_Error(t *testing.T) {
	api := newFakeAPI()
	api.updateStatusErr = &clickup.APIError{StatusCode: 400, Body: `{"err":"Invalid status"}`}

	svc := app.NewTaskService(api)
	_, err := svc.UpdateTaskStatus(context.Background(), "t1", "invalid_status")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "update task status")
}

func TestTaskService_UpdateTaskStatus_Unauthorized(t *testing.T) {
	api := newFakeAPI()
	api.updateStatusErr = &clickup.APIError{StatusCode: 401, Body: `{"err":"Token invalid"}`}

	svc := app.NewTaskService(api)
	_, err := svc.UpdateTaskStatus(context.Background(), "t1", "done")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "update task status")

	var apiErr *clickup.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 401, apiErr.StatusCode)
}

func TestTaskService_UpdateTaskStatus_RateLimit(t *testing.T) {
	api := newFakeAPI()
	api.updateStatusErr = &clickup.APIError{StatusCode: 429, Body: `{"err":"Rate limit exceeded"}`}

	svc := app.NewTaskService(api)
	_, err := svc.UpdateTaskStatus(context.Background(), "t1", "done")
	require.Error(t, err)

	var apiErr *clickup.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 429, apiErr.StatusCode)
}

func TestTaskService_LoadTaskDetail_LongDescription(t *testing.T) {
	// Verify that descriptions longer than 500 chars are truncated.
	long := make([]byte, 600)
	for i := range long {
		long[i] = 'x'
	}
	api := newFakeAPI()
	api.tasksByID["t1"] = &clickup.Task{
		ID:          "t1",
		Name:        "Long desc",
		Status:      clickup.Status{Status: "open"},
		Description: string(long),
		List:        clickup.TaskRef{ID: "l1"},
	}

	svc := app.NewTaskService(api)
	detail, err := svc.LoadTaskDetail(context.Background(), "t1")
	require.NoError(t, err)
	assert.LessOrEqual(t, len(detail.Description), 503) // 500 + "..."
	assert.True(t, len(detail.Description) > 0)
}

func TestTaskService_LoadTaskDetail_InvalidDateFormat(t *testing.T) {
	// A non-numeric date string should be returned as-is rather than failing.
	api := newFakeAPI()
	api.tasksByID["t1"] = &clickup.Task{
		ID:          "t1",
		Name:        "Bad date",
		Status:      clickup.Status{Status: "open"},
		DueDate:     "not-a-timestamp",
		DateCreated: "null",
		List:        clickup.TaskRef{ID: "l1"},
	}

	svc := app.NewTaskService(api)
	detail, err := svc.LoadTaskDetail(context.Background(), "t1")
	require.NoError(t, err)
	// Invalid epoch strings are returned verbatim.
	assert.Equal(t, "not-a-timestamp", detail.DueDate)
	// "null" should produce an empty string.
	assert.Empty(t, detail.DateCreated)
}

func TestTaskService_LoadTasks_SetsParentField(t *testing.T) {
	api := newFakeAPI()
	api.tasks["l1"] = []clickup.Task{
		{
			ID:     "parent1",
			Name:   "Parent task",
			Status: clickup.Status{Status: "open"},
		},
		{
			ID:     "child1",
			Name:   "Child task",
			Status: clickup.Status{Status: "open"},
			Parent: "parent1",
		},
	}

	svc := app.NewTaskService(api)
	summaries, err := svc.LoadTasks(context.Background(), "l1", 0)
	require.NoError(t, err)
	require.Len(t, summaries, 2)

	// Parent appears first, child immediately after.
	assert.Equal(t, "parent1", summaries[0].ID)
	assert.Empty(t, summaries[0].Parent)

	assert.Equal(t, "child1", summaries[1].ID)
	assert.Equal(t, "parent1", summaries[1].Parent)
}

func TestTaskService_LoadTaskDetail_WithSubtasks(t *testing.T) {
	api := newFakeAPI()
	api.tasksByID["t1"] = &clickup.Task{
		ID:     "t1",
		Name:   "Parent task",
		Status: clickup.Status{Status: "in progress"},
		List:   clickup.TaskRef{ID: "l1"},
		Subtasks: []clickup.Task{
			{
				ID:     "st1",
				Name:   "Subtask one",
				Status: clickup.Status{Status: "open"},
			},
			{
				ID:     "st2",
				Name:   "Subtask two",
				Status: clickup.Status{Status: "done"},
			},
		},
	}

	svc := app.NewTaskService(api)
	detail, err := svc.LoadTaskDetail(context.Background(), "t1")
	require.NoError(t, err)
	require.Len(t, detail.Subtasks, 2)

	assert.Equal(t, "st1", detail.Subtasks[0].ID)
	assert.Equal(t, "Subtask one", detail.Subtasks[0].Name)
	assert.Equal(t, "open", detail.Subtasks[0].Status)

	assert.Equal(t, "st2", detail.Subtasks[1].ID)
	assert.Equal(t, "Subtask two", detail.Subtasks[1].Name)
	assert.Equal(t, "done", detail.Subtasks[1].Status)
}

func TestTaskService_LoadTaskDetail_NoSubtasks(t *testing.T) {
	api := newFakeAPI()
	api.tasksByID["t1"] = &clickup.Task{
		ID:     "t1",
		Name:   "Leaf task",
		Status: clickup.Status{Status: "open"},
		List:   clickup.TaskRef{ID: "l1"},
	}

	svc := app.NewTaskService(api)
	detail, err := svc.LoadTaskDetail(context.Background(), "t1")
	require.NoError(t, err)
	assert.Empty(t, detail.Subtasks)
}

func TestTaskService_UpdateTaskStatus_EmptySubtasks(t *testing.T) {
	// Status update API responses typically don't include nested subtasks.
	// Verify the detail is returned with an empty Subtasks slice.
	api := newFakeAPI()
	api.tasksByID["t1"] = &clickup.Task{
		ID:     "t1",
		Name:   "Fix login",
		Status: clickup.Status{Status: "open"},
		List:   clickup.TaskRef{ID: "l1", Name: "Sprint 42"},
		// No Subtasks field set — simulates the update API response.
	}

	svc := app.NewTaskService(api)
	detail, err := svc.UpdateTaskStatus(context.Background(), "t1", "in progress")
	require.NoError(t, err)
	require.NotNil(t, detail)

	assert.Equal(t, "in progress", detail.Status)
	assert.Empty(t, detail.Subtasks)
}

func TestOrderByParent_ParentFollowedByChildren(t *testing.T) {
	input := []app.TaskSummary{
		{ID: "p1", Name: "Parent"},
		{ID: "c1", Name: "Child 1", Parent: "p1"},
		{ID: "c2", Name: "Child 2", Parent: "p1"},
	}

	// Run through LoadTasks by setting up the fake API.
	api := newFakeAPI()
	api.tasks["l1"] = []clickup.Task{
		{ID: "p1", Name: "Parent", Status: clickup.Status{Status: "open"}},
		{ID: "c1", Name: "Child 1", Status: clickup.Status{Status: "open"}, Parent: "p1"},
		{ID: "c2", Name: "Child 2", Status: clickup.Status{Status: "open"}, Parent: "p1"},
	}

	svc := app.NewTaskService(api)
	result, err := svc.LoadTasks(context.Background(), "l1", 0)
	require.NoError(t, err)
	require.Len(t, result, len(input))

	assert.Equal(t, "p1", result[0].ID)
	assert.Equal(t, "c1", result[1].ID)
	assert.Equal(t, "c2", result[2].ID)
}

func TestOrderByParent_MultipleParentsContiguous(t *testing.T) {
	// API returns: P1, P2, C1-of-P1, C2-of-P2
	// Expected:    P1, C1-of-P1, P2, C2-of-P2
	api := newFakeAPI()
	api.tasks["l1"] = []clickup.Task{
		{ID: "p1", Name: "Parent 1", Status: clickup.Status{Status: "open"}},
		{ID: "p2", Name: "Parent 2", Status: clickup.Status{Status: "open"}},
		{ID: "c1", Name: "Child of P1", Status: clickup.Status{Status: "open"}, Parent: "p1"},
		{ID: "c2", Name: "Child of P2", Status: clickup.Status{Status: "open"}, Parent: "p2"},
	}

	svc := app.NewTaskService(api)
	result, err := svc.LoadTasks(context.Background(), "l1", 0)
	require.NoError(t, err)
	require.Len(t, result, 4)

	assert.Equal(t, "p1", result[0].ID)
	assert.Equal(t, "c1", result[1].ID)
	assert.Equal(t, "p2", result[2].ID)
	assert.Equal(t, "c2", result[3].ID)
}

func TestOrderByParent_OrphanSubtaskTreatedAsTopLevel(t *testing.T) {
	// Subtask whose parent is not in the list — should stay in position.
	api := newFakeAPI()
	api.tasks["l1"] = []clickup.Task{
		{ID: "t1", Name: "Normal task", Status: clickup.Status{Status: "open"}},
		{ID: "orphan", Name: "Orphan subtask", Status: clickup.Status{Status: "open"}, Parent: "missing_parent"},
		{ID: "t2", Name: "Another task", Status: clickup.Status{Status: "open"}},
	}

	svc := app.NewTaskService(api)
	result, err := svc.LoadTasks(context.Background(), "l1", 0)
	require.NoError(t, err)
	require.Len(t, result, 3)

	assert.Equal(t, "t1", result[0].ID)
	assert.Equal(t, "orphan", result[1].ID)
	assert.Equal(t, "missing_parent", result[1].Parent, "orphan keeps its Parent value")
	assert.Equal(t, "t2", result[2].ID)
}

func TestOrderByParent_AllTopLevel(t *testing.T) {
	// No subtasks — order should be preserved exactly.
	api := newFakeAPI()
	api.tasks["l1"] = []clickup.Task{
		{ID: "a", Name: "A", Status: clickup.Status{Status: "open"}},
		{ID: "b", Name: "B", Status: clickup.Status{Status: "open"}},
		{ID: "c", Name: "C", Status: clickup.Status{Status: "open"}},
	}

	svc := app.NewTaskService(api)
	result, err := svc.LoadTasks(context.Background(), "l1", 0)
	require.NoError(t, err)
	require.Len(t, result, 3)

	assert.Equal(t, "a", result[0].ID)
	assert.Equal(t, "b", result[1].ID)
	assert.Equal(t, "c", result[2].ID)
}

func TestOrderByParent_EmptyInput(t *testing.T) {
	api := newFakeAPI()
	api.tasks["l1"] = []clickup.Task{}

	svc := app.NewTaskService(api)
	result, err := svc.LoadTasks(context.Background(), "l1", 0)
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestOrderByParent_OnlySubtasks(t *testing.T) {
	// All tasks have parents not present in the list — all treated as top-level.
	api := newFakeAPI()
	api.tasks["l1"] = []clickup.Task{
		{ID: "s1", Name: "Sub 1", Status: clickup.Status{Status: "open"}, Parent: "x"},
		{ID: "s2", Name: "Sub 2", Status: clickup.Status{Status: "open"}, Parent: "y"},
		{ID: "s3", Name: "Sub 3", Status: clickup.Status{Status: "open"}, Parent: "z"},
	}

	svc := app.NewTaskService(api)
	result, err := svc.LoadTasks(context.Background(), "l1", 0)
	require.NoError(t, err)
	require.Len(t, result, 3)

	// Order preserved since all are orphan subtasks (treated as top-level).
	assert.Equal(t, "s1", result[0].ID)
	assert.Equal(t, "s2", result[1].ID)
	assert.Equal(t, "s3", result[2].ID)
}

// countingAPI wraps a fakeAPI and counts calls to the methods under test.
// Counters use atomic ints so tests can safely read them after concurrent use.
type countingAPI struct {
	*fakeAPI
	taskCalls         atomic.Int64
	tasksCalls        atomic.Int64
	listStatusesCalls atomic.Int64
	updateStatusCalls atomic.Int64
}

func newCountingAPI() *countingAPI {
	return &countingAPI{fakeAPI: newFakeAPI()}
}

func (c *countingAPI) Task(ctx context.Context, taskID string) (*clickup.Task, error) {
	c.taskCalls.Add(1)
	return c.fakeAPI.Task(ctx, taskID)
}

func (c *countingAPI) Tasks(ctx context.Context, listID string, page int) ([]clickup.Task, error) {
	c.tasksCalls.Add(1)
	return c.fakeAPI.Tasks(ctx, listID, page)
}

func (c *countingAPI) ListStatuses(ctx context.Context, listID string) ([]clickup.Status, error) {
	c.listStatusesCalls.Add(1)
	return c.fakeAPI.ListStatuses(ctx, listID)
}

func (c *countingAPI) UpdateTaskStatus(ctx context.Context, taskID, status string) (*clickup.Task, error) {
	c.updateStatusCalls.Add(1)
	return c.fakeAPI.UpdateTaskStatus(ctx, taskID, status)
}

// --- Cache behavior tests ---

func TestTaskService_LoadTaskDetail_CachesResult(t *testing.T) {
	api := newCountingAPI()
	api.tasksByID["t1"] = &clickup.Task{
		ID:     "t1",
		Name:   "Cached task",
		Status: clickup.Status{Status: "open"},
		List:   clickup.TaskRef{ID: "l1"},
	}

	svc := app.NewTaskService(api)
	ctx := context.Background()

	// First call — cache miss, should call API.
	d1, err := svc.LoadTaskDetail(ctx, "t1")
	require.NoError(t, err)
	assert.Equal(t, "t1", d1.ID)
	assert.Equal(t, int64(1), api.taskCalls.Load())

	// Second call — cache hit, API must not be called again.
	d2, err := svc.LoadTaskDetail(ctx, "t1")
	require.NoError(t, err)
	assert.Equal(t, d1, d2)
	assert.Equal(t, int64(1), api.taskCalls.Load())
}

func TestTaskService_LoadTasks_CachesResult(t *testing.T) {
	api := newCountingAPI()
	api.tasks["l1"] = []clickup.Task{
		{ID: "t1", Name: "Task 1", Status: clickup.Status{Status: "open"}},
	}

	svc := app.NewTaskService(api)
	ctx := context.Background()

	// First call — cache miss.
	s1, err := svc.LoadTasks(ctx, "l1", 0)
	require.NoError(t, err)
	require.Len(t, s1, 1)
	assert.Equal(t, int64(1), api.tasksCalls.Load())

	// Second call with the same page — cache hit, no new API call.
	s2, err := svc.LoadTasks(ctx, "l1", 0)
	require.NoError(t, err)
	assert.Equal(t, s1, s2)
	assert.Equal(t, int64(1), api.tasksCalls.Load())

	// Call with a different page — cache miss, new API call.
	_, err = svc.LoadTasks(ctx, "l1", 1)
	require.NoError(t, err)
	assert.Equal(t, int64(2), api.tasksCalls.Load(), "different page should not hit cache")
}

func TestTaskService_LoadListStatuses_CachesResult(t *testing.T) {
	api := newCountingAPI()
	api.statusesByListID["l1"] = []clickup.Status{
		{Status: "open", Color: "#ccc", Type: "open"},
	}

	svc := app.NewTaskService(api)
	ctx := context.Background()

	// First call — cache miss.
	o1, err := svc.LoadListStatuses(ctx, "l1")
	require.NoError(t, err)
	require.Len(t, o1, 1)
	assert.Equal(t, int64(1), api.listStatusesCalls.Load())

	// Second call — cache hit.
	o2, err := svc.LoadListStatuses(ctx, "l1")
	require.NoError(t, err)
	assert.Equal(t, o1, o2)
	assert.Equal(t, int64(1), api.listStatusesCalls.Load())
}

func TestTaskService_UpdateTaskStatus_EvictsDetailAndListCaches(t *testing.T) {
	api := newCountingAPI()
	api.tasksByID["t1"] = &clickup.Task{
		ID:     "t1",
		Name:   "Fix login",
		Status: clickup.Status{Status: "open"},
		List:   clickup.TaskRef{ID: "l1", Name: "Sprint 42"},
	}
	api.tasks["l1"] = []clickup.Task{
		{ID: "t1", Name: "Fix login", Status: clickup.Status{Status: "open"}},
	}

	svc := app.NewTaskService(api)
	ctx := context.Background()

	// Prime both caches.
	_, err := svc.LoadTaskDetail(ctx, "t1")
	require.NoError(t, err)
	_, err = svc.LoadTasks(ctx, "l1", 0)
	require.NoError(t, err)
	assert.Equal(t, int64(1), api.taskCalls.Load())
	assert.Equal(t, int64(1), api.tasksCalls.Load())

	// Mutation: update status.
	detail, err := svc.UpdateTaskStatus(ctx, "t1", "in progress")
	require.NoError(t, err)
	assert.Equal(t, "in progress", detail.Status)

	// LoadTaskDetail should return the fresh detail cached by UpdateTaskStatus
	// without calling the API again.
	d2, err := svc.LoadTaskDetail(ctx, "t1")
	require.NoError(t, err)
	assert.Equal(t, "in progress", d2.Status)
	assert.Equal(t, int64(1), api.taskCalls.Load(), "detail should be served from cache populated by UpdateTaskStatus")

	// LoadTasks should miss the cache (evicted) and call API again.
	_, err = svc.LoadTasks(ctx, "l1", 0)
	require.NoError(t, err)
	assert.Equal(t, int64(2), api.tasksCalls.Load(), "task list cache should have been evicted by UpdateTaskStatus")
}

func TestTaskService_InvalidateTaskDetail_CausesAPIMiss(t *testing.T) {
	api := newCountingAPI()
	api.tasksByID["t1"] = &clickup.Task{
		ID:     "t1",
		Name:   "Fix login",
		Status: clickup.Status{Status: "open"},
		List:   clickup.TaskRef{ID: "l1"},
	}

	svc := app.NewTaskService(api)
	ctx := context.Background()

	// Prime cache.
	_, err := svc.LoadTaskDetail(ctx, "t1")
	require.NoError(t, err)
	assert.Equal(t, int64(1), api.taskCalls.Load())

	// Invalidate.
	svc.InvalidateTaskDetail("t1")

	// Next call should hit the API again.
	_, err = svc.LoadTaskDetail(ctx, "t1")
	require.NoError(t, err)
	assert.Equal(t, int64(2), api.taskCalls.Load())
}

func TestTaskService_InvalidateTaskList_CausesAPIMiss(t *testing.T) {
	api := newCountingAPI()
	api.tasks["l1"] = []clickup.Task{
		{ID: "t1", Name: "Task 1", Status: clickup.Status{Status: "open"}},
	}

	svc := app.NewTaskService(api)
	ctx := context.Background()

	// Prime cache for two pages.
	_, err := svc.LoadTasks(ctx, "l1", 0)
	require.NoError(t, err)
	_, err = svc.LoadTasks(ctx, "l1", 1)
	require.NoError(t, err)
	assert.Equal(t, int64(2), api.tasksCalls.Load())

	// Invalidate by listID — should evict all pages.
	svc.InvalidateTaskList("l1")

	// Both pages should miss the cache and hit the API again.
	_, err = svc.LoadTasks(ctx, "l1", 0)
	require.NoError(t, err)
	assert.Equal(t, int64(3), api.tasksCalls.Load(), "page 0 should be evicted")

	_, err = svc.LoadTasks(ctx, "l1", 1)
	require.NoError(t, err)
	assert.Equal(t, int64(4), api.tasksCalls.Load(), "page 1 should be evicted")
}
