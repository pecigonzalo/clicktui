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

func TestHierarchyService_LoadAllLists_FlattensAcrossWorkspacesAndSpaces(t *testing.T) {
	t.Parallel()

	api := &fakeAPI{
		teams: []clickup.Team{{ID: "w1", Name: "Workspace One"}},
		spaces: map[string][]clickup.Space{
			"w1": {{ID: "s1", Name: "Space One"}},
		},
		folders: map[string][]clickup.Folder{
			"s1": {{ID: "f1", Name: "Folder One", Lists: []clickup.List{
				{ID: "l1", Name: "List In Folder"},
			}}},
		},
		folderless: map[string][]clickup.List{
			"s1": {{ID: "l2", Name: "Folderless List"}},
		},
	}
	svc := app.NewHierarchyService(api)

	refs, err := svc.LoadAllLists(context.Background())
	require.NoError(t, err)
	require.Len(t, refs, 2)

	assert.Equal(t, "l1", refs[0].ID)
	assert.Equal(t, "List In Folder", refs[0].Name)
	assert.Equal(t, "w1", refs[0].WorkspaceID)
	assert.Equal(t, "Workspace One", refs[0].WorkspaceName)
	assert.Equal(t, "Space One", refs[0].SpaceName)
	assert.Equal(t, "Folder One", refs[0].FolderName)

	assert.Equal(t, "l2", refs[1].ID)
	assert.Equal(t, "Folderless List", refs[1].Name)
	assert.Empty(t, refs[1].FolderName)
}

func TestHierarchyService_LoadAllLists_CachesAfterFirstLoad(t *testing.T) {
	t.Parallel()

	calls := 0
	api := &fakeAPI{
		teams: []clickup.Team{{ID: "w1", Name: "W"}},
		spaces: map[string][]clickup.Space{
			"w1": {{ID: "s1", Name: "S"}},
		},
	}
	svc := app.NewHierarchyService(&teamsCountingAPI{fakeAPI: api, onTeams: func() { calls++ }})

	_, err := svc.LoadAllLists(context.Background())
	require.NoError(t, err)
	_, err = svc.LoadAllLists(context.Background())
	require.NoError(t, err)

	assert.Equal(t, 1, calls, "Teams should only be called once — second load must hit the cache")
}

func TestHierarchyService_LoadAllLists_InvalidateForcesRefresh(t *testing.T) {
	t.Parallel()

	calls := 0
	api := &fakeAPI{teams: []clickup.Team{{ID: "w1", Name: "W"}}}
	svc := app.NewHierarchyService(&teamsCountingAPI{fakeAPI: api, onTeams: func() { calls++ }})

	_, err := svc.LoadAllLists(context.Background())
	require.NoError(t, err)
	svc.InvalidateAllLists()
	_, err = svc.LoadAllLists(context.Background())
	require.NoError(t, err)

	assert.Equal(t, 2, calls)
}

func TestHierarchyService_LoadAllLists_TeamsError(t *testing.T) {
	t.Parallel()

	api := &fakeAPI{teamsErr: errors.New("boom")}
	svc := app.NewHierarchyService(api)

	_, err := svc.LoadAllLists(context.Background())
	require.Error(t, err)
}

func TestHierarchyService_LoadAllLists_SpacesError(t *testing.T) {
	t.Parallel()

	api := &fakeAPI{
		teams:     []clickup.Team{{ID: "w1", Name: "W"}},
		spacesErr: errors.New("boom"),
	}
	svc := app.NewHierarchyService(api)

	_, err := svc.LoadAllLists(context.Background())
	require.Error(t, err)
}

func TestHierarchyService_LoadAllLists_EmptyWorkspace(t *testing.T) {
	t.Parallel()

	svc := app.NewHierarchyService(&fakeAPI{})

	refs, err := svc.LoadAllLists(context.Background())
	require.NoError(t, err)
	assert.Empty(t, refs)
}

// teamsCountingAPI wraps fakeAPI to observe how many times Teams is called,
// letting cache-hit tests assert the underlying API was not re-queried.
type teamsCountingAPI struct {
	*fakeAPI
	onTeams func()
}

func (c *teamsCountingAPI) Teams(ctx context.Context) ([]clickup.Team, error) {
	if c.onTeams != nil {
		c.onTeams()
	}
	return c.fakeAPI.Teams(ctx)
}

func TestFilterListRefs_EmptyQueryReturnsAllUnchanged(t *testing.T) {
	t.Parallel()
	refs := []app.ListRef{{ID: "1", Name: "Alpha"}, {ID: "2", Name: "Beta"}}
	got := app.FilterListRefs(refs, "")
	assert.Equal(t, refs, got)
}

func TestFilterListRefs_MatchesByFuzzyName(t *testing.T) {
	t.Parallel()
	refs := []app.ListRef{
		{ID: "1", Name: "Engineering Sprint"},
		{ID: "2", Name: "Marketing Calendar"},
		{ID: "3", Name: "Eng Backlog"},
	}
	got := app.FilterListRefs(refs, "eng")
	require.NotEmpty(t, got)
	// Fuzzy matching allows non-contiguous subsequences, so "Marketing
	// Calendar" can still appear (e-n-g is a subsequence of "Marketing") —
	// what matters is that the strongest matches rank first.
	assert.Contains(t, []string{"1", "3"}, got[0].ID)
}

func TestFilterListRefs_NoMatchReturnsNil(t *testing.T) {
	t.Parallel()
	refs := []app.ListRef{{ID: "1", Name: "Alpha"}}
	got := app.FilterListRefs(refs, "zzzzz-nomatch")
	assert.Nil(t, got)
}
