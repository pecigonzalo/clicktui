package app_test

import (
	"context"
	"errors"
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
