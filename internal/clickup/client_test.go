package clickup_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pecigonzalo/clicktui/internal/auth"
	"github.com/pecigonzalo/clicktui/internal/clickup"
)

// staticProvider is a test Provider that sets a fixed Authorization header.
type staticProvider struct{ token string }

func (p *staticProvider) Method() auth.Method { return auth.MethodPersonalToken }
func (p *staticProvider) Authorize(_ context.Context, r *http.Request) error {
	r.Header.Set("Authorization", p.token)
	return nil
}

// newTestClient returns a Client pointed at srv with a static token.
func newTestClient(t *testing.T, token string, srv *httptest.Server) *clickup.Client {
	t.Helper()
	c := clickup.New(&staticProvider{token: token})
	c.SetBaseURL(srv.URL) // test hook — see client.go
	return c
}

// errorProvider is a test Provider that always returns an error from Authorize.
type errorProvider struct{}

func (p *errorProvider) Method() auth.Method { return auth.MethodPersonalToken }
func (p *errorProvider) Authorize(_ context.Context, _ *http.Request) error {
	return errors.New("credential not found")
}

func TestAPIError_Error(t *testing.T) {
	err := &clickup.APIError{StatusCode: 429, Body: `{"err":"Rate limit exceeded"}`}
	assert.Contains(t, err.Error(), "429")
	assert.Contains(t, err.Error(), "Rate limit exceeded")
}

func TestClient_AuthorizeFailure(t *testing.T) {
	// When the Provider.Authorize returns an error the client must propagate it
	// without making any HTTP call.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// This handler must never be reached.
		t.Error("unexpected HTTP request")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := clickup.New(&errorProvider{})
	c.SetBaseURL(srv.URL)

	_, err := c.AuthorizedUser(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "authorize request")
}

func TestClient_MalformedJSONResponse(t *testing.T) {
	// A 200 response with invalid JSON must return a decode error.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{broken`))
	}))
	defer srv.Close()

	client := newTestClient(t, "pk_test", srv)
	_, err := client.AuthorizedUser(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode response")
}

func TestAuthorizedUser_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "pk_test", r.Header.Get("Authorization"))
		assert.Equal(t, "/user", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"user": map[string]any{
				"id":       1,
				"username": "alice",
				"email":    "alice@example.com",
			},
		})
	}))
	defer srv.Close()

	client := newTestClient(t, "pk_test", srv)
	user, err := client.AuthorizedUser(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "alice", user.Username)
	assert.Equal(t, "alice@example.com", user.Email)
}

func TestAuthorizedUser_ErrorStatuses(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		token  string
	}{
		{name: "unauthorized", status: http.StatusUnauthorized, body: `{"err":"Token invalid"}`, token: "bad_token"},
		{name: "forbidden", status: http.StatusForbidden, body: `{"err":"Forbidden"}`, token: "pk_test"},
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
			_, err := client.AuthorizedUser(context.Background())
			require.Error(t, err)

			var apiErr *clickup.APIError
			require.ErrorAs(t, err, &apiErr)
			assert.Equal(t, tc.status, apiErr.StatusCode)
		})
	}
}

func TestTeams_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/team", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"teams": []map[string]any{
				{"id": "t1", "name": "My Workspace"},
			},
		})
	}))
	defer srv.Close()

	client := newTestClient(t, "pk_test", srv)
	teams, err := client.Teams(context.Background())
	require.NoError(t, err)
	require.Len(t, teams, 1)
	assert.Equal(t, "My Workspace", teams[0].Name)
}

func TestTeams_ErrorStatuses(t *testing.T) {
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
			_, err := client.Teams(context.Background())
			require.Error(t, err)

			var apiErr *clickup.APIError
			require.ErrorAs(t, err, &apiErr)
			assert.Equal(t, tc.status, apiErr.StatusCode)
		})
	}
}

func TestListStatuses_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/list/l1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":   "l1",
			"name": "Sprint 42",
			"statuses": []map[string]any{
				{"status": "open", "color": "#d3d3d3", "type": "open"},
				{"status": "in progress", "color": "#4169e1", "type": "custom"},
				{"status": "done", "color": "#00ff00", "type": "closed"},
			},
		})
	}))
	defer srv.Close()

	client := newTestClient(t, "pk_test", srv)
	statuses, err := client.ListStatuses(context.Background(), "l1")
	require.NoError(t, err)
	require.Len(t, statuses, 3)
	assert.Equal(t, "open", statuses[0].Status)
	assert.Equal(t, "#d3d3d3", statuses[0].Color)
	assert.Equal(t, "in progress", statuses[1].Status)
	assert.Equal(t, "done", statuses[2].Status)
}

func TestListStatuses_ErrorStatuses(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		listID string
	}{
		{name: "not_found", status: http.StatusNotFound, body: `{"err":"List not found"}`, listID: "missing"},
		{name: "unauthorized", status: http.StatusUnauthorized, body: `{"err":"Token invalid"}`, listID: "l1"},
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
			_, err := client.ListStatuses(context.Background(), tc.listID)
			require.Error(t, err)

			var apiErr *clickup.APIError
			require.ErrorAs(t, err, &apiErr)
			assert.Equal(t, tc.status, apiErr.StatusCode)
		})
	}
}

func TestUpdateTaskStatus_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/task/t1", r.URL.Path)
		assert.Equal(t, http.MethodPut, r.Method)

		var body map[string]any
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "in progress", body["status"])

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":   "t1",
			"name": "Fix login",
			"status": map[string]any{
				"status": "in progress",
				"color":  "#4169e1",
				"type":   "custom",
			},
		})
	}))
	defer srv.Close()

	client := newTestClient(t, "pk_test", srv)
	task, err := client.UpdateTaskStatus(context.Background(), "t1", "in progress")
	require.NoError(t, err)
	require.NotNil(t, task)
	assert.Equal(t, "t1", task.ID)
	assert.Equal(t, "in progress", task.Status.Status)
}

func TestUpdateTaskStatus_ErrorStatuses(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		token  string
		value  string
	}{
		{name: "bad_request", status: http.StatusBadRequest, body: `{"err":"Invalid status value"}`, token: "pk_test", value: "not_a_real_status"},
		{name: "unauthorized", status: http.StatusUnauthorized, body: `{"err":"Token invalid"}`, token: "bad_token", value: "done"},
		{name: "forbidden", status: http.StatusForbidden, body: `{"err":"Forbidden"}`, token: "pk_test", value: "done"},
		{name: "rate_limit", status: http.StatusTooManyRequests, body: `{"err":"Rate limit exceeded"}`, token: "pk_test", value: "done"},
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
			_, err := client.UpdateTaskStatus(context.Background(), "t1", tc.value)
			require.Error(t, err)

			var apiErr *clickup.APIError
			require.ErrorAs(t, err, &apiErr)
			assert.Equal(t, tc.status, apiErr.StatusCode)
		})
	}
}

func TestMoveTaskToList_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v3/workspaces/w1/tasks/t1/home_list/l2", r.URL.Path)
		assert.Equal(t, http.MethodPut, r.Method)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":   "t1",
			"name": "Fix login",
			"list": map[string]any{
				"id":   "l2",
				"name": "In Progress",
			},
			"status": map[string]any{
				"status": "open",
				"color":  "#d3d3d3",
				"type":   "open",
			},
		})
	}))
	defer srv.Close()

	client := newTestClient(t, "pk_test", srv)
	task, err := client.MoveTaskToList(context.Background(), "w1", "t1", "l2")
	require.NoError(t, err)
	require.NotNil(t, task)
	assert.Equal(t, "t1", task.ID)
	assert.Equal(t, "l2", task.List.ID)
}

func TestMoveTaskToList_InvalidList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"err":"Invalid list_id value"}`))
	}))
	defer srv.Close()

	client := newTestClient(t, "pk_test", srv)
	_, err := client.MoveTaskToList(context.Background(), "w1", "t1", "missing")
	require.Error(t, err)

	var apiErr *clickup.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 400, apiErr.StatusCode)
}

func TestTasks_SendsSubtasksParam(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/list/l1/task", r.URL.Path)
		assert.Equal(t, "true", r.URL.Query().Get("subtasks"))
		assert.Equal(t, "0", r.URL.Query().Get("page"))

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"tasks": []map[string]any{
				{"id": "t1", "name": "Parent task", "parent": ""},
				{"id": "t2", "name": "Child task", "parent": "t1"},
			},
		})
	}))
	defer srv.Close()

	client := newTestClient(t, "pk_test", srv)
	tasks, err := client.Tasks(context.Background(), "l1", 0)
	require.NoError(t, err)
	require.Len(t, tasks, 2)
	assert.Equal(t, "t1", tasks[0].ID)
	assert.Equal(t, "", tasks[0].Parent)
	assert.Equal(t, "t2", tasks[1].ID)
	assert.Equal(t, "t1", tasks[1].Parent)
}

func TestTask_SendsIncludeSubtasksParam(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/task/t1", r.URL.Path)
		assert.Equal(t, "true", r.URL.Query().Get("include_subtasks"))

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":   "t1",
			"name": "Parent task",
			"subtasks": []map[string]any{
				{"id": "t2", "name": "Child A", "parent": "t1"},
				{"id": "t3", "name": "Child B", "parent": "t1"},
			},
		})
	}))
	defer srv.Close()

	client := newTestClient(t, "pk_test", srv)
	task, err := client.Task(context.Background(), "t1")
	require.NoError(t, err)
	require.NotNil(t, task)
	assert.Equal(t, "t1", task.ID)
	assert.Equal(t, "Parent task", task.Name)
	require.Len(t, task.Subtasks, 2)
	assert.Equal(t, "t2", task.Subtasks[0].ID)
	assert.Equal(t, "Child A", task.Subtasks[0].Name)
	assert.Equal(t, "t1", task.Subtasks[0].Parent)
	assert.Equal(t, "t3", task.Subtasks[1].ID)
	assert.Equal(t, "Child B", task.Subtasks[1].Name)
}

func TestTask_SubtasksEmptyWhenAbsent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "true", r.URL.Query().Get("include_subtasks"))

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":   "t1",
			"name": "Leaf task",
		})
	}))
	defer srv.Close()

	client := newTestClient(t, "pk_test", srv)
	task, err := client.Task(context.Background(), "t1")
	require.NoError(t, err)
	require.NotNil(t, task)
	assert.Empty(t, task.Subtasks)
}

func TestUpdateTask_Success(t *testing.T) {
	name := "Updated name"
	desc := "Updated description"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/task/t1", r.URL.Path)
		assert.Equal(t, http.MethodPut, r.Method)

		var body map[string]any
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "Updated name", body["name"])
		assert.Equal(t, "Updated description", body["description"])

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":          "t1",
			"name":        "Updated name",
			"description": "Updated description",
			"status": map[string]any{
				"status": "open",
				"color":  "#d3d3d3",
				"type":   "open",
			},
		})
	}))
	defer srv.Close()

	client := newTestClient(t, "pk_test", srv)
	task, err := client.UpdateTask(context.Background(), "t1", clickup.UpdateTaskRequest{
		Name:        &name,
		Description: &desc,
	})
	require.NoError(t, err)
	require.NotNil(t, task)
	assert.Equal(t, "t1", task.ID)
	assert.Equal(t, "Updated name", task.Name)
}

func TestUpdateTask_WithAssignees(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/task/t1", r.URL.Path)
		assert.Equal(t, http.MethodPut, r.Method)

		var body map[string]any
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assignees, ok := body["assignees"].(map[string]any)
		if !assert.True(t, ok, "assignees field should be an object") {
			return
		}
		add, ok := assignees["add"].([]any)
		if !assert.True(t, ok) {
			return
		}
		assert.Len(t, add, 1)
		assert.Equal(t, float64(42), add[0])

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":   "t1",
			"name": "Task",
		})
	}))
	defer srv.Close()

	client := newTestClient(t, "pk_test", srv)
	task, err := client.UpdateTask(context.Background(), "t1", clickup.UpdateTaskRequest{
		Assignees: &clickup.AssigneeUpdate{Add: []int{42}},
	})
	require.NoError(t, err)
	require.NotNil(t, task)
}

func TestUpdateTask_ErrorStatuses(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		token  string
		taskID string
	}{
		{name: "not_found", status: http.StatusNotFound, body: `{"err":"Task not found"}`, token: "pk_test", taskID: "missing"},
		{name: "unauthorized", status: http.StatusUnauthorized, body: `{"err":"Token invalid"}`, token: "bad_token", taskID: "t1"},
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
			_, err := client.UpdateTask(context.Background(), tc.taskID, clickup.UpdateTaskRequest{})
			require.Error(t, err)

			var apiErr *clickup.APIError
			require.ErrorAs(t, err, &apiErr)
			assert.Equal(t, tc.status, apiErr.StatusCode)
		})
	}
}

func TestCreateTask_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/list/l1/task", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		var body map[string]any
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "New task", body["name"])
		assert.Equal(t, "open", body["status"])
		assert.Equal(t, "Some description", body["description"])

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":          "new-t1",
			"name":        "New task",
			"description": "Some description",
			"status": map[string]any{
				"status": "open",
				"color":  "#d3d3d3",
				"type":   "open",
			},
		})
	}))
	defer srv.Close()

	client := newTestClient(t, "pk_test", srv)
	task, err := client.CreateTask(context.Background(), "l1", clickup.CreateTaskRequest{
		Name:        "New task",
		Status:      "open",
		Description: "Some description",
	})
	require.NoError(t, err)
	require.NotNil(t, task)
	assert.Equal(t, "new-t1", task.ID)
	assert.Equal(t, "New task", task.Name)
}

func TestCreateTask_WithAssigneesAndPriority(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/list/l1/task", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		var body map[string]any
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, float64(2), body["priority"])
		assignees, ok := body["assignees"].([]any)
		if !assert.True(t, ok) {
			return
		}
		assert.Len(t, assignees, 2)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":   "new-t2",
			"name": "Assigned task",
		})
	}))
	defer srv.Close()

	client := newTestClient(t, "pk_test", srv)
	task, err := client.CreateTask(context.Background(), "l1", clickup.CreateTaskRequest{
		Name:      "Assigned task",
		Priority:  2,
		Assignees: []int{10, 20},
	})
	require.NoError(t, err)
	require.NotNil(t, task)
	assert.Equal(t, "new-t2", task.ID)
}

func TestCreateTask_ErrorStatuses(t *testing.T) {
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
			_, err := client.CreateTask(context.Background(), "l1", clickup.CreateTaskRequest{Name: "Task"})
			require.Error(t, err)

			var apiErr *clickup.APIError
			require.ErrorAs(t, err, &apiErr)
			assert.Equal(t, tc.status, apiErr.StatusCode)
		})
	}
}

func TestListMembers_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/list/l1/member", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"members": []map[string]any{
				{
					"id":       1,
					"username": "alice",
					"email":    "alice@example.com",
					"initials": "A",
				},
				{
					"id":       2,
					"username": "bob",
					"email":    "bob@example.com",
					"initials": "B",
				},
			},
		})
	}))
	defer srv.Close()

	client := newTestClient(t, "pk_test", srv)
	members, err := client.ListMembers(context.Background(), "l1")
	require.NoError(t, err)
	require.Len(t, members, 2)
	assert.Equal(t, 1, members[0].ID)
	assert.Equal(t, "alice", members[0].Username)
	assert.Equal(t, "alice@example.com", members[0].Email)
	assert.Equal(t, "A", members[0].Initials)
	assert.Equal(t, 2, members[1].ID)
	assert.Equal(t, "bob", members[1].Username)
}

func TestListMembers_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/list/l1/member", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"members": []any{},
		})
	}))
	defer srv.Close()

	client := newTestClient(t, "pk_test", srv)
	members, err := client.ListMembers(context.Background(), "l1")
	require.NoError(t, err)
	assert.Empty(t, members)
}

func TestListMembers_ErrorStatuses(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		token  string
		listID string
	}{
		{name: "not_found", status: http.StatusNotFound, body: `{"err":"List not found"}`, token: "pk_test", listID: "missing"},
		{name: "unauthorized", status: http.StatusUnauthorized, body: `{"err":"Token invalid"}`, token: "bad_token", listID: "l1"},
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
			_, err := client.ListMembers(context.Background(), tc.listID)
			require.Error(t, err)

			var apiErr *clickup.APIError
			require.ErrorAs(t, err, &apiErr)
			assert.Equal(t, tc.status, apiErr.StatusCode)
		})
	}
}
