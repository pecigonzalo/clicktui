// Package tui — hierarchy tree pane.
package tui

import (
	"context"
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/pecigonzalo/clicktui/internal/app"
)

// TreePane renders the workspace/space/folder/list hierarchy as a tree.
type TreePane struct {
	*tview.TreeView
	app         *App
	root        *tview.TreeNode
	styler      *PaneStyler
	selected    string               // name of the currently selected list, empty if none
	cachedNodes []*app.HierarchyNode // unfiltered top-level nodes for filter restore
}

// NewTreePane creates an empty hierarchy tree.
func NewTreePane(a *App) *TreePane {
	root := tview.NewTreeNode("Workspaces").SetColor(ColorNodeWorkspace)
	tree := tview.NewTreeView().SetRoot(root).SetCurrentNode(root)
	tree.SetBorder(true)

	// Turn off the default line graphics — cleaner look with our symbol prefixes.
	tree.SetGraphics(false)

	tp := &TreePane{
		TreeView: tree,
		app:      a,
		root:     root,
	}

	tree.SetSelectedFunc(tp.onSelected)
	return tp
}

// SetWorkspaces populates the tree root with workspace nodes.
func (tp *TreePane) SetWorkspaces(ctx context.Context, nodes []*app.HierarchyNode) {
	tp.root.ClearChildren()
	for _, n := range nodes {
		child := tp.makeTreeNode(n)
		tp.root.AddChild(child)
	}
	tp.snapshotNodes()
}

// SetSpaces populates the tree with space nodes for a single workspace,
// skipping the workspace-level listing. Used when only a workspace ID is
// configured.
func (tp *TreePane) SetSpaces(_ context.Context, workspaceID string, nodes []*app.HierarchyNode) {
	tp.root.ClearChildren()
	wsNode := &app.HierarchyNode{
		ID:       workspaceID,
		Name:     "Workspace",
		Kind:     app.NodeWorkspace,
		Loaded:   true,
		Children: nodes,
	}
	wsTreeNode := tp.makeTreeNode(wsNode)
	for _, n := range nodes {
		child := tp.makeTreeNode(n)
		wsTreeNode.AddChild(child)
	}
	wsTreeNode.SetExpanded(true)
	tp.root.AddChild(wsTreeNode)
	tp.snapshotNodes()
}

// SetSpacesAndExpand populates the tree for a workspace, then expands the
// target space with its pre-loaded contents. Used when both workspace and
// space IDs are configured for auto-navigation.
func (tp *TreePane) SetSpacesAndExpand(
	_ context.Context,
	workspaceID string,
	spaces []*app.HierarchyNode,
	targetSpaceID string,
	contents []*app.HierarchyNode,
) {
	tp.root.ClearChildren()
	wsNode := &app.HierarchyNode{
		ID:       workspaceID,
		Name:     "Workspace",
		Kind:     app.NodeWorkspace,
		Loaded:   true,
		Children: spaces,
	}
	wsTreeNode := tp.makeTreeNode(wsNode)

	for _, s := range spaces {
		spaceTreeNode := tp.makeTreeNode(s)
		if s.ID == targetSpaceID {
			// Mark as loaded and populate with fetched contents.
			s.Loaded = true
			s.Children = contents
			for _, c := range contents {
				child := tp.makeTreeNode(c)
				if c.Kind == app.NodeFolder {
					for _, listNode := range c.Children {
						child.AddChild(tp.makeTreeNode(listNode))
					}
				}
				spaceTreeNode.AddChild(child)
			}
			spaceTreeNode.SetExpanded(true)
			tp.SetCurrentNode(spaceTreeNode)

			// Update the tree title to show the space name.
			tp.selected = s.Name
			tp.refreshTitle()
		}
		wsTreeNode.AddChild(spaceTreeNode)
	}
	wsTreeNode.SetExpanded(true)
	tp.root.AddChild(wsTreeNode)
	tp.snapshotNodes()
}

// refreshTitle updates the pane title to show the selected list context.
// Call after styler is assigned and after selection changes.
func (tp *TreePane) refreshTitle() {
	if tp.styler == nil {
		return
	}
	if tp.selected != "" {
		// Show selected list name in a muted accent so it reads as context
		// without overpowering the base title colour.
		tp.styler.title = "Hierarchy  " + tagColor(ColorTextMuted) + tview.Escape(tp.selected) + "[-]"
	} else {
		tp.styler.title = "Hierarchy"
	}
	// Re-apply current focus state so the title is redrawn.
	tp.styler.reapply()
}

func (tp *TreePane) onSelected(node *tview.TreeNode) {
	ref, ok := node.GetReference().(*app.HierarchyNode)
	if !ok {
		// Root node or placeholder — expand/collapse.
		node.SetExpanded(!node.IsExpanded())
		return
	}

	switch ref.Kind {
	case app.NodeWorkspace:
		tp.expandWorkspace(node, ref)
	case app.NodeSpace:
		tp.expandSpace(node, ref)
	case app.NodeFolder:
		node.SetExpanded(!node.IsExpanded())
	case app.NodeList:
		// Collapse all sibling branches so only the path to the selected
		// list remains expanded. Skip when a filter is active — the filter
		// already expands everything for visibility.
		if tp.app.treeFilter == nil || !tp.app.treeFilter.IsActive() {
			tp.collapseExcept(node)
		}
		tp.selectList(ref)
	}
}

func (tp *TreePane) expandWorkspace(node *tview.TreeNode, ref *app.HierarchyNode) {
	if ref.Loaded {
		node.SetExpanded(!node.IsExpanded())
		return
	}

	node.ClearChildren()
	tp.app.setStatusLoading("Loading spaces…")

	ctx := context.Background()
	go func() {
		children, err := tp.app.hierarchy.LoadSpaces(ctx, ref.ID)
		tp.app.tviewApp.QueueUpdateDraw(func() {
			node.ClearChildren()
			if err != nil {
				tp.app.logger.Error("load spaces", "workspace", ref.ID, "error", err)
				tp.app.setError("load spaces: %v", err)
				node.AddChild(tview.NewTreeNode(fmt.Sprintf("[red]Error: %v", err)).SetSelectable(false))
				return
			}
			ref.Children = children
			ref.Loaded = true
			for _, c := range children {
				child := tp.makeTreeNode(c)
				node.AddChild(child)
			}
			node.SetExpanded(true)
			tp.app.footer.SetStatusReady("Ready")
			tp.snapshotNodes()
		})
	}()
}

func (tp *TreePane) expandSpace(node *tview.TreeNode, ref *app.HierarchyNode) {
	if ref.Loaded {
		node.SetExpanded(!node.IsExpanded())
		return
	}

	node.ClearChildren()
	tp.app.setStatusLoading("Loading folders and lists…")

	ctx := context.Background()
	go func() {
		children, err := tp.app.hierarchy.LoadSpaceContents(ctx, ref.ID)
		tp.app.tviewApp.QueueUpdateDraw(func() {
			node.ClearChildren()
			if err != nil {
				tp.app.logger.Error("load space contents", "space", ref.ID, "error", err)
				tp.app.setError("load space contents: %v", err)
				node.AddChild(tview.NewTreeNode(fmt.Sprintf("[red]Error: %v", err)).SetSelectable(false))
				return
			}
			ref.Children = children
			ref.Loaded = true
			for _, c := range children {
				child := tp.makeTreeNode(c)
				if c.Kind == app.NodeFolder {
					for _, listNode := range c.Children {
						child.AddChild(tp.makeTreeNode(listNode))
					}
				}
				node.AddChild(child)
			}
			node.SetExpanded(true)
			tp.app.footer.SetStatusReady("Ready")
			tp.snapshotNodes()
		})
	}()
}

func (tp *TreePane) selectList(ref *app.HierarchyNode) {
	tp.selected = ref.Name
	tp.refreshTitle()
	tp.app.taskList.LoadTasks(ref.ID, ref.Name)
}

// SelectedNodeID returns the ID of the currently selected tree node, or ""
// when no node with a HierarchyNode reference is selected.
func (tp *TreePane) SelectedNodeID() string {
	node := tp.GetCurrentNode()
	if node == nil {
		return ""
	}
	ref, ok := node.GetReference().(*app.HierarchyNode)
	if !ok {
		return ""
	}
	return ref.ID
}

// snapshotNodes captures the current tree's HierarchyNode references so the
// unfiltered state can be restored later.
func (tp *TreePane) snapshotNodes() {
	tp.cachedNodes = tp.collectRootNodes()
}

// collectRootNodes walks the root's direct children and returns their
// HierarchyNode references.
func (tp *TreePane) collectRootNodes() []*app.HierarchyNode {
	var nodes []*app.HierarchyNode
	for _, child := range tp.root.GetChildren() {
		ref, ok := child.GetReference().(*app.HierarchyNode)
		if ok {
			nodes = append(nodes, ref)
		}
	}
	return nodes
}

// ApplyFilter rebuilds the visual tree using the given filtered hierarchy
// nodes. Pass nil to show all nodes (equivalent to ClearFilter). Must be
// called from the UI goroutine.
func (tp *TreePane) ApplyFilter(filtered []*app.HierarchyNode) {
	if filtered == nil {
		tp.ClearFilter()
		return
	}
	tp.rebuildFromNodes(filtered)
}

// ClearFilter restores the full unfiltered tree. Must be called from the UI
// goroutine.
func (tp *TreePane) ClearFilter() {
	if tp.cachedNodes == nil {
		return
	}
	tp.rebuildFromNodes(tp.cachedNodes)
}

// rebuildFromNodes replaces all visual tree content under root with tree nodes
// constructed from the given hierarchy nodes. All nodes are expanded so the
// filter results are fully visible.
func (tp *TreePane) rebuildFromNodes(nodes []*app.HierarchyNode) {
	tp.root.ClearChildren()
	for _, n := range nodes {
		child := tp.buildSubtree(n)
		tp.root.AddChild(child)
	}
}

// buildSubtree recursively creates tview.TreeNode structures from a
// HierarchyNode and all its children. All nodes are expanded so filter
// results are immediately visible.
func (tp *TreePane) buildSubtree(n *app.HierarchyNode) *tview.TreeNode {
	node := tp.makeTreeNode(n)
	for _, c := range n.Children {
		node.AddChild(tp.buildSubtree(c))
	}
	// If this node has been loaded from the API but the filter produced
	// children, expand it so results are visible.
	if len(n.Children) > 0 {
		node.SetExpanded(true)
	}
	return node
}

// makeTreeNode creates a styled tree node for a hierarchy entry.
// Format: "Symbol Name" — the symbol conveys type, spacing conveys depth via
// tview's own indentation on child nodes.
// For list nodes an additional muted "#ID" suffix is appended so users can
// identify the correct ID to pass to the --list flag.
func (tp *TreePane) makeTreeNode(n *app.HierarchyNode) *tview.TreeNode {
	sym := nodeKindSymbol(n.Kind)
	text := sym + " " + n.Name
	if n.Kind == app.NodeList {
		text += " " + tagColor(ColorTextMuted) + "#" + n.ID + "[-]"
	}
	// Selection style: black on blue so the current node is unmistakably visible.
	selStyle := tcell.StyleDefault.
		Foreground(tcell.ColorBlack).
		Background(ColorBorderFocused).
		Attributes(tcell.AttrBold)
	return tview.NewTreeNode(text).
		SetReference(n).
		SetColor(nodeColor(n.Kind)).
		SetSelectedTextStyle(selStyle).
		SetSelectable(true)
}

// collapseExcept collapses all non-leaf nodes that are not ancestors of the
// selected node. The root and the direct path from root to selected stay
// expanded; everything else is collapsed. This is a no-op if the selected
// node cannot be found in the tree.
func (tp *TreePane) collapseExcept(selected *tview.TreeNode) {
	ancestors := make(map[*tview.TreeNode]bool)
	if !tp.findAncestors(tp.root, selected, ancestors) {
		return
	}
	tp.collapseNonAncestors(tp.root, ancestors)
}

// findAncestors walks the subtree rooted at node, looking for target. When
// target is found, every node on the path from node to target is added to
// ancestors and true is returned.
func (tp *TreePane) findAncestors(node, target *tview.TreeNode, ancestors map[*tview.TreeNode]bool) bool {
	if node == target {
		ancestors[node] = true
		return true
	}
	for _, child := range node.GetChildren() {
		if tp.findAncestors(child, target, ancestors) {
			ancestors[node] = true
			return true
		}
	}
	return false
}

// collapseNonAncestors collapses every non-leaf node that is not in the
// ancestors set. Ancestor nodes stay expanded so the selected node remains
// visible.
func (tp *TreePane) collapseNonAncestors(node *tview.TreeNode, ancestors map[*tview.TreeNode]bool) {
	if !ancestors[node] {
		node.SetExpanded(false)
		return
	}
	// This node is on the path to the selection — keep it expanded and
	// recurse so sibling subtrees below it are collapsed.
	for _, child := range node.GetChildren() {
		tp.collapseNonAncestors(child, ancestors)
	}
}

func nodeColor(k app.NodeKind) tcell.Color {
	switch k {
	case app.NodeWorkspace:
		return ColorNodeWorkspace
	case app.NodeSpace:
		return ColorNodeSpace
	case app.NodeFolder:
		return ColorNodeFolder
	case app.NodeList:
		return ColorNodeList
	default:
		return ColorText
	}
}
