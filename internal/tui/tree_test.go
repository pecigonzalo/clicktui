// Package tui — unit tests for tree pane helper functions.
package tui

import (
	"testing"

	"github.com/rivo/tview"

	"github.com/pecigonzalo/clicktui/internal/app"
)

// ── helper: build a minimal TreePane for testing ─────────────────────────────

// newTestTreePane creates a TreePane with just enough wiring to call
// findAncestors, collapseNonAncestors, and collapseExcept. It does NOT
// require a running tview.Application.
func newTestTreePane() *TreePane {
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
//
// Each node gets a HierarchyNode reference so collapseNonAncestors can update
// node text (it reads from the reference).
func buildTestTree() (*tview.TreeNode, map[string]*tview.TreeNode) {
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

func TestFindAncestors_TargetIsRoot(t *testing.T) {
	tp := newTestTreePane()
	root, nodes := buildTestTree()
	tp.root = root

	ancestors := make(map[*tview.TreeNode]bool)
	found := tp.findAncestors(root, nodes["root"], ancestors)
	if !found {
		t.Fatal("findAncestors should find root when target is root")
	}
	if !ancestors[root] {
		t.Error("ancestors should contain root")
	}
	if len(ancestors) != 1 {
		t.Errorf("ancestors should have 1 entry, got %d", len(ancestors))
	}
}

func TestFindAncestors_LeafNode(t *testing.T) {
	tp := newTestTreePane()
	root, nodes := buildTestTree()
	tp.root = root

	ancestors := make(map[*tview.TreeNode]bool)
	found := tp.findAncestors(root, nodes["B1a"], ancestors)
	if !found {
		t.Fatal("findAncestors should find deeply nested node B1a")
	}

	// Path: root → B → B1 → B1a — all four should be ancestors.
	expected := []string{"root", "B", "B1", "B1a"}
	for _, name := range expected {
		if !ancestors[nodes[name]] {
			t.Errorf("ancestors should contain %q", name)
		}
	}
	if len(ancestors) != len(expected) {
		t.Errorf("ancestors count = %d, want %d", len(ancestors), len(expected))
	}
}

func TestFindAncestors_NotFound(t *testing.T) {
	tp := newTestTreePane()
	root, _ := buildTestTree()
	tp.root = root

	orphan := tview.NewTreeNode("orphan")
	ancestors := make(map[*tview.TreeNode]bool)
	found := tp.findAncestors(root, orphan, ancestors)
	if found {
		t.Error("findAncestors should return false for a node not in the tree")
	}
	if len(ancestors) != 0 {
		t.Errorf("ancestors should be empty for missing node, got %d entries", len(ancestors))
	}
}

func TestFindAncestors_DirectChild(t *testing.T) {
	tp := newTestTreePane()
	root, nodes := buildTestTree()
	tp.root = root

	ancestors := make(map[*tview.TreeNode]bool)
	found := tp.findAncestors(root, nodes["A"], ancestors)
	if !found {
		t.Fatal("findAncestors should find direct child A")
	}
	if !ancestors[root] || !ancestors[nodes["A"]] {
		t.Error("ancestors should contain root and A")
	}
	if len(ancestors) != 2 {
		t.Errorf("ancestors count = %d, want 2", len(ancestors))
	}
}

// ── collapseNonAncestors ────────────────────────────────────────────────────

func TestCollapseNonAncestors_CollapsesNonPathBranches(t *testing.T) {
	tp := newTestTreePane()
	root, nodes := buildTestTree()
	tp.root = root

	// Select B1a — path is root → B → B1 → B1a
	ancestors := make(map[*tview.TreeNode]bool)
	tp.findAncestors(root, nodes["B1a"], ancestors)
	tp.collapseNonAncestors(root, ancestors)

	// A and C should be collapsed (not on the path).
	if nodes["A"].IsExpanded() {
		t.Error("node A should be collapsed (not on path to B1a)")
	}
	if nodes["C"].IsExpanded() {
		t.Error("node C should be collapsed (not on path to B1a)")
	}
	// B2 should be collapsed (sibling of B1, not on path).
	if nodes["B2"].IsExpanded() {
		t.Error("node B2 should be collapsed (sibling of B1)")
	}

	// Ancestor nodes should stay expanded.
	if !nodes["B"].IsExpanded() {
		t.Error("node B should stay expanded (on path to B1a)")
	}
	if !nodes["B1"].IsExpanded() {
		t.Error("node B1 should stay expanded (on path to B1a)")
	}
}

func TestCollapseNonAncestors_EmptyAncestors_CollapsesRoot(t *testing.T) {
	tp := newTestTreePane()
	root, _ := buildTestTree()
	tp.root = root

	// Empty ancestors — root itself is not an ancestor so it gets collapsed.
	// Its children are not visited (collapsing a parent hides them visually).
	tp.collapseNonAncestors(root, map[*tview.TreeNode]bool{})

	if root.IsExpanded() {
		t.Error("root should be collapsed with empty ancestors")
	}
}

// ── collapseExcept (integration of findAncestors + collapseNonAncestors) ────

func TestCollapseExcept_ExpandsPathOnly(t *testing.T) {
	tp := newTestTreePane()
	root, nodes := buildTestTree()
	tp.root = root

	tp.collapseExcept(nodes["A2"])

	// A2's path: root → A → A2. A should remain expanded.
	if !nodes["A"].IsExpanded() {
		t.Error("A should stay expanded (parent of selected A2)")
	}
	// B should be collapsed.
	if nodes["B"].IsExpanded() {
		t.Error("B should be collapsed (not on path to A2)")
	}
	if nodes["C"].IsExpanded() {
		t.Error("C should be collapsed (not on path to A2)")
	}
}

func TestCollapseExcept_OrphanNode_NoOp(t *testing.T) {
	tp := newTestTreePane()
	root, nodes := buildTestTree()
	tp.root = root

	// Selecting a node not in the tree should be a no-op (no panic, no change).
	orphan := tview.NewTreeNode("orphan")
	tp.collapseExcept(orphan)

	// All nodes should remain in their original expanded state (all expanded from buildTestTree).
	if !nodes["A"].IsExpanded() {
		t.Error("A should remain expanded after no-op collapseExcept")
	}
	if !nodes["B"].IsExpanded() {
		t.Error("B should remain expanded after no-op collapseExcept")
	}
}

// ── makeTreeNode ─────────────────────────────────────────────────────────────

func TestMakeTreeNode_SetsReference(t *testing.T) {
	tp := newTestTreePane()
	hn := &app.HierarchyNode{ID: "test1", Name: "Test Node", Kind: app.NodeList}
	node := tp.makeTreeNode(hn)

	ref, ok := node.GetReference().(*app.HierarchyNode)
	if !ok {
		t.Fatal("makeTreeNode should set *app.HierarchyNode reference")
	}
	if ref.ID != "test1" {
		t.Errorf("reference ID = %q, want 'test1'", ref.ID)
	}
}

func TestMakeTreeNode_ListIncludesID(t *testing.T) {
	tp := newTestTreePane()
	hn := &app.HierarchyNode{ID: "abc123", Name: "My List", Kind: app.NodeList}
	node := tp.makeTreeNode(hn)

	text := node.GetText()
	if text == "" {
		t.Fatal("makeTreeNode should set non-empty text")
	}
	// List nodes should include #ID suffix.
	stripped := stripTviewTags(text)
	if !containsSubstring(stripped, "#abc123") {
		t.Errorf("list node text %q should contain '#abc123'", stripped)
	}
}

func TestMakeTreeNode_NonListOmitsID(t *testing.T) {
	tp := newTestTreePane()
	hn := &app.HierarchyNode{ID: "space1", Name: "My Space", Kind: app.NodeSpace}
	node := tp.makeTreeNode(hn)

	text := node.GetText()
	stripped := stripTviewTags(text)
	if containsSubstring(stripped, "#space1") {
		t.Errorf("non-list node text %q should not contain '#space1'", stripped)
	}
}

func TestMakeTreeNode_AllKinds(t *testing.T) {
	tp := newTestTreePane()
	kinds := []app.NodeKind{
		app.NodeWorkspace,
		app.NodeSpace,
		app.NodeFolder,
		app.NodeList,
	}
	for _, k := range kinds {
		hn := &app.HierarchyNode{ID: "id", Name: "Name", Kind: k}
		node := tp.makeTreeNode(hn)
		if node.GetText() == "" {
			t.Errorf("makeTreeNode(kind=%v) should set non-empty text", k)
		}
		// tview.TreeNode.SetSelectable returns the node for chaining;
		// we verify the reference was set correctly instead.
		ref, ok := node.GetReference().(*app.HierarchyNode)
		if !ok || ref.Kind != k {
			t.Errorf("makeTreeNode(kind=%v) reference mismatch", k)
		}
	}
}

// ── updateNodeText ──────────────────────────────────────────────────────────

func TestUpdateNodeText_FolderExpandedToggle(t *testing.T) {
	tp := newTestTreePane()
	hn := &app.HierarchyNode{ID: "f1", Name: "Folder", Kind: app.NodeFolder}
	node := tp.makeTreeNode(hn)

	// Initially collapsed — should use closed icon.
	node.SetExpanded(false)
	tp.updateNodeText(node, hn)
	collapsed := stripTviewTags(node.GetText())

	node.SetExpanded(true)
	tp.updateNodeText(node, hn)
	expanded := stripTviewTags(node.GetText())

	if collapsed == expanded {
		t.Errorf("folder node text should differ when expanded vs collapsed; both = %q", collapsed)
	}
}

// ── findNodeByListID ─────────────────────────────────────────────────────────

func TestFindNodeByListID_Found(t *testing.T) {
	tp := newTestTreePane()
	root, nodes := buildTestTree()
	tp.root = root

	// B1a is a NodeList node with ID "B1a" (set by buildTestTree).
	got := tp.findNodeByListID(root, "B1a")
	if got == nil {
		t.Fatal("findNodeByListID should find list node B1a")
	}
	if got != nodes["B1a"] {
		t.Errorf("findNodeByListID returned wrong node")
	}
}

func TestFindNodeByListID_NotFound(t *testing.T) {
	tp := newTestTreePane()
	root, _ := buildTestTree()
	tp.root = root

	got := tp.findNodeByListID(root, "nonexistent")
	if got != nil {
		t.Error("findNodeByListID should return nil for a missing list ID")
	}
}

func TestFindNodeByListID_WrongKind(t *testing.T) {
	tp := newTestTreePane()
	root, _ := buildTestTree()
	tp.root = root

	// "A" exists in the tree but is a NodeFolder, not a NodeList.
	got := tp.findNodeByListID(root, "A")
	if got != nil {
		t.Error("findNodeByListID should not match non-list nodes")
	}
}

// ── SetSpacesAndExpand ────────────────────────────────────────────────────────

func TestSetSpacesAndExpand_SelectsTargetSpace(t *testing.T) {
	tp := newTestTreePane()

	spaces := []*app.HierarchyNode{
		{ID: "space-a", Name: "Space A", Kind: app.NodeSpace},
	}
	contents := []*app.HierarchyNode{{ID: "list-1", Name: "List 1", Kind: app.NodeList}}

	tp.SetSpacesAndExpand("ws-1", spaces, "space-a", contents)

	current := tp.GetCurrentNode()
	if current == nil {
		t.Fatal("current node should be set")
	}
	ref, ok := current.GetReference().(*app.HierarchyNode)
	if !ok {
		t.Fatal("current node should reference a hierarchy node")
	}
	if ref.Kind != app.NodeSpace || ref.ID != "space-a" {
		t.Fatalf("current node = %v %q, want space %q", ref.Kind, ref.ID, "space-a")
	}
}

// ── helper ──────────────────────────────────────────────────────────────────

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || findSubstring(s, sub))
}

func findSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
