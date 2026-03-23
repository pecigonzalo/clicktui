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

// fakeAPI is an in-memory stub of the ClickUpAPI interface for testing.
type fakeAPI struct {
	teams         []clickup.Team
	spaces        map[string][]clickup.Space
	folders       map[string][]clickup.Folder
	folderless    map[string][]clickup.List
	tasks         map[string][]clickup.Task
	tasksByID     map[string]*clickup.Task
	teamsErr      error
	spacesErr     error
	foldersErr    error
	folderlessErr error
	tasksErr      error
	taskErr       error
}

func (f *fakeAPI) Teams(_ context.Context) ([]clickup.Team, error) {
	return f.teams, f.teamsErr
}

func (f *fakeAPI) Spaces(_ context.Context, teamID string) ([]clickup.Space, error) {
	return f.spaces[teamID], f.spacesErr
}

func (f *fakeAPI) Folders(_ context.Context, spaceID string) ([]clickup.Folder, error) {
	return f.folders[spaceID], f.foldersErr
}

func (f *fakeAPI) FolderlessLists(_ context.Context, spaceID string) ([]clickup.List, error) {
	return f.folderless[spaceID], f.folderlessErr
}

func (f *fakeAPI) Tasks(_ context.Context, listID string, _ int) ([]clickup.Task, error) {
	return f.tasks[listID], f.tasksErr
}

func (f *fakeAPI) Task(_ context.Context, taskID string) (*clickup.Task, error) {
	t, ok := f.tasksByID[taskID]
	if !ok {
		return nil, f.taskErr
	}
	return t, f.taskErr
}

func newFakeAPI() *fakeAPI {
	return &fakeAPI{
		spaces:     make(map[string][]clickup.Space),
		folders:    make(map[string][]clickup.Folder),
		folderless: make(map[string][]clickup.List),
		tasks:      make(map[string][]clickup.Task),
		tasksByID:  make(map[string]*clickup.Task),
	}
}

func TestHierarchyService_LoadWorkspaces(t *testing.T) {
	api := newFakeAPI()
	api.teams = []clickup.Team{
		{ID: "t1", Name: "Acme"},
		{ID: "t2", Name: "Beta"},
	}

	svc := app.NewHierarchyService(api)
	nodes, err := svc.LoadWorkspaces(context.Background())
	require.NoError(t, err)
	require.Len(t, nodes, 2)

	assert.Equal(t, "t1", nodes[0].ID)
	assert.Equal(t, "Acme", nodes[0].Name)
	assert.Equal(t, app.NodeWorkspace, nodes[0].Kind)
}

func TestHierarchyService_LoadWorkspaces_Error(t *testing.T) {
	api := newFakeAPI()
	api.teamsErr = errors.New("network failure")

	svc := app.NewHierarchyService(api)
	_, err := svc.LoadWorkspaces(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "load workspaces")
}

func TestHierarchyService_LoadSpaces(t *testing.T) {
	api := newFakeAPI()
	api.spaces["t1"] = []clickup.Space{
		{ID: "s1", Name: "Engineering"},
	}

	svc := app.NewHierarchyService(api)
	nodes, err := svc.LoadSpaces(context.Background(), "t1")
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Equal(t, app.NodeSpace, nodes[0].Kind)
	assert.Equal(t, "t1", nodes[0].ParentID)
}

func TestHierarchyService_LoadSpaceContents_FoldersAndFolderlessLists(t *testing.T) {
	api := newFakeAPI()
	api.folders["s1"] = []clickup.Folder{
		{
			ID:   "f1",
			Name: "Backend",
			Lists: []clickup.List{
				{ID: "l1", Name: "Sprint 42"},
				{ID: "l2", Name: "Sprint 43"},
			},
		},
	}
	api.folderless["s1"] = []clickup.List{
		{ID: "l3", Name: "Backlog"},
	}

	svc := app.NewHierarchyService(api)
	nodes, err := svc.LoadSpaceContents(context.Background(), "s1")
	require.NoError(t, err)

	// Should have 1 folder + 1 folderless list = 2 top-level nodes.
	require.Len(t, nodes, 2)

	// First node is the folder with its lists as children.
	assert.Equal(t, app.NodeFolder, nodes[0].Kind)
	assert.Equal(t, "Backend", nodes[0].Name)
	require.Len(t, nodes[0].Children, 2)
	assert.Equal(t, app.NodeList, nodes[0].Children[0].Kind)
	assert.Equal(t, "Sprint 42", nodes[0].Children[0].Name)
	assert.True(t, nodes[0].Loaded)

	// Second node is the folderless list.
	assert.Equal(t, app.NodeList, nodes[1].Kind)
	assert.Equal(t, "Backlog", nodes[1].Name)
	assert.Equal(t, "s1", nodes[1].ParentID)
}

func TestHierarchyService_LoadSpaceContents_OnlyFolderless(t *testing.T) {
	api := newFakeAPI()
	api.folders["s1"] = nil
	api.folderless["s1"] = []clickup.List{
		{ID: "l1", Name: "Standalone"},
	}

	svc := app.NewHierarchyService(api)
	nodes, err := svc.LoadSpaceContents(context.Background(), "s1")
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	assert.Equal(t, app.NodeList, nodes[0].Kind)
}

func TestHierarchyService_LoadSpaceContents_Empty(t *testing.T) {
	api := newFakeAPI()

	svc := app.NewHierarchyService(api)
	nodes, err := svc.LoadSpaceContents(context.Background(), "s1")
	require.NoError(t, err)
	assert.Empty(t, nodes)
}
