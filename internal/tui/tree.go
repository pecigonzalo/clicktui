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
	app      *App
	root     *tview.TreeNode
	styler   *PaneStyler
	selected string // name of the currently selected list, empty if none
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
		// Add a placeholder so the node is expandable.
		child.AddChild(tview.NewTreeNode("  loading…").SetColor(ColorTextSubtle).SetSelectable(false))
		tp.root.AddChild(child)
	}
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
		child.AddChild(tview.NewTreeNode("  loading…").SetColor(ColorTextSubtle).SetSelectable(false))
		wsTreeNode.AddChild(child)
	}
	wsTreeNode.SetExpanded(true)
	tp.root.AddChild(wsTreeNode)
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
		} else {
			spaceTreeNode.AddChild(
				tview.NewTreeNode("  loading…").SetColor(ColorTextSubtle).SetSelectable(false),
			)
		}
		wsTreeNode.AddChild(spaceTreeNode)
	}
	wsTreeNode.SetExpanded(true)
	tp.root.AddChild(wsTreeNode)
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
		tp.selectList(ref)
	}
}

func (tp *TreePane) expandWorkspace(node *tview.TreeNode, ref *app.HierarchyNode) {
	if ref.Loaded {
		node.SetExpanded(!node.IsExpanded())
		return
	}

	node.ClearChildren()
	node.AddChild(tview.NewTreeNode("  loading spaces…").SetColor(ColorTextSubtle).SetSelectable(false))
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
				child.AddChild(tview.NewTreeNode("  loading…").SetColor(ColorTextSubtle).SetSelectable(false))
				node.AddChild(child)
			}
			node.SetExpanded(true)
			tp.app.footer.SetStatusReady("Ready")
		})
	}()
}

func (tp *TreePane) expandSpace(node *tview.TreeNode, ref *app.HierarchyNode) {
	if ref.Loaded {
		node.SetExpanded(!node.IsExpanded())
		return
	}

	node.ClearChildren()
	node.AddChild(tview.NewTreeNode("  loading contents…").SetColor(ColorTextSubtle).SetSelectable(false))
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
		})
	}()
}

func (tp *TreePane) selectList(ref *app.HierarchyNode) {
	tp.selected = ref.Name
	tp.refreshTitle()
	tp.app.taskList.LoadTasks(ref.ID, ref.Name)
}

// makeTreeNode creates a styled tree node for a hierarchy entry.
// Format: "Symbol Name" — the symbol conveys type, spacing conveys depth via
// tview's own indentation on child nodes.
func (tp *TreePane) makeTreeNode(n *app.HierarchyNode) *tview.TreeNode {
	sym := nodeKindSymbol(n.Kind)
	text := sym + " " + n.Name
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
