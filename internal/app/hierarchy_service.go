// Package app provides business-logic services that coordinate ClickUp API
// calls and transform data for presentation in the TUI.
package app

import (
	"context"
	"fmt"

	"github.com/pecigonzalo/clicktui/internal/clickup"
)

// ClickUpAPI is the subset of the ClickUp client used by services.
// Accepting an interface rather than a concrete *clickup.Client keeps
// the services testable without HTTP.
type ClickUpAPI interface {
	Teams(ctx context.Context) ([]clickup.Team, error)
	Spaces(ctx context.Context, teamID string) ([]clickup.Space, error)
	Folders(ctx context.Context, spaceID string) ([]clickup.Folder, error)
	FolderlessLists(ctx context.Context, spaceID string) ([]clickup.List, error)
	Tasks(ctx context.Context, listID string, page int) ([]clickup.Task, error)
	Task(ctx context.Context, taskID string) (*clickup.Task, error)
	ListStatuses(ctx context.Context, listID string) ([]clickup.Status, error)
	UpdateTaskStatus(ctx context.Context, taskID, status string) (*clickup.Task, error)
	MoveTaskToList(ctx context.Context, workspaceID, taskID, listID string) (*clickup.Task, error)
}

// HierarchyNode represents a single node in the workspace tree.
type HierarchyNode struct {
	// ID is the ClickUp entity ID.
	ID string
	// Name is the display name.
	Name string
	// Kind describes the node type.
	Kind NodeKind
	// Children are lazily loaded.
	Children []*HierarchyNode
	// Loaded indicates whether children have been fetched.
	Loaded bool
	// ParentID is a reference for contextual lookups.
	ParentID string
}

// NodeKind identifies the type of hierarchy entity.
type NodeKind int

const (
	NodeWorkspace NodeKind = iota + 1
	NodeSpace
	NodeFolder
	NodeList
)

// String returns a human-readable label for the node kind.
func (k NodeKind) String() string {
	switch k {
	case NodeWorkspace:
		return "Workspace"
	case NodeSpace:
		return "Space"
	case NodeFolder:
		return "Folder"
	case NodeList:
		return "List"
	default:
		return "Unknown"
	}
}

// HierarchyService loads and caches the ClickUp workspace hierarchy.
type HierarchyService struct {
	api ClickUpAPI
}

// NewHierarchyService creates a HierarchyService backed by the given API.
func NewHierarchyService(api ClickUpAPI) *HierarchyService {
	return &HierarchyService{api: api}
}

// LoadWorkspaces returns the top-level workspace nodes.
func (s *HierarchyService) LoadWorkspaces(ctx context.Context) ([]*HierarchyNode, error) {
	teams, err := s.api.Teams(ctx)
	if err != nil {
		return nil, fmt.Errorf("load workspaces: %w", err)
	}
	nodes := make([]*HierarchyNode, len(teams))
	for i, t := range teams {
		nodes[i] = &HierarchyNode{ID: t.ID, Name: t.Name, Kind: NodeWorkspace}
	}
	return nodes, nil
}

// LoadSpaces returns space nodes for a workspace.
func (s *HierarchyService) LoadSpaces(ctx context.Context, teamID string) ([]*HierarchyNode, error) {
	spaces, err := s.api.Spaces(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("load spaces: %w", err)
	}
	nodes := make([]*HierarchyNode, len(spaces))
	for i, sp := range spaces {
		nodes[i] = &HierarchyNode{
			ID:       sp.ID,
			Name:     sp.Name,
			Kind:     NodeSpace,
			ParentID: teamID,
		}
	}
	return nodes, nil
}

// LoadSpaceContents returns folders and folderless lists for a space,
// combining them into a single ordered list (folders first, then folderless lists).
func (s *HierarchyService) LoadSpaceContents(ctx context.Context, spaceID string) ([]*HierarchyNode, error) {
	folders, err := s.api.Folders(ctx, spaceID)
	if err != nil {
		return nil, fmt.Errorf("load folders: %w", err)
	}

	folderlessLists, err := s.api.FolderlessLists(ctx, spaceID)
	if err != nil {
		return nil, fmt.Errorf("load folderless lists: %w", err)
	}

	var nodes []*HierarchyNode

	for _, f := range folders {
		fNode := &HierarchyNode{
			ID:       f.ID,
			Name:     f.Name,
			Kind:     NodeFolder,
			ParentID: spaceID,
			Loaded:   true,
		}
		for _, l := range f.Lists {
			fNode.Children = append(fNode.Children, &HierarchyNode{
				ID:       l.ID,
				Name:     l.Name,
				Kind:     NodeList,
				ParentID: f.ID,
				Loaded:   true,
			})
		}
		nodes = append(nodes, fNode)
	}

	for _, l := range folderlessLists {
		nodes = append(nodes, &HierarchyNode{
			ID:       l.ID,
			Name:     l.Name,
			Kind:     NodeList,
			ParentID: spaceID,
			Loaded:   true,
		})
	}

	return nodes, nil
}
