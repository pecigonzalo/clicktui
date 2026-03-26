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
	teams            []clickup.Team
	spaces           map[string][]clickup.Space
	folders          map[string][]clickup.Folder
	folderless       map[string][]clickup.List
	tasks            map[string][]clickup.Task
	tasksByID        map[string]*clickup.Task
	listsByID        map[string]*clickup.List
	statusesByListID map[string][]clickup.Status
	membersByListID  map[string][]clickup.Member
	updatedTasks     map[string]*clickup.Task // taskID -> result of UpdateTaskStatus
	movedTasks       map[string]*clickup.Task // taskID -> result of MoveTaskToList
	createdTask      *clickup.Task            // result returned by CreateTask
	teamsErr         error
	spacesErr        error
	foldersErr       error
	folderlessErr    error
	tasksErr         error
	taskErr          error
	getListErr       error
	listStatusesErr  error
	updateStatusErr  error
	moveTaskErr      error
	updateTaskErr    error
	createTaskErr    error
	listMembersErr   error
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

func (f *fakeAPI) GetList(_ context.Context, listID string) (*clickup.List, error) {
	if f.getListErr != nil {
		return nil, f.getListErr
	}
	if l, ok := f.listsByID[listID]; ok {
		return l, nil
	}
	return nil, errors.New("list not found")
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

func (f *fakeAPI) ListStatuses(_ context.Context, listID string) ([]clickup.Status, error) {
	return f.statusesByListID[listID], f.listStatusesErr
}

func (f *fakeAPI) UpdateTaskStatus(_ context.Context, taskID, status string) (*clickup.Task, error) {
	if f.updateStatusErr != nil {
		return nil, f.updateStatusErr
	}
	if t, ok := f.updatedTasks[taskID]; ok {
		return t, nil
	}
	// Mirror the existing task with the new status applied.
	if t, ok := f.tasksByID[taskID]; ok {
		updated := *t
		updated.Status.Status = status
		return &updated, nil
	}
	return &clickup.Task{ID: taskID, Status: clickup.Status{Status: status}}, nil
}

func (f *fakeAPI) MoveTaskToList(_ context.Context, _ string, taskID, listID string) (*clickup.Task, error) {
	if f.moveTaskErr != nil {
		return nil, f.moveTaskErr
	}
	if t, ok := f.movedTasks[taskID]; ok {
		tt := *t
		f.tasksByID[taskID] = &tt
		return t, nil
	}
	if t, ok := f.tasksByID[taskID]; ok {
		moved := *t
		moved.List.ID = listID
		f.tasksByID[taskID] = &moved
		return &moved, nil
	}
	moved := &clickup.Task{ID: taskID, List: clickup.TaskRef{ID: listID}}
	f.tasksByID[taskID] = moved
	return moved, nil
}

func (f *fakeAPI) UpdateTask(_ context.Context, taskID string, _ clickup.UpdateTaskRequest) (*clickup.Task, error) {
	if f.updateTaskErr != nil {
		return nil, f.updateTaskErr
	}
	if t, ok := f.tasksByID[taskID]; ok {
		cp := *t
		return &cp, nil
	}
	return &clickup.Task{ID: taskID}, nil
}

func (f *fakeAPI) CreateTask(_ context.Context, _ string, req clickup.CreateTaskRequest) (*clickup.Task, error) {
	if f.createTaskErr != nil {
		return nil, f.createTaskErr
	}
	if f.createdTask != nil {
		return f.createdTask, nil
	}
	return &clickup.Task{ID: "new-task", Name: req.Name}, nil
}

func (f *fakeAPI) ListMembers(_ context.Context, listID string) ([]clickup.Member, error) {
	return f.membersByListID[listID], f.listMembersErr
}

func newFakeAPI() *fakeAPI {
	return &fakeAPI{
		spaces:           make(map[string][]clickup.Space),
		folders:          make(map[string][]clickup.Folder),
		folderless:       make(map[string][]clickup.List),
		tasks:            make(map[string][]clickup.Task),
		tasksByID:        make(map[string]*clickup.Task),
		listsByID:        make(map[string]*clickup.List),
		statusesByListID: make(map[string][]clickup.Status),
		membersByListID:  make(map[string][]clickup.Member),
		updatedTasks:     make(map[string]*clickup.Task),
		movedTasks:       make(map[string]*clickup.Task),
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

func TestHierarchyService_LoadSpaces_Error(t *testing.T) {
	api := newFakeAPI()
	api.spacesErr = errors.New("api failure")

	svc := app.NewHierarchyService(api)
	_, err := svc.LoadSpaces(context.Background(), "t1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "load spaces")
}

func TestHierarchyService_LoadSpaceContents_FoldersError(t *testing.T) {
	api := newFakeAPI()
	api.foldersErr = errors.New("forbidden")

	svc := app.NewHierarchyService(api)
	_, err := svc.LoadSpaceContents(context.Background(), "s1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "load folders")
}

func TestHierarchyService_LoadSpaceContents_FolderlessError(t *testing.T) {
	api := newFakeAPI()
	// Folders succeed but folderless lists fail.
	api.folders["s1"] = []clickup.Folder{}
	api.folderlessErr = errors.New("rate limited")

	svc := app.NewHierarchyService(api)
	_, err := svc.LoadSpaceContents(context.Background(), "s1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "load folderless lists")
}
