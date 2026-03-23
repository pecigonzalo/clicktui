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
	app  *App
	root *tview.TreeNode
}

// NewTreePane creates an empty hierarchy tree.
func NewTreePane(a *App) *TreePane {
	root := tview.NewTreeNode("Workspaces").SetColor(ColorNodeWorkspace)
	tree := tview.NewTreeView().SetRoot(root).SetCurrentNode(root)
	tree.SetBorder(true)

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
		child.AddChild(tview.NewTreeNode("Loading…").SetColor(ColorTextSubtle))
		tp.root.AddChild(child)
	}
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
	node.AddChild(tview.NewTreeNode("Loading spaces…").SetColor(ColorTextSubtle))
	tp.app.setStatusLoading("Loading spaces…")

	ctx := context.Background()
	go func() {
		children, err := tp.app.hierarchy.LoadSpaces(ctx, ref.ID)
		tp.app.tviewApp.QueueUpdateDraw(func() {
			node.ClearChildren()
			if err != nil {
				tp.app.logger.Error("load spaces", "workspace", ref.ID, "error", err)
				tp.app.setError("load spaces: %v", err)
				node.AddChild(tview.NewTreeNode(fmt.Sprintf("[red]Error: %v", err)))
				return
			}
			ref.Children = children
			ref.Loaded = true
			for _, c := range children {
				child := tp.makeTreeNode(c)
				child.AddChild(tview.NewTreeNode("Loading…").SetColor(ColorTextSubtle))
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
	node.AddChild(tview.NewTreeNode("Loading contents…").SetColor(ColorTextSubtle))
	tp.app.setStatusLoading("Loading folders and lists…")

	ctx := context.Background()
	go func() {
		children, err := tp.app.hierarchy.LoadSpaceContents(ctx, ref.ID)
		tp.app.tviewApp.QueueUpdateDraw(func() {
			node.ClearChildren()
			if err != nil {
				tp.app.logger.Error("load space contents", "space", ref.ID, "error", err)
				tp.app.setError("load space contents: %v", err)
				node.AddChild(tview.NewTreeNode(fmt.Sprintf("[red]Error: %v", err)))
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
	tp.app.taskList.LoadTasks(ref.ID, ref.Name)
}

func (tp *TreePane) makeTreeNode(n *app.HierarchyNode) *tview.TreeNode {
	prefix := nodePrefix(n.Kind)
	return tview.NewTreeNode(prefix + n.Name).
		SetReference(n).
		SetColor(nodeColor(n.Kind)).
		SetSelectable(true)
}

func nodePrefix(k app.NodeKind) string {
	switch k {
	case app.NodeWorkspace:
		return "  "
	case app.NodeSpace:
		return "  ∙ "
	case app.NodeFolder:
		return "    ▸ "
	case app.NodeList:
		return "      "
	default:
		return ""
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
