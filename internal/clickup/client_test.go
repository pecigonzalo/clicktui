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

func TestAuthorizedUser_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"err":"Token invalid"}`))
	}))
	defer srv.Close()

	client := newTestClient(t, "bad_token", srv)
	_, err := client.AuthorizedUser(context.Background())
	require.Error(t, err)

	var apiErr *clickup.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 401, apiErr.StatusCode)
}

func TestAuthorizedUser_Forbidden(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"err":"Forbidden"}`))
	}))
	defer srv.Close()

	client := newTestClient(t, "pk_test", srv)
	_, err := client.AuthorizedUser(context.Background())
	require.Error(t, err)

	var apiErr *clickup.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 403, apiErr.StatusCode)
}

func TestAuthorizedUser_RateLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"err":"Rate limit exceeded"}`))
	}))
	defer srv.Close()

	client := newTestClient(t, "pk_test", srv)
	_, err := client.AuthorizedUser(context.Background())
	require.Error(t, err)

	var apiErr *clickup.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 429, apiErr.StatusCode)
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

func TestTeams_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"err":"Token invalid"}`))
	}))
	defer srv.Close()

	client := newTestClient(t, "bad_token", srv)
	_, err := client.Teams(context.Background())
	require.Error(t, err)

	var apiErr *clickup.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 401, apiErr.StatusCode)
}

func TestTeams_RateLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"err":"Rate limit exceeded"}`))
	}))
	defer srv.Close()

	client := newTestClient(t, "pk_test", srv)
	_, err := client.Teams(context.Background())
	require.Error(t, err)

	var apiErr *clickup.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 429, apiErr.StatusCode)
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

func TestListStatuses_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"err":"List not found"}`))
	}))
	defer srv.Close()

	client := newTestClient(t, "pk_test", srv)
	_, err := client.ListStatuses(context.Background(), "missing")
	require.Error(t, err)

	var apiErr *clickup.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 404, apiErr.StatusCode)
}

func TestListStatuses_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"err":"Token invalid"}`))
	}))
	defer srv.Close()

	client := newTestClient(t, "pk_test", srv)
	_, err := client.ListStatuses(context.Background(), "l1")
	require.Error(t, err)

	var apiErr *clickup.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 401, apiErr.StatusCode)
}

func TestListStatuses_RateLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"err":"Rate limit exceeded"}`))
	}))
	defer srv.Close()

	client := newTestClient(t, "pk_test", srv)
	_, err := client.ListStatuses(context.Background(), "l1")
	require.Error(t, err)

	var apiErr *clickup.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 429, apiErr.StatusCode)
}

func TestUpdateTaskStatus_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/task/t1", r.URL.Path)
		assert.Equal(t, http.MethodPut, r.Method)

		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
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

func TestUpdateTaskStatus_InvalidStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"err":"Invalid status value"}`))
	}))
	defer srv.Close()

	client := newTestClient(t, "pk_test", srv)
	_, err := client.UpdateTaskStatus(context.Background(), "t1", "not_a_real_status")
	require.Error(t, err)

	var apiErr *clickup.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 400, apiErr.StatusCode)
}

func TestUpdateTaskStatus_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"err":"Token invalid"}`))
	}))
	defer srv.Close()

	client := newTestClient(t, "bad_token", srv)
	_, err := client.UpdateTaskStatus(context.Background(), "t1", "done")
	require.Error(t, err)

	var apiErr *clickup.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 401, apiErr.StatusCode)
}

func TestUpdateTaskStatus_Forbidden(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"err":"Forbidden"}`))
	}))
	defer srv.Close()

	client := newTestClient(t, "pk_test", srv)
	_, err := client.UpdateTaskStatus(context.Background(), "t1", "done")
	require.Error(t, err)

	var apiErr *clickup.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 403, apiErr.StatusCode)
}

func TestUpdateTaskStatus_RateLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"err":"Rate limit exceeded"}`))
	}))
	defer srv.Close()

	client := newTestClient(t, "pk_test", srv)
	_, err := client.UpdateTaskStatus(context.Background(), "t1", "done")
	require.Error(t, err)

	var apiErr *clickup.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 429, apiErr.StatusCode)
}
