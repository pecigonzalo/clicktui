// Package cli — tests for task list, view, and status commands.
package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pecigonzalo/clicktui/internal/app"
)

// --- fakes ---

// fakeTaskListSvc is a fake that satisfies taskLister.
type fakeTaskListSvc struct {
	tasks map[string][]app.TaskSummary // keyed by "listID:page"
	err   error
	errs  map[string]error // keyed by "listID:page"
}

func (f *fakeTaskListSvc) LoadTasks(_ context.Context, listID string, page int) ([]app.TaskSummary, error) {
	if f.err != nil {
		return nil, f.err
	}
	key := fmt.Sprintf("%s:%d", listID, page)
	if f.errs != nil {
		if err, ok := f.errs[key]; ok {
			return nil, err
		}
	}
	return f.tasks[key], nil
}

// fakeTaskViewSvc is a fake that satisfies taskViewer.
type fakeTaskViewSvc struct {
	detail *app.TaskDetail
	err    error
}

func (f *fakeTaskViewSvc) LoadTaskDetail(_ context.Context, _ string) (*app.TaskDetail, error) {
	return f.detail, f.err
}

// fakeTaskStatusSvc is a fake that satisfies taskStatusService.
type fakeTaskStatusSvc struct {
	detail   *app.TaskDetail
	statuses []app.StatusOption
	updated  *app.TaskDetail

	detailErr   error
	statusesErr error
	updateErr   error

	updatedID     string
	updatedStatus string
}

func (f *fakeTaskStatusSvc) LoadTaskDetail(_ context.Context, _ string) (*app.TaskDetail, error) {
	return f.detail, f.detailErr
}

func (f *fakeTaskStatusSvc) LoadListStatuses(_ context.Context, _ string) ([]app.StatusOption, error) {
	return f.statuses, f.statusesErr
}

func (f *fakeTaskStatusSvc) UpdateTaskStatus(_ context.Context, taskID, status string) (*app.TaskDetail, error) {
	f.updatedID = taskID
	f.updatedStatus = status
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	if f.updated != nil {
		return f.updated, nil
	}
	return &app.TaskDetail{ID: taskID, Status: status}, nil
}

// --- helper: build a cobra.Command whose output goes to a buffer ---

func newTestCmd(t *testing.T, buf *bytes.Buffer) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{Use: "test"}
	cmd.SetOut(buf)
	// Register a local "output" flag so resolveOutputMode works in tests.
	cmd.Flags().String("output", "text", "output format")
	return cmd
}

// newTestCmdWithOutput is like newTestCmd but sets the output flag to the given mode.
func newTestCmdWithOutput(t *testing.T, buf *bytes.Buffer, mode string) *cobra.Command {
	t.Helper()
	cmd := newTestCmd(t, buf)
	if err := cmd.Flags().Set("output", mode); err != nil {
		t.Fatalf("newTestCmdWithOutput: set output flag: %v", err)
	}
	return cmd
}

// --- task list tests ---

func TestRunTaskList_PrintsTable(t *testing.T) {
	svc := &fakeTaskListSvc{
		tasks: map[string][]app.TaskSummary{
			"l1:0": {
				{ID: "t1", Name: "Fix login", Status: "open", Priority: "high", DueDate: "2026-04-01"},
				{ID: "t2", Name: "Update docs", Status: "in progress", Priority: "none"},
			},
		},
	}

	var buf bytes.Buffer
	cmd := newTestCmd(t, &buf)

	err := runTaskList(context.Background(), svc, cmd, "l1", 0, false)
	require.NoError(t, err)

	out := buf.String()
	for _, want := range []string{"t1", "Fix login", "2026-04-01", "STATUS"} {
		assert.Contains(t, out, want)
	}
}

func TestRunTaskList_SubtaskIndented(t *testing.T) {
	svc := &fakeTaskListSvc{
		tasks: map[string][]app.TaskSummary{
			"l1:0": {
				{ID: "p1", Name: "Parent"},
				{ID: "c1", Name: "Child", Parent: "p1"},
			},
		},
	}

	var buf bytes.Buffer
	cmd := newTestCmd(t, &buf)

	err := runTaskList(context.Background(), svc, cmd, "l1", 0, false)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "↳ Child")
}

func TestRunTaskList_EmptyDueDate_ShowsDash(t *testing.T) {
	svc := &fakeTaskListSvc{
		tasks: map[string][]app.TaskSummary{
			"l1:0": {
				{ID: "t1", Name: "NoDue", Status: "open", Priority: "none"},
			},
		},
	}

	var buf bytes.Buffer
	cmd := newTestCmd(t, &buf)
	err := runTaskList(context.Background(), svc, cmd, "l1", 0, false)
	require.NoError(t, err)

	// The DUE column should show "-" when DueDate is empty.
	var taskLine string
	for line := range strings.SplitSeq(buf.String(), "\n") {
		if strings.Contains(line, "t1") {
			taskLine = line
			break
		}
	}
	require.NotEmpty(t, taskLine)
	fields := strings.Fields(taskLine)
	require.GreaterOrEqual(t, len(fields), 5)
	assert.Equal(t, "t1", fields[0])
	assert.Equal(t, "-", fields[3])
}

func TestRunTaskList_ServiceError(t *testing.T) {
	serviceErr := errors.New("api down")
	svc := &fakeTaskListSvc{err: serviceErr}

	var buf bytes.Buffer
	cmd := newTestCmd(t, &buf)

	err := runTaskList(context.Background(), svc, cmd, "l1", 0, false)
	require.Error(t, err)
	require.ErrorIs(t, err, serviceErr)
	assert.Contains(t, err.Error(), "load tasks")
}

func TestRunTaskList_AllFlag_FetchesAllPages(t *testing.T) {
	svc := &fakeTaskListSvc{
		tasks: map[string][]app.TaskSummary{
			"l1:0": {{ID: "t1", Name: "Task 1"}},
			"l1:1": {{ID: "t2", Name: "Task 2"}},
			// page 2 (key "l1:2") is absent → empty slice → stop
		},
	}

	var buf bytes.Buffer
	cmd := newTestCmd(t, &buf)
	err := runTaskList(context.Background(), svc, cmd, "l1", 0, true)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "t1")
	assert.Contains(t, out, "t2")
}

func TestRunTaskList_AllFlag_PageTwoError(t *testing.T) {
	pageErr := errors.New("page 1 unavailable")
	svc := &fakeTaskListSvc{
		tasks: map[string][]app.TaskSummary{
			"l1:0": {{ID: "t1", Name: "Task 1"}},
		},
		errs: map[string]error{
			"l1:1": pageErr,
		},
	}

	var buf bytes.Buffer
	cmd := newTestCmd(t, &buf)

	err := runTaskList(context.Background(), svc, cmd, "l1", 0, true)
	require.Error(t, err)
	require.ErrorIs(t, err, pageErr)
	assert.Contains(t, err.Error(), "page 1")
}

// --- task list: missing list ID is caught by the command's RunE ---

func TestTaskListCmd_MissingListID_ReturnsError(t *testing.T) {
	// Build a minimal root so profileFlag is available.
	root := New()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	// Point config at an empty temp dir so no list_id profile default exists.
	setTestConfigDir(t, t.TempDir())

	root.SetArgs([]string{"task", "list"})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no list ID")
}

func TestRunTaskList_ExplicitListIDUsed(t *testing.T) {
	// Verify that when runTaskList is called with an explicit list ID it uses
	// that ID to key into the service (not some other default).
	svc := &fakeTaskListSvc{
		tasks: map[string][]app.TaskSummary{
			"explicit:0": {{ID: "t9", Name: "Task from explicit list"}},
		},
	}

	var buf bytes.Buffer
	cmd := newTestCmd(t, &buf)
	err := runTaskList(context.Background(), svc, cmd, "explicit", 0, false)
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "t9")
}

// --- task view tests ---

func TestRunTaskView_PrintsDetail(t *testing.T) {
	svc := &fakeTaskViewSvc{
		detail: &app.TaskDetail{
			ID:          "abc123",
			Name:        "Fix login bug",
			Status:      "in progress",
			Priority:    "high",
			DueDate:     "2026-04-01",
			Assignees:   []string{"alice", "bob"},
			Tags:        []string{"backend", "auth"},
			URL:         "https://app.clickup.com/t/abc123",
			List:        "Sprint 12",
			Folder:      "Backend",
			Space:       "Engineering",
			Description: "Long description text here.",
		},
	}

	var buf bytes.Buffer
	cmd := newTestCmd(t, &buf)
	err := runTaskView(context.Background(), svc, cmd, "abc123")
	require.NoError(t, err)

	out := buf.String()
	for _, want := range []string{
		"abc123", "Fix login bug", "in progress", "high",
		"alice, bob", "backend, auth", "Sprint 12", "Backend", "Engineering",
		"Long description text here.",
	} {
		assert.Contains(t, out, want)
	}
}

func TestRunTaskView_SubtasksRendered(t *testing.T) {
	svc := &fakeTaskViewSvc{
		detail: &app.TaskDetail{
			ID:   "p1",
			Name: "Parent",
			Subtasks: []app.SubtaskSummary{
				{ID: "s1", Name: "Sub 1", Status: "open"},
				{ID: "s2", Name: "Sub 2", Status: "done"},
			},
		},
	}

	var buf bytes.Buffer
	cmd := newTestCmd(t, &buf)
	err := runTaskView(context.Background(), svc, cmd, "p1")
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "↳ [open]  Sub 1 (s1)")
	assert.Contains(t, out, "↳ [done]  Sub 2 (s2)")
}

func TestRunTaskView_EmptyDescription(t *testing.T) {
	svc := &fakeTaskViewSvc{
		detail: &app.TaskDetail{ID: "t1", Name: "No desc"},
	}

	var buf bytes.Buffer
	cmd := newTestCmd(t, &buf)
	err := runTaskView(context.Background(), svc, cmd, "t1")
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "Description:")
}

func TestRunTaskView_ServiceError(t *testing.T) {
	serviceErr := errors.New("not found")
	svc := &fakeTaskViewSvc{err: serviceErr}

	var buf bytes.Buffer
	cmd := newTestCmd(t, &buf)

	err := runTaskView(context.Background(), svc, cmd, "missing")
	require.Error(t, err)
	require.ErrorIs(t, err, serviceErr)
	assert.Contains(t, err.Error(), "load task detail")
}

// --- task status tests ---

func TestRunTaskStatus_Success(t *testing.T) {
	svc := &fakeTaskStatusSvc{
		detail: &app.TaskDetail{ID: "t1", ListID: "l1"},
		statuses: []app.StatusOption{
			{Name: "open"},
			{Name: "in progress"},
			{Name: "done"},
		},
	}

	var buf bytes.Buffer
	cmd := newTestCmd(t, &buf)

	err := runTaskStatus(context.Background(), svc, cmd, "t1", "in progress")
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, `status updated to "in progress"`)
	assert.Equal(t, "t1", svc.updatedID)
	assert.Equal(t, "in progress", svc.updatedStatus)
}

func TestRunTaskStatus_InvalidStatus_PrintsAvailableAndErrors(t *testing.T) {
	svc := &fakeTaskStatusSvc{
		detail: &app.TaskDetail{ID: "t1", ListID: "l1"},
		statuses: []app.StatusOption{
			{Name: "open"},
			{Name: "done"},
		},
	}

	var buf bytes.Buffer
	cmd := newTestCmd(t, &buf)

	err := runTaskStatus(context.Background(), svc, cmd, "t1", "bogus")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid status")

	// Output should list available statuses.
	out := buf.String()
	assert.Contains(t, out, "available statuses")
	assert.Contains(t, out, "open")
	assert.Contains(t, out, "done")
	// Mutation must not have been called.
	assert.Empty(t, svc.updatedID)
}

func TestRunTaskStatus_CaseInsensitiveMatch(t *testing.T) {
	svc := &fakeTaskStatusSvc{
		detail: &app.TaskDetail{ID: "t1", ListID: "l1"},
		statuses: []app.StatusOption{
			{Name: "In Progress"},
		},
	}

	var buf bytes.Buffer
	cmd := newTestCmd(t, &buf)

	err := runTaskStatus(context.Background(), svc, cmd, "t1", "in progress")
	require.NoError(t, err)
}

func TestRunTaskStatus_DetailError(t *testing.T) {
	detailErr := errors.New("task not found")
	svc := &fakeTaskStatusSvc{detailErr: detailErr}

	var buf bytes.Buffer
	cmd := newTestCmd(t, &buf)

	err := runTaskStatus(context.Background(), svc, cmd, "missing", "done")
	require.Error(t, err)
	require.ErrorIs(t, err, detailErr)
	assert.Contains(t, err.Error(), "load task detail")
}

func TestRunTaskStatus_LoadStatusesError(t *testing.T) {
	statusesErr := errors.New("list not found")
	svc := &fakeTaskStatusSvc{
		detail:      &app.TaskDetail{ID: "t1", ListID: "l1"},
		statusesErr: statusesErr,
	}

	var buf bytes.Buffer
	cmd := newTestCmd(t, &buf)

	err := runTaskStatus(context.Background(), svc, cmd, "t1", "done")
	require.Error(t, err)
	require.ErrorIs(t, err, statusesErr)
	assert.Contains(t, err.Error(), "load list statuses")
}

func TestRunTaskStatus_UpdateError(t *testing.T) {
	updateErr := errors.New("forbidden")
	svc := &fakeTaskStatusSvc{
		detail:    &app.TaskDetail{ID: "t1", ListID: "l1"},
		statuses:  []app.StatusOption{{Name: "done"}},
		updateErr: updateErr,
	}

	var buf bytes.Buffer
	cmd := newTestCmd(t, &buf)

	err := runTaskStatus(context.Background(), svc, cmd, "t1", "done")
	require.Error(t, err)
	require.ErrorIs(t, err, updateErr)
	assert.Contains(t, err.Error(), "update task status")
}

// --- subcommand registration ---

func TestNewTaskCmd_HasExpectedSubcommands(t *testing.T) {
	cmd := newTaskCmd()
	registered := make(map[string]bool)
	for _, c := range cmd.Commands() {
		registered[c.Name()] = true
	}
	for _, want := range []string{"list", "view", "status", "create", "move", "update"} {
		assert.True(t, registered[want], "expected subcommand %q to be registered; have: %v", want, registered)
	}
}

// --- task create fakes ---

// fakeTaskCreateSvc is a fake that satisfies taskCreator.
type fakeTaskCreateSvc struct {
	taskID         string
	err            error
	capturedListID string
	capturedInput  app.CreateTaskInput
}

func (f *fakeTaskCreateSvc) CreateTask(_ context.Context, listID string, input app.CreateTaskInput) (string, error) {
	f.capturedListID = listID
	f.capturedInput = input
	if f.err != nil {
		return "", f.err
	}
	return f.taskID, nil
}

// --- task create tests ---

func TestRunTaskCreate_HappyPath(t *testing.T) {
	svc := &fakeTaskCreateSvc{taskID: "new-task-id"}

	var buf bytes.Buffer
	cmd := newTestCmd(t, &buf)

	input := app.CreateTaskInput{Name: "Fix login bug"}
	err := runTaskCreate(context.Background(), svc, cmd, "list-1", input)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "Task created: new-task-id")
	assert.Equal(t, "list-1", svc.capturedListID)
}

func TestRunTaskCreate_ServiceError(t *testing.T) {
	serviceErr := errors.New("api error")
	svc := &fakeTaskCreateSvc{err: serviceErr}

	var buf bytes.Buffer
	cmd := newTestCmd(t, &buf)

	err := runTaskCreate(context.Background(), svc, cmd, "list-1", app.CreateTaskInput{Name: "Task"})
	require.Error(t, err)
	require.ErrorIs(t, err, serviceErr)
	assert.Contains(t, err.Error(), "create task")
}

func TestTaskCreateCmd_MissingName_ReturnsError(t *testing.T) {
	root := New()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	setTestConfigDir(t, t.TempDir())

	root.SetArgs([]string{"task", "create", "--list", "l1"})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name")
}

func TestTaskCreateCmd_MissingListID_ReturnsError(t *testing.T) {
	root := New()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	setTestConfigDir(t, t.TempDir())

	root.SetArgs([]string{"task", "create", "--name", "My Task"})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no list ID")
}

// --- task move fakes ---

// fakeTaskMoveSvc is a fake that satisfies taskMover.
type fakeTaskMoveSvc struct {
	detail            *app.TaskDetail
	err               error
	capturedWorkspace string
	capturedTaskID    string
	capturedListID    string
}

func (f *fakeTaskMoveSvc) MoveTaskToList(_ context.Context, workspaceID, taskID, listID string) (*app.TaskDetail, error) {
	f.capturedWorkspace = workspaceID
	f.capturedTaskID = taskID
	f.capturedListID = listID
	if f.err != nil {
		return nil, f.err
	}
	if f.detail != nil {
		return f.detail, nil
	}
	return &app.TaskDetail{ID: taskID, ListID: listID}, nil
}

// --- task move tests ---

func TestRunTaskMove_HappyPath(t *testing.T) {
	svc := &fakeTaskMoveSvc{}

	var buf bytes.Buffer
	cmd := newTestCmd(t, &buf)

	err := runTaskMove(context.Background(), svc, cmd, "ws-1", "task-abc", "list-99")
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "Task task-abc moved to list list-99.")
	assert.Equal(t, "ws-1", svc.capturedWorkspace)
}

func TestRunTaskMove_ServiceError(t *testing.T) {
	serviceErr := errors.New("forbidden")
	svc := &fakeTaskMoveSvc{err: serviceErr}

	var buf bytes.Buffer
	cmd := newTestCmd(t, &buf)

	err := runTaskMove(context.Background(), svc, cmd, "ws-1", "task-abc", "list-99")
	require.Error(t, err)
	require.ErrorIs(t, err, serviceErr)
	assert.Contains(t, err.Error(), "move task")
}

func TestTaskMoveCmd_MissingWorkspaceID_ReturnsError(t *testing.T) {
	root := New()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	setTestConfigDir(t, t.TempDir())

	root.SetArgs([]string{"task", "move", "task-abc", "--to-list", "list-99"})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no workspace ID")
}

// --- task update fakes ---

// fakeTaskUpdateSvc is a fake that satisfies taskUpdater.
type fakeTaskUpdateSvc struct {
	err            error
	capturedTaskID string
	capturedInput  app.UpdateTaskInput
}

func (f *fakeTaskUpdateSvc) UpdateTask(_ context.Context, taskID string, input app.UpdateTaskInput) error {
	f.capturedTaskID = taskID
	f.capturedInput = input
	return f.err
}

// --- task update tests ---

func TestRunTaskUpdate_HappyPath_Name(t *testing.T) {
	svc := &fakeTaskUpdateSvc{}

	var buf bytes.Buffer
	cmd := newTestCmd(t, &buf)

	name := "New title"
	input := app.UpdateTaskInput{Name: &name}

	err := runTaskUpdate(context.Background(), svc, cmd, "task-xyz", input)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "Task task-xyz updated.")
	assert.Equal(t, "task-xyz", svc.capturedTaskID)
	require.NotNil(t, svc.capturedInput.Name)
	assert.Equal(t, "New title", *svc.capturedInput.Name)
}

func TestRunTaskUpdate_ServiceError(t *testing.T) {
	serviceErr := errors.New("forbidden")
	svc := &fakeTaskUpdateSvc{err: serviceErr}

	var buf bytes.Buffer
	cmd := newTestCmd(t, &buf)

	name := "New title"
	err := runTaskUpdate(context.Background(), svc, cmd, "task-xyz", app.UpdateTaskInput{Name: &name})
	require.Error(t, err)
	require.ErrorIs(t, err, serviceErr)
	assert.Contains(t, err.Error(), "update task")
}

func TestTaskUpdateCmd_NoFields_ReturnsError(t *testing.T) {
	root := New()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	setTestConfigDir(t, t.TempDir())

	root.SetArgs([]string{"task", "update", "task-abc"})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no update fields specified")
}

func TestTaskUpdateCmd_MutuallyExclusiveDescriptionFlags_ReturnsError(t *testing.T) {
	root := New()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	setTestConfigDir(t, t.TempDir())

	root.SetArgs([]string{"task", "update", "task-abc", "--description", "desc", "--clear-description"})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--description and --clear-description are mutually exclusive")
}

func TestTaskUpdateCmd_MutuallyExclusiveDueFlags_ReturnsError(t *testing.T) {
	root := New()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	setTestConfigDir(t, t.TempDir())

	root.SetArgs([]string{"task", "update", "task-abc", "--due", "2026-05-01", "--clear-due"})
	err := root.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--due and --clear-due are mutually exclusive")
}

// --- helper conversion tests ---

func TestParsePriority(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{"urgent", "urgent", 1, false},
		{"high", "high", 2, false},
		{"uppercase_rejected", "HIGH", 0, true},
		{"normal", "normal", 3, false},
		{"low", "low", 4, false},
		{"none", "none", 0, false},
		{"empty", "", 0, false},
		{"unknown", "critical", 0, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := parsePriority(tc.input)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestParseDueDate(t *testing.T) {
	cases := []struct {
		name      string
		input     string
		wantMS    bool // only check that result is non-empty epoch ms when true
		wantErr   bool
		wantEmpty bool
	}{
		{"valid_date", "2026-05-01", true, false, false},
		{"empty_input", "", false, false, true},
		{"invalid_format", "05/01/2026", false, true, false},
		{"invalid_date", "not-a-date", false, true, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseDueDate(tc.input)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tc.wantEmpty {
				assert.Equal(t, "", got)
			}
			if tc.wantMS {
				assert.NotEmpty(t, got)
			}
		})
	}
}

func TestParseDueDate_EpochMillisValue(t *testing.T) {
	// 2026-05-01 UTC should produce 1777593600000 ms.
	got, err := parseDueDate("2026-05-01")
	require.NoError(t, err)
	const want = "1777593600000"
	assert.Equal(t, want, got)
}

// --- JSON output mode tests ---

func TestRunTaskList_JSONMode(t *testing.T) {
	svc := &fakeTaskListSvc{
		tasks: map[string][]app.TaskSummary{
			"l1:0": {
				{ID: "t1", Name: "Fix login", Status: "open", Priority: "high", DueDate: "2026-04-01"},
				{ID: "t2", Name: "Update docs", Status: "in progress", Priority: "none"},
			},
		},
	}

	var buf bytes.Buffer
	cmd := newTestCmdWithOutput(t, &buf, "json")

	err := runTaskList(context.Background(), svc, cmd, "l1", 0, false)
	require.NoError(t, err)

	var got []app.TaskSummary
	err = json.Unmarshal(buf.Bytes(), &got)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "t1", got[0].ID)
	assert.Equal(t, "t2", got[1].ID)
}

func TestRunTaskList_JSONMode_EmptySlice(t *testing.T) {
	svc := &fakeTaskListSvc{
		tasks: map[string][]app.TaskSummary{},
	}

	var buf bytes.Buffer
	cmd := newTestCmdWithOutput(t, &buf, "json")

	err := runTaskList(context.Background(), svc, cmd, "l1", 0, false)
	require.NoError(t, err)

	// Should encode as [] not null.
	trimmed := strings.TrimSpace(buf.String())
	assert.Equal(t, "[]", trimmed)
}

func TestRunTaskView_JSONMode(t *testing.T) {
	svc := &fakeTaskViewSvc{
		detail: &app.TaskDetail{
			ID:        "abc123",
			Name:      "Fix login bug",
			Status:    "in progress",
			Priority:  "high",
			DueDate:   "2026-04-01",
			Assignees: []string{"alice", "bob"},
			Tags:      []string{"backend"},
			URL:       "https://app.clickup.com/t/abc123",
			List:      "Sprint 12",
		},
	}

	var buf bytes.Buffer
	cmd := newTestCmdWithOutput(t, &buf, "json")

	err := runTaskView(context.Background(), svc, cmd, "abc123")
	require.NoError(t, err)

	var got app.TaskDetail
	err = json.Unmarshal(buf.Bytes(), &got)
	require.NoError(t, err)
	assert.Equal(t, "abc123", got.ID)
	assert.Equal(t, "Fix login bug", got.Name)
	require.Len(t, got.Assignees, 2)
}

func TestRunTaskView_JSONMode_NilSubtasks(t *testing.T) {
	svc := &fakeTaskViewSvc{
		detail: &app.TaskDetail{
			ID:       "abc123",
			Name:     "Fix login bug",
			Subtasks: nil,
		},
	}

	var buf bytes.Buffer
	cmd := newTestCmdWithOutput(t, &buf, "json")

	err := runTaskView(context.Background(), svc, cmd, "abc123")
	require.NoError(t, err)
	assert.Contains(t, strings.TrimSpace(buf.String()), `"subtasks":null`)
}

func TestRunTaskStatus_JSONMode(t *testing.T) {
	svc := &fakeTaskStatusSvc{
		detail: &app.TaskDetail{ID: "t1", ListID: "l1"},
		statuses: []app.StatusOption{
			{Name: "done"},
		},
	}

	var buf bytes.Buffer
	cmd := newTestCmdWithOutput(t, &buf, "json")

	err := runTaskStatus(context.Background(), svc, cmd, "t1", "done")
	require.NoError(t, err)

	var got taskMutationResult
	err = json.Unmarshal(buf.Bytes(), &got)
	require.NoError(t, err)
	assert.True(t, got.OK)
	assert.Equal(t, "t1", got.ID)
}

func TestRunTaskCreate_JSONMode(t *testing.T) {
	svc := &fakeTaskCreateSvc{taskID: "new-task-id"}

	var buf bytes.Buffer
	cmd := newTestCmdWithOutput(t, &buf, "json")

	input := app.CreateTaskInput{Name: "Fix login bug"}
	err := runTaskCreate(context.Background(), svc, cmd, "list-1", input)
	require.NoError(t, err)

	var got taskMutationResult
	err = json.Unmarshal(buf.Bytes(), &got)
	require.NoError(t, err)
	assert.True(t, got.OK)
	assert.Equal(t, "new-task-id", got.ID)
}

func TestRunTaskMove_JSONMode(t *testing.T) {
	svc := &fakeTaskMoveSvc{}

	var buf bytes.Buffer
	cmd := newTestCmdWithOutput(t, &buf, "json")

	err := runTaskMove(context.Background(), svc, cmd, "ws-1", "task-abc", "list-99")
	require.NoError(t, err)

	var got taskMutationResult
	err = json.Unmarshal(buf.Bytes(), &got)
	require.NoError(t, err)
	assert.True(t, got.OK)
	assert.Equal(t, "task-abc", got.ID)
}

func TestRunTaskUpdate_JSONMode(t *testing.T) {
	svc := &fakeTaskUpdateSvc{}

	var buf bytes.Buffer
	cmd := newTestCmdWithOutput(t, &buf, "json")

	name := "New title"
	input := app.UpdateTaskInput{Name: &name}

	err := runTaskUpdate(context.Background(), svc, cmd, "task-xyz", input)
	require.NoError(t, err)

	var got taskMutationResult
	err = json.Unmarshal(buf.Bytes(), &got)
	require.NoError(t, err)
	assert.True(t, got.OK)
	assert.Equal(t, "task-xyz", got.ID)
}

// --- invalid output mode test ---

func TestRunTaskCreate_InvalidOutputMode_ErrorBeforeServiceCall(t *testing.T) {
	svc := &fakeTaskCreateSvc{taskID: "irrelevant"}

	var buf bytes.Buffer
	cmd := newTestCmdWithOutput(t, &buf, "xml") // unsupported format

	input := app.CreateTaskInput{Name: "Task"}
	err := runTaskCreate(context.Background(), svc, cmd, "list-1", input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported output format")
	assert.Equal(t, "", svc.capturedListID)
}
