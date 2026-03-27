package clickup_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pecigonzalo/clicktui/internal/clickup"
)

func TestSpaces_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/team/t1/space", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"spaces": []map[string]any{
				{"id": "s1", "name": "Engineering"},
				{"id": "s2", "name": "Marketing"},
			},
		})
	}))
	defer srv.Close()

	client := newTestClient(t, "pk_test", srv)
	spaces, err := client.Spaces(context.Background(), "t1")
	require.NoError(t, err)
	require.Len(t, spaces, 2)
	assert.Equal(t, "Engineering", spaces[0].Name)
	assert.Equal(t, "s2", spaces[1].ID)
}

func TestFolders_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/space/s1/folder", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"folders": []map[string]any{
				{
					"id":   "f1",
					"name": "Backend",
					"lists": []map[string]any{
						{"id": "l1", "name": "Sprint 42"},
					},
				},
			},
		})
	}))
	defer srv.Close()

	client := newTestClient(t, "pk_test", srv)
	folders, err := client.Folders(context.Background(), "s1")
	require.NoError(t, err)
	require.Len(t, folders, 1)
	assert.Equal(t, "Backend", folders[0].Name)
	require.Len(t, folders[0].Lists, 1)
	assert.Equal(t, "Sprint 42", folders[0].Lists[0].Name)
}

func TestFolderlessLists_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/space/s1/list", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"lists": []map[string]any{
				{"id": "l2", "name": "Backlog"},
			},
		})
	}))
	defer srv.Close()

	client := newTestClient(t, "pk_test", srv)
	lists, err := client.FolderlessLists(context.Background(), "s1")
	require.NoError(t, err)
	require.Len(t, lists, 1)
	assert.Equal(t, "Backlog", lists[0].Name)
}

func TestTasks_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/list/l1/task", r.URL.Path)
		assert.Equal(t, "0", r.URL.Query().Get("page"))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"tasks": []map[string]any{
				{
					"id":   "abc",
					"name": "Fix bug",
					"status": map[string]any{
						"status": "open",
						"color":  "#d3d3d3",
						"type":   "open",
					},
				},
			},
		})
	}))
	defer srv.Close()

	client := newTestClient(t, "pk_test", srv)
	tasks, err := client.Tasks(context.Background(), "l1", 0)
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	assert.Equal(t, "Fix bug", tasks[0].Name)
	assert.Equal(t, "open", tasks[0].Status.Status)
}

func TestTask_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/task/abc", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":   "abc",
			"name": "Fix bug",
			"status": map[string]any{
				"status": "open",
				"color":  "#d3d3d3",
				"type":   "open",
			},
			"assignees": []map[string]any{
				{"id": 1, "username": "alice", "email": "alice@example.com"},
			},
			"tags": []map[string]any{
				{"name": "urgent"},
			},
			"url":          "https://app.clickup.com/t/abc",
			"date_created": "1700000000000",
			"list":         map[string]any{"id": "l1", "name": "Sprint 42"},
			"folder":       map[string]any{"id": "f1", "name": "Backend"},
			"space":        map[string]any{"id": "s1", "name": "Engineering"},
		})
	}))
	defer srv.Close()

	client := newTestClient(t, "pk_test", srv)
	task, err := client.Task(context.Background(), "abc")
	require.NoError(t, err)
	assert.Equal(t, "Fix bug", task.Name)
	assert.Equal(t, "open", task.Status.Status)
	require.Len(t, task.Assignees, 1)
	assert.Equal(t, "alice", task.Assignees[0].Username)
	assert.Equal(t, "Sprint 42", task.List.Name)
}

func TestTasks_ErrorStatuses(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{name: "rate_limit", status: http.StatusTooManyRequests, body: `{"err":"Rate limit exceeded"}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()

			client := newTestClient(t, "pk_test", srv)
			_, err := client.Tasks(context.Background(), "l1", 0)
			require.Error(t, err)

			var apiErr *clickup.APIError
			require.ErrorAs(t, err, &apiErr)
			assert.Equal(t, tc.status, apiErr.StatusCode)
		})
	}
}

func TestSpaces_ErrorStatuses(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		token  string
	}{
		{name: "unauthorized", status: http.StatusUnauthorized, body: `{"err":"Token invalid"}`, token: "bad_token"},
		{name: "rate_limit", status: http.StatusTooManyRequests, body: `{"err":"Rate limit exceeded"}`, token: "pk_test"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()

			client := newTestClient(t, tc.token, srv)
			_, err := client.Spaces(context.Background(), "t1")
			require.Error(t, err)

			var apiErr *clickup.APIError
			require.ErrorAs(t, err, &apiErr)
			assert.Equal(t, tc.status, apiErr.StatusCode)
		})
	}
}

func TestFolders_ErrorStatuses(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{name: "unauthorized", status: http.StatusUnauthorized, body: `{"err":"Token invalid"}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()

			client := newTestClient(t, "bad_token", srv)
			_, err := client.Folders(context.Background(), "s1")
			require.Error(t, err)

			var apiErr *clickup.APIError
			require.ErrorAs(t, err, &apiErr)
			assert.Equal(t, tc.status, apiErr.StatusCode)
		})
	}
}

func TestFolderlessLists_ErrorStatuses(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{name: "rate_limit", status: http.StatusTooManyRequests, body: `{"err":"Rate limit exceeded"}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()

			client := newTestClient(t, "pk_test", srv)
			_, err := client.FolderlessLists(context.Background(), "s1")
			require.Error(t, err)

			var apiErr *clickup.APIError
			require.ErrorAs(t, err, &apiErr)
			assert.Equal(t, tc.status, apiErr.StatusCode)
		})
	}
}

func TestTask_ErrorStatuses(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		token  string
		taskID string
	}{
		{name: "not_found", status: http.StatusNotFound, body: `{"err":"Task not found"}`, token: "pk_test", taskID: "missing"},
		{name: "unauthorized", status: http.StatusUnauthorized, body: `{"err":"Token invalid"}`, token: "bad_token", taskID: "t1"},
		{name: "rate_limit", status: http.StatusTooManyRequests, body: `{"err":"Rate limit exceeded"}`, token: "pk_test", taskID: "t1"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()

			client := newTestClient(t, tc.token, srv)
			_, err := client.Task(context.Background(), tc.taskID)
			require.Error(t, err)

			var apiErr *clickup.APIError
			require.ErrorAs(t, err, &apiErr)
			assert.Equal(t, tc.status, apiErr.StatusCode)
		})
	}
}

func TestGetList_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/list/l1", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":   "l1",
			"name": "Sprint 42",
			"space": map[string]any{
				"id":   "s1",
				"name": "Engineering",
			},
			"folder": map[string]any{
				"id":   "f1",
				"name": "Backend",
			},
			"statuses": []map[string]any{
				{"status": "open", "color": "#d3d3d3", "type": "open"},
			},
		})
	}))
	defer srv.Close()

	client := newTestClient(t, "pk_test", srv)
	list, err := client.GetList(context.Background(), "l1")
	require.NoError(t, err)
	require.NotNil(t, list)
	assert.Equal(t, "l1", list.ID)
	assert.Equal(t, "Sprint 42", list.Name)
	assert.Equal(t, "s1", list.Space.ID)
	assert.Equal(t, "f1", list.Folder.ID)
}

func TestGetList_Error(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		listID string
	}{
		{name: "not_found", status: http.StatusNotFound, body: `{"err":"List not found"}`, listID: "missing"},
		{name: "rate_limit", status: http.StatusTooManyRequests, body: `{"err":"Rate limit exceeded"}`, listID: "l1"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()

			client := newTestClient(t, "pk_test", srv)
			_, err := client.GetList(context.Background(), tc.listID)
			require.Error(t, err)

			var apiErr *clickup.APIError
			require.ErrorAs(t, err, &apiErr)
			assert.Equal(t, tc.status, apiErr.StatusCode)
		})
	}
}
