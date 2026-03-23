package app_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pecigonzalo/clicktui/internal/app"
)

// buildTestTree creates a hierarchy for testing:
//
//	Workspace "Acme"
//	├── Space "Engineering"
//	│   ├── Folder "Backend"
//	│   │   ├── List "API Tasks"
//	│   │   └── List "Auth Module"
//	│   └── List "Frontend"
//	└── Space "Marketing"
//	    └── List "Campaigns"
func buildTestTree() []*app.HierarchyNode {
	return []*app.HierarchyNode{
		{
			ID: "w1", Name: "Acme", Kind: app.NodeWorkspace,
			Children: []*app.HierarchyNode{
				{
					ID: "s1", Name: "Engineering", Kind: app.NodeSpace,
					Children: []*app.HierarchyNode{
						{
							ID: "f1", Name: "Backend", Kind: app.NodeFolder,
							Children: []*app.HierarchyNode{
								{ID: "l1", Name: "API Tasks", Kind: app.NodeList},
								{ID: "l2", Name: "Auth Module", Kind: app.NodeList},
							},
						},
						{ID: "l3", Name: "Frontend", Kind: app.NodeList},
					},
				},
				{
					ID: "s2", Name: "Marketing", Kind: app.NodeSpace,
					Children: []*app.HierarchyNode{
						{ID: "l4", Name: "Campaigns", Kind: app.NodeList},
					},
				},
			},
		},
	}
}

func TestFilterHierarchy_EmptyQuery(t *testing.T) {
	tree := buildTestTree()
	result := app.FilterHierarchy(tree, "")
	assert.Nil(t, result, "empty query should return nil (show all)")
}

func TestFilterHierarchy_NilNodes(t *testing.T) {
	result := app.FilterHierarchy(nil, "auth")
	assert.Nil(t, result)
}

func TestFilterHierarchy_MatchLeafNode(t *testing.T) {
	tree := buildTestTree()
	result := app.FilterHierarchy(tree, "Auth Module")
	require.NotNil(t, result)

	// Should retain ancestors: Acme → Engineering → Backend → Auth Module
	require.Len(t, result, 1, "workspace level")
	assert.Equal(t, "Acme", result[0].Name)

	require.Len(t, result[0].Children, 1, "space level")
	assert.Equal(t, "Engineering", result[0].Children[0].Name)

	require.Len(t, result[0].Children[0].Children, 1, "folder level")
	assert.Equal(t, "Backend", result[0].Children[0].Children[0].Name)

	// The folder should contain the matching list.
	backend := result[0].Children[0].Children[0]
	found := false
	for _, child := range backend.Children {
		if child.Name == "Auth Module" {
			found = true
			break
		}
	}
	assert.True(t, found, "should find Auth Module in Backend children")
}

func TestFilterHierarchy_MatchMiddleNode(t *testing.T) {
	tree := buildTestTree()
	result := app.FilterHierarchy(tree, "Backend")
	require.NotNil(t, result)

	// Should retain: Acme → Engineering → Backend (with all its children).
	require.Len(t, result, 1)
	assert.Equal(t, "Acme", result[0].Name)

	eng := result[0].Children[0]
	assert.Equal(t, "Engineering", eng.Name)

	// Backend should still have its children (deep copied).
	backend := eng.Children[0]
	assert.Equal(t, "Backend", backend.Name)
	assert.Len(t, backend.Children, 2, "matching node should keep all children")
}

func TestFilterHierarchy_MatchRootNode(t *testing.T) {
	tree := buildTestTree()
	result := app.FilterHierarchy(tree, "Acme")
	require.NotNil(t, result)
	require.Len(t, result, 1)
	assert.Equal(t, "Acme", result[0].Name)
	// The entire subtree should be preserved.
	assert.Len(t, result[0].Children, 2, "root match should keep all children")
}

func TestFilterHierarchy_NoMatches(t *testing.T) {
	tree := buildTestTree()
	result := app.FilterHierarchy(tree, "zzzzzzz")
	assert.Nil(t, result)
}

func TestFilterHierarchy_MultipleMatches(t *testing.T) {
	tree := buildTestTree()
	// "a" should fuzzy-match many nodes.
	result := app.FilterHierarchy(tree, "Campaign")
	require.NotNil(t, result)

	// Should include Acme → Marketing → Campaigns.
	var found bool
	for _, ws := range result {
		for _, space := range ws.Children {
			if space.Name == "Marketing" {
				for _, list := range space.Children {
					if list.Name == "Campaigns" {
						found = true
					}
				}
			}
		}
	}
	assert.True(t, found, "should find Campaigns via Marketing path")
}

func TestFilterHierarchy_DoesNotMutateOriginal(t *testing.T) {
	tree := buildTestTree()
	originalChildCount := len(tree[0].Children)

	_ = app.FilterHierarchy(tree, "Auth Module")

	assert.Len(t, tree[0].Children, originalChildCount,
		"original tree should not be mutated")
}

func TestFilterHierarchy_AncestorRetention(t *testing.T) {
	// Match only "Frontend" (a direct child of Engineering space, not in a folder).
	tree := buildTestTree()
	result := app.FilterHierarchy(tree, "Frontend")
	require.NotNil(t, result)

	// Path: Acme → Engineering → Frontend
	require.Len(t, result, 1)
	ws := result[0]
	assert.Equal(t, "Acme", ws.Name)
	require.Len(t, ws.Children, 1)
	assert.Equal(t, "Engineering", ws.Children[0].Name)

	// Engineering should contain Frontend (and possibly Backend if it also matches).
	eng := ws.Children[0]
	var hasFrontend bool
	for _, child := range eng.Children {
		if child.Name == "Frontend" {
			hasFrontend = true
		}
	}
	assert.True(t, hasFrontend, "should retain Frontend under Engineering")
}

func TestFilterHierarchy_EmptyNodes(t *testing.T) {
	result := app.FilterHierarchy([]*app.HierarchyNode{}, "test")
	assert.Nil(t, result)
}
