// Package tui — unit tests for tree pane helper functions.
package tui

import (
	"strings"
	"testing"

	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pecigonzalo/clicktui/internal/app"
)

// ── helper: build a minimal TreePane for testing ─────────────────────────────

func newTestTreePane(t *testing.T) *TreePane {
	t.Helper()
	root := tview.NewTreeNode("root")
	return &TreePane{
		TreeView: tview.NewTreeView().SetRoot(root),
		root:     root,
	}
}

// buildTestTree creates a tree structure:
//
//	root
//	├── A
//	│   ├── A1
//	│   └── A2
//	├── B
//	│   ├── B1
//	│   │   └── B1a
//	│   └── B2
//	└── C
func buildTestTree(t *testing.T) (*tview.TreeNode, map[string]*tview.TreeNode) {
	t.Helper()
	nodes := make(map[string]*tview.TreeNode)
	mk := func(name string, kind app.NodeKind) *tview.TreeNode {
		n := tview.NewTreeNode(name).
			SetReference(&app.HierarchyNode{ID: name, Name: name, Kind: kind}).
			SetExpanded(true)
		nodes[name] = n
		return n
	}

	root := tview.NewTreeNode("root")
	root.SetExpanded(true)
	nodes["root"] = root

	a := mk("A", app.NodeFolder)
	a1 := mk("A1", app.NodeList)
	a2 := mk("A2", app.NodeList)
	a.AddChild(a1).AddChild(a2)

	b := mk("B", app.NodeFolder)
	b1 := mk("B1", app.NodeFolder)
	b1a := mk("B1a", app.NodeList)
	b1.AddChild(b1a)
	b2 := mk("B2", app.NodeList)
	b.AddChild(b1).AddChild(b2)

	c := mk("C", app.NodeList)

	root.AddChild(a).AddChild(b).AddChild(c)

	return root, nodes
}

// ── findAncestors ────────────────────────────────────────────────────────────

func TestFindAncestors(t *testing.T) {
	tests := []struct {
		name      string
		target    string
		orphan    bool
		wantFound bool
		wantPath  []string
	}{
		{name: "target_is_root", target: "root", wantFound: true, wantPath: []string{"root"}},
		{name: "leaf_node", target: "B1a", wantFound: true, wantPath: []string{"root", "B", "B1", "B1a"}},
		{name: "direct_child", target: "A", wantFound: true, wantPath: []string{"root", "A"}},
		{name: "not_found", orphan: true, wantFound: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tp := newTestTreePane(t)
			root, nodes := buildTestTree(t)
			tp.root = root

			var target *tview.TreeNode
			if tc.orphan {
				target = tview.NewTreeNode("orphan")
			} else {
				target = nodes[tc.target]
			}

			ancestors := make(map[*tview.TreeNode]bool)
			found := tp.findAncestors(root, target, ancestors)
			assert.Equal(t, tc.wantFound, found)

			if !tc.wantFound {
				assert.Len(t, ancestors, 0)
				return
			}

			for _, name := range tc.wantPath {
				assert.True(t, ancestors[nodes[name]], "ancestors should contain %q", name)
			}
			assert.Len(t, ancestors, len(tc.wantPath))
		})
	}
}

// ── collapseNonAncestors ────────────────────────────────────────────────────

func TestCollapseNonAncestors_CollapsesNonPathBranches(t *testing.T) {
	tp := newTestTreePane(t)
	root, nodes := buildTestTree(t)
	tp.root = root

	ancestors := make(map[*tview.TreeNode]bool)
	require.True(t, tp.findAncestors(root, nodes["B1a"], ancestors))
	tp.collapseNonAncestors(root, ancestors)

	assert.False(t, nodes["A"].IsExpanded())
	assert.False(t, nodes["C"].IsExpanded())
	assert.False(t, nodes["B2"].IsExpanded())
	assert.True(t, nodes["B"].IsExpanded())
	assert.True(t, nodes["B1"].IsExpanded())
}

func TestCollapseNonAncestors_EmptyAncestors_CollapsesRoot(t *testing.T) {
	tp := newTestTreePane(t)
	root, _ := buildTestTree(t)
	tp.root = root

	tp.collapseNonAncestors(root, map[*tview.TreeNode]bool{})

	assert.False(t, root.IsExpanded())
}

// ── collapseExcept (integration of findAncestors + collapseNonAncestors) ───

func TestCollapseExcept_ExpandsPathOnly(t *testing.T) {
	tp := newTestTreePane(t)
	root, nodes := buildTestTree(t)
	tp.root = root

	tp.collapseExcept(nodes["A2"])

	assert.True(t, nodes["A"].IsExpanded())
	assert.False(t, nodes["B"].IsExpanded())
	assert.False(t, nodes["C"].IsExpanded())
}

func TestCollapseExcept_OrphanNode_NoOp(t *testing.T) {
	tp := newTestTreePane(t)
	root, nodes := buildTestTree(t)
	tp.root = root

	tp.collapseExcept(tview.NewTreeNode("orphan"))

	assert.True(t, nodes["A"].IsExpanded())
	assert.True(t, nodes["B"].IsExpanded())
}

// ── makeTreeNode ─────────────────────────────────────────────────────────────

func TestMakeTreeNode_SetsReference(t *testing.T) {
	tp := newTestTreePane(t)
	hn := &app.HierarchyNode{ID: "test1", Name: "Test Node", Kind: app.NodeList}
	node := tp.makeTreeNode(hn)

	ref, ok := node.GetReference().(*app.HierarchyNode)
	require.True(t, ok)
	assert.Equal(t, "test1", ref.ID)
}

func TestMakeTreeNode_ListIncludesID(t *testing.T) {
	tp := newTestTreePane(t)
	hn := &app.HierarchyNode{ID: "abc123", Name: "My List", Kind: app.NodeList}
	node := tp.makeTreeNode(hn)

	text := node.GetText()
	require.NotEmpty(t, text)
	stripped := stripTviewTags(text)
	assert.True(t, strings.Contains(stripped, "#abc123"))
}

func TestMakeTreeNode_NonListOmitsID(t *testing.T) {
	tp := newTestTreePane(t)
	hn := &app.HierarchyNode{ID: "space1", Name: "My Space", Kind: app.NodeSpace}
	node := tp.makeTreeNode(hn)

	stripped := stripTviewTags(node.GetText())
	assert.False(t, strings.Contains(stripped, "#space1"))
}

func TestMakeTreeNode_AllKinds(t *testing.T) {
	tests := []struct {
		name string
		kind app.NodeKind
	}{
		{name: "workspace", kind: app.NodeWorkspace},
		{name: "space", kind: app.NodeSpace},
		{name: "folder", kind: app.NodeFolder},
		{name: "list", kind: app.NodeList},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tp := newTestTreePane(t)
			hn := &app.HierarchyNode{ID: "id", Name: "Name", Kind: tc.kind}
			node := tp.makeTreeNode(hn)
			require.NotEmpty(t, node.GetText())
			ref, ok := node.GetReference().(*app.HierarchyNode)
			require.True(t, ok)
			assert.Equal(t, tc.kind, ref.Kind)
		})
	}
}

// ── updateNodeText ───────────────────────────────────────────────────────────

func TestUpdateNodeText_FolderExpandedToggle(t *testing.T) {
	tp := newTestTreePane(t)
	hn := &app.HierarchyNode{ID: "f1", Name: "Folder", Kind: app.NodeFolder}
	node := tp.makeTreeNode(hn)

	node.SetExpanded(false)
	tp.updateNodeText(node, hn)
	collapsed := stripTviewTags(node.GetText())

	node.SetExpanded(true)
	tp.updateNodeText(node, hn)
	expanded := stripTviewTags(node.GetText())

	assert.NotEqual(t, collapsed, expanded)
}

// ── findNodeByListID ─────────────────────────────────────────────────────────

func TestFindNodeByListID_Found(t *testing.T) {
	tp := newTestTreePane(t)
	root, nodes := buildTestTree(t)
	tp.root = root

	got := tp.findNodeByListID(root, "B1a")
	require.NotNil(t, got)
	assert.Equal(t, nodes["B1a"], got)
}

func TestFindNodeByListID_NotFound(t *testing.T) {
	tp := newTestTreePane(t)
	root, _ := buildTestTree(t)
	tp.root = root

	got := tp.findNodeByListID(root, "nonexistent")
	assert.Nil(t, got)
}

func TestFindNodeByListID_WrongKind(t *testing.T) {
	tp := newTestTreePane(t)
	root, _ := buildTestTree(t)
	tp.root = root

	got := tp.findNodeByListID(root, "A")
	assert.Nil(t, got)
}

// ── SetSpacesAndExpand ───────────────────────────────────────────────────────

func TestSetSpacesAndExpand_SelectsTargetSpace(t *testing.T) {
	tp := newTestTreePane(t)

	spaces := []*app.HierarchyNode{{ID: "space-a", Name: "Space A", Kind: app.NodeSpace}}
	contents := []*app.HierarchyNode{{ID: "list-1", Name: "List 1", Kind: app.NodeList}}

	tp.SetSpacesAndExpand("ws-1", spaces, "space-a", contents)

	current := tp.GetCurrentNode()
	require.NotNil(t, current)
	ref, ok := current.GetReference().(*app.HierarchyNode)
	require.True(t, ok)
	assert.Equal(t, app.NodeSpace, ref.Kind)
	assert.Equal(t, "space-a", ref.ID)
}
