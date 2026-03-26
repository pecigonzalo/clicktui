// Package cli — tests for task list, view, and status commands.
package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/pecigonzalo/clicktui/internal/app"
)

// --- fakes ---

// fakeTaskListSvc is a fake that satisfies taskLister.
type fakeTaskListSvc struct {
	tasks map[string][]app.TaskSummary // keyed by "listID:page"
	err   error
}

func (f *fakeTaskListSvc) LoadTasks(_ context.Context, listID string, page int) ([]app.TaskSummary, error) {
	if f.err != nil {
		return nil, f.err
	}
	key := fmt.Sprintf("%s:%d", listID, page)
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

func newTestCmd(buf *bytes.Buffer) *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.SetOut(buf)
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
	cmd := newTestCmd(&buf)

	if err := runTaskList(context.Background(), svc, cmd, "l1", 0, false); err != nil {
		t.Fatalf("runTaskList() error = %v", err)
	}

	out := buf.String()
	for _, want := range []string{"t1", "Fix login", "2026-04-01", "STATUS"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output, got:\n%s", want, out)
		}
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
	cmd := newTestCmd(&buf)

	if err := runTaskList(context.Background(), svc, cmd, "l1", 0, false); err != nil {
		t.Fatalf("runTaskList() error = %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "↳ Child") {
		t.Errorf("expected subtask prefix '↳ Child' in output, got:\n%s", out)
	}
}

func TestRunTaskList_EmptyDueDate_ShowsDash(t *testing.T) {
	svc := &fakeTaskListSvc{
		tasks: map[string][]app.TaskSummary{
			"l1:0": {
				{ID: "t1", Name: "No due", Status: "open"},
			},
		},
	}

	var buf bytes.Buffer
	cmd := newTestCmd(&buf)
	if err := runTaskList(context.Background(), svc, cmd, "l1", 0, false); err != nil {
		t.Fatalf("runTaskList() error = %v", err)
	}

	// The DUE column should show "-" when DueDate is empty.
	if !strings.Contains(buf.String(), "-") {
		t.Errorf("expected '-' for missing due date, got:\n%s", buf.String())
	}
}

func TestRunTaskList_ServiceError(t *testing.T) {
	svc := &fakeTaskListSvc{err: errors.New("api down")}

	var buf bytes.Buffer
	cmd := newTestCmd(&buf)

	err := runTaskList(context.Background(), svc, cmd, "l1", 0, false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "load tasks") {
		t.Errorf("error should mention 'load tasks', got: %v", err)
	}
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
	cmd := newTestCmd(&buf)
	if err := runTaskList(context.Background(), svc, cmd, "l1", 0, true); err != nil {
		t.Fatalf("runTaskList(all=true) error = %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "t1") {
		t.Errorf("expected t1 in all-pages output, got:\n%s", out)
	}
	if !strings.Contains(out, "t2") {
		t.Errorf("expected t2 in all-pages output, got:\n%s", out)
	}
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
	if err == nil {
		t.Fatal("expected error when no list ID is provided, got nil")
	}
	if !strings.Contains(err.Error(), "no list ID") {
		t.Errorf("expected 'no list ID' error, got: %v", err)
	}
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
	cmd := newTestCmd(&buf)
	if err := runTaskList(context.Background(), svc, cmd, "explicit", 0, false); err != nil {
		t.Fatalf("runTaskList() error = %v", err)
	}

	if !strings.Contains(buf.String(), "t9") {
		t.Errorf("expected task from explicit list, got:\n%s", buf.String())
	}
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
	cmd := newTestCmd(&buf)
	if err := runTaskView(context.Background(), svc, cmd, "abc123"); err != nil {
		t.Fatalf("runTaskView() error = %v", err)
	}

	out := buf.String()
	for _, want := range []string{
		"abc123", "Fix login bug", "in progress", "high",
		"alice, bob", "backend, auth", "Sprint 12", "Backend", "Engineering",
		"Long description text here.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output, got:\n%s", want, out)
		}
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
	cmd := newTestCmd(&buf)
	if err := runTaskView(context.Background(), svc, cmd, "p1"); err != nil {
		t.Fatalf("runTaskView() error = %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "↳ [open]  Sub 1 (s1)") {
		t.Errorf("expected subtask line in output, got:\n%s", out)
	}
	if !strings.Contains(out, "↳ [done]  Sub 2 (s2)") {
		t.Errorf("expected subtask line in output, got:\n%s", out)
	}
}

func TestRunTaskView_EmptyDescription(t *testing.T) {
	svc := &fakeTaskViewSvc{
		detail: &app.TaskDetail{ID: "t1", Name: "No desc"},
	}

	var buf bytes.Buffer
	cmd := newTestCmd(&buf)
	if err := runTaskView(context.Background(), svc, cmd, "t1"); err != nil {
		t.Fatalf("runTaskView() error = %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Description:") {
		t.Errorf("expected 'Description:' in output, got:\n%s", out)
	}
}

func TestRunTaskView_ServiceError(t *testing.T) {
	svc := &fakeTaskViewSvc{err: errors.New("not found")}

	var buf bytes.Buffer
	cmd := newTestCmd(&buf)

	err := runTaskView(context.Background(), svc, cmd, "missing")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "load task detail") {
		t.Errorf("expected 'load task detail' in error, got: %v", err)
	}
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
	cmd := newTestCmd(&buf)

	if err := runTaskStatus(context.Background(), svc, cmd, "t1", "in progress"); err != nil {
		t.Fatalf("runTaskStatus() error = %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, `status updated to "in progress"`) {
		t.Errorf("expected success message in output, got:\n%s", out)
	}
	if svc.updatedID != "t1" || svc.updatedStatus != "in progress" {
		t.Errorf("UpdateTaskStatus called with (%q, %q), want (t1, in progress)",
			svc.updatedID, svc.updatedStatus)
	}
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
	cmd := newTestCmd(&buf)

	err := runTaskStatus(context.Background(), svc, cmd, "t1", "bogus")
	if err == nil {
		t.Fatal("expected error for invalid status, got nil")
	}
	if !strings.Contains(err.Error(), "invalid status") {
		t.Errorf("expected 'invalid status' in error, got: %v", err)
	}

	// Output should list available statuses.
	out := buf.String()
	if !strings.Contains(out, "available statuses") {
		t.Errorf("expected 'available statuses' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "open") || !strings.Contains(out, "done") {
		t.Errorf("expected status names in output, got:\n%s", out)
	}
	// Mutation must not have been called.
	if svc.updatedID != "" {
		t.Errorf("UpdateTaskStatus should not have been called for invalid status")
	}
}

func TestRunTaskStatus_CaseInsensitiveMatch(t *testing.T) {
	svc := &fakeTaskStatusSvc{
		detail: &app.TaskDetail{ID: "t1", ListID: "l1"},
		statuses: []app.StatusOption{
			{Name: "In Progress"},
		},
	}

	var buf bytes.Buffer
	cmd := newTestCmd(&buf)

	if err := runTaskStatus(context.Background(), svc, cmd, "t1", "in progress"); err != nil {
		t.Fatalf("runTaskStatus() error = %v; expected case-insensitive match to succeed", err)
	}
}

func TestRunTaskStatus_DetailError(t *testing.T) {
	svc := &fakeTaskStatusSvc{detailErr: errors.New("task not found")}

	var buf bytes.Buffer
	cmd := newTestCmd(&buf)

	err := runTaskStatus(context.Background(), svc, cmd, "missing", "done")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "load task detail") {
		t.Errorf("expected 'load task detail' in error, got: %v", err)
	}
}

func TestRunTaskStatus_LoadStatusesError(t *testing.T) {
	svc := &fakeTaskStatusSvc{
		detail:      &app.TaskDetail{ID: "t1", ListID: "l1"},
		statusesErr: errors.New("list not found"),
	}

	var buf bytes.Buffer
	cmd := newTestCmd(&buf)

	err := runTaskStatus(context.Background(), svc, cmd, "t1", "done")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "load list statuses") {
		t.Errorf("expected 'load list statuses' in error, got: %v", err)
	}
}

func TestRunTaskStatus_UpdateError(t *testing.T) {
	svc := &fakeTaskStatusSvc{
		detail:    &app.TaskDetail{ID: "t1", ListID: "l1"},
		statuses:  []app.StatusOption{{Name: "done"}},
		updateErr: errors.New("forbidden"),
	}

	var buf bytes.Buffer
	cmd := newTestCmd(&buf)

	err := runTaskStatus(context.Background(), svc, cmd, "t1", "done")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "update task status") {
		t.Errorf("expected 'update task status' in error, got: %v", err)
	}
}

// --- subcommand registration ---

func TestNewTaskCmd_HasExpectedSubcommands(t *testing.T) {
	cmd := newTaskCmd()
	registered := make(map[string]bool)
	for _, c := range cmd.Commands() {
		registered[c.Name()] = true
	}
	for _, want := range []string{"list", "view", "status", "create", "move", "update"} {
		if !registered[want] {
			t.Errorf("expected subcommand %q to be registered; have: %v", want, registered)
		}
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
	cmd := newTestCmd(&buf)

	input := app.CreateTaskInput{Name: "Fix login bug"}
	if err := runTaskCreate(context.Background(), svc, cmd, "list-1", input); err != nil {
		t.Fatalf("runTaskCreate() error = %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Created task new-task-id.") {
		t.Errorf("expected 'Created task new-task-id.' in output, got:\n%s", out)
	}
	if svc.capturedListID != "list-1" {
		t.Errorf("CreateTask listID = %q, want %q", svc.capturedListID, "list-1")
	}
}

func TestRunTaskCreate_ServiceError(t *testing.T) {
	svc := &fakeTaskCreateSvc{err: errors.New("api error")}

	var buf bytes.Buffer
	cmd := newTestCmd(&buf)

	err := runTaskCreate(context.Background(), svc, cmd, "list-1", app.CreateTaskInput{Name: "Task"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "create task") {
		t.Errorf("expected 'create task' in error, got: %v", err)
	}
}

func TestTaskCreateCmd_MissingName_ReturnsError(t *testing.T) {
	root := New()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	setTestConfigDir(t, t.TempDir())

	root.SetArgs([]string{"task", "create", "--list", "l1"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when --name is missing, got nil")
	}
	if !strings.Contains(err.Error(), "name") {
		t.Errorf("expected 'name' in error, got: %v", err)
	}
}

func TestTaskCreateCmd_MissingListID_ReturnsError(t *testing.T) {
	root := New()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	setTestConfigDir(t, t.TempDir())

	root.SetArgs([]string{"task", "create", "--name", "My Task"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when no list ID is provided, got nil")
	}
	if !strings.Contains(err.Error(), "no list ID") {
		t.Errorf("expected 'no list ID' in error, got: %v", err)
	}
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
	cmd := newTestCmd(&buf)

	if err := runTaskMove(context.Background(), svc, cmd, "ws-1", "task-abc", "list-99"); err != nil {
		t.Fatalf("runTaskMove() error = %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Moved task task-abc to list list-99.") {
		t.Errorf("expected 'Moved task task-abc to list list-99.' in output, got:\n%s", out)
	}
	if svc.capturedWorkspace != "ws-1" {
		t.Errorf("MoveTaskToList workspaceID = %q, want %q", svc.capturedWorkspace, "ws-1")
	}
}

func TestRunTaskMove_ServiceError(t *testing.T) {
	svc := &fakeTaskMoveSvc{err: errors.New("forbidden")}

	var buf bytes.Buffer
	cmd := newTestCmd(&buf)

	err := runTaskMove(context.Background(), svc, cmd, "ws-1", "task-abc", "list-99")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "move task") {
		t.Errorf("expected 'move task' in error, got: %v", err)
	}
}

func TestTaskMoveCmd_MissingWorkspaceID_ReturnsError(t *testing.T) {
	root := New()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	setTestConfigDir(t, t.TempDir())

	root.SetArgs([]string{"task", "move", "task-abc", "--to-list", "list-99"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when no workspace ID is provided, got nil")
	}
	if !strings.Contains(err.Error(), "no workspace ID") {
		t.Errorf("expected 'no workspace ID' in error, got: %v", err)
	}
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
	cmd := newTestCmd(&buf)

	name := "New title"
	input := app.UpdateTaskInput{Name: &name}

	if err := runTaskUpdate(context.Background(), svc, cmd, "task-xyz", input); err != nil {
		t.Fatalf("runTaskUpdate() error = %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Updated task task-xyz.") {
		t.Errorf("expected 'Updated task task-xyz.' in output, got:\n%s", out)
	}
	if svc.capturedTaskID != "task-xyz" {
		t.Errorf("UpdateTask taskID = %q, want %q", svc.capturedTaskID, "task-xyz")
	}
	if svc.capturedInput.Name == nil || *svc.capturedInput.Name != "New title" {
		t.Errorf("UpdateTask input.Name = %v, want %q", svc.capturedInput.Name, "New title")
	}
}

func TestRunTaskUpdate_ServiceError(t *testing.T) {
	svc := &fakeTaskUpdateSvc{err: errors.New("forbidden")}

	var buf bytes.Buffer
	cmd := newTestCmd(&buf)

	name := "New title"
	err := runTaskUpdate(context.Background(), svc, cmd, "task-xyz", app.UpdateTaskInput{Name: &name})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "update task") {
		t.Errorf("expected 'update task' in error, got: %v", err)
	}
}

func TestTaskUpdateCmd_NoFields_ReturnsError(t *testing.T) {
	root := New()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	setTestConfigDir(t, t.TempDir())

	root.SetArgs([]string{"task", "update", "task-abc"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when no update fields are specified, got nil")
	}
	if !strings.Contains(err.Error(), "no update fields specified") {
		t.Errorf("expected 'no update fields specified' in error, got: %v", err)
	}
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
		{"normal", "normal", 3, false},
		{"low", "low", 4, false},
		{"none", "none", 0, false},
		{"empty", "", 0, false},
		{"unknown", "critical", 0, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parsePriority(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Errorf("parsePriority(%q) = %d, want error", tc.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parsePriority(%q) error = %v", tc.input, err)
			}
			if got != tc.want {
				t.Errorf("parsePriority(%q) = %d, want %d", tc.input, got, tc.want)
			}
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
			got, err := parseDueDate(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Errorf("parseDueDate(%q) = %q, want error", tc.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseDueDate(%q) error = %v", tc.input, err)
			}
			if tc.wantEmpty && got != "" {
				t.Errorf("parseDueDate(%q) = %q, want empty string", tc.input, got)
			}
			if tc.wantMS && got == "" {
				t.Errorf("parseDueDate(%q) = %q, want non-empty epoch ms string", tc.input, got)
			}
		})
	}
}

func TestParseDueDate_EpochMillisValue(t *testing.T) {
	// 2026-05-01 UTC should produce 1777593600000 ms.
	got, err := parseDueDate("2026-05-01")
	if err != nil {
		t.Fatalf("parseDueDate(\"2026-05-01\") error = %v", err)
	}
	const want = "1777593600000"
	if got != want {
		t.Errorf("parseDueDate(\"2026-05-01\") = %q, want %q", got, want)
	}
}
