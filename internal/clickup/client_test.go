package clickup_test

import (
	"context"
	"encoding/json"
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
