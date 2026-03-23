// Package tui implements the terminal user interface using tview/tcell.
//
// The TUI uses a 3-pane layout:
//   - Left:   workspace/space/folder/list hierarchy tree
//   - Center: task list for the selected list
//   - Right:  task details for the selected task
//
// Modal overlays (e.g. the status picker) are layered on top via tview.Pages.
package tui

import (
	"context"
	"log/slog"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/pecigonzalo/clicktui/internal/app"
)

const pageMain = "main"

// paneID identifies which pane has focus.
type paneID int

const (
	paneTree paneID = iota
	paneTaskList
	paneTaskDetail
)

// LaunchOptions configures the initial navigation state of the TUI.
// When all three IDs are set (WorkspaceID, SpaceID, ListID), the TUI
// auto-navigates directly to that list, loading tasks immediately and
// focusing the task list pane. When only WorkspaceID and SpaceID are set,
// the TUI expands the space in the tree and focuses the tree pane. When
// only WorkspaceID is set, the TUI loads spaces for that workspace. When
// neither is set, the TUI loads all workspaces (default behaviour).
type LaunchOptions struct {
	WorkspaceID string
	SpaceID     string
	// ListID is an optional ClickUp list ID to navigate to on launch.
	// Requires WorkspaceID and SpaceID to also be set.
	ListID string
}

// App is the main TUI application.
type App struct {
	tviewApp   *tview.Application
	hierarchy  *app.HierarchyService
	tasks      *app.TaskService
	logger     *slog.Logger
	pages      *tview.Pages
	panes      *tview.Flex // column flex holding tree, taskList, taskDetail
	tree       *TreePane
	taskList   *TaskListPane
	taskDetail *TaskDetailPane
	footer     *Footer
	// paneStylers maps paneID to the chrome controller for that pane.
	paneStylers [3]*PaneStyler
	// focusOrder tracks the panes for Tab cycling.
	focusOrder []tview.Primitive
	focusIdx   int

	// Filter overlays — one per filterable pane (tree, task list).
	// The detail pane does not support filtering.
	treeFilter     *FilterOverlay
	taskListFilter *FilterOverlay
	// treeFilterRow and taskListFilterRow are Flex containers that hold the
	// pane + its filter input. They replace the raw pane in the column layout
	// so the filter bar appears at the bottom of the pane.
	treeFilterRow     *tview.Flex
	taskListFilterRow *tview.Flex

	launch       LaunchOptions
	treeVisible  bool
	treeMinWidth int // proportion when tree is visible
}

// New creates a TUI application wired to the given services.
func New(hierarchy *app.HierarchyService, tasks *app.TaskService, logger *slog.Logger, opts LaunchOptions) *App {
	a := &App{
		tviewApp:     tview.NewApplication(),
		hierarchy:    hierarchy,
		tasks:        tasks,
		logger:       logger,
		launch:       opts,
		treeVisible:  true,
		treeMinWidth: 3,
	}
	a.buildLayout()
	return a
}

func (a *App) buildLayout() {
	a.tree = NewTreePane(a)
	a.taskList = NewTaskListPane(a)
	a.taskDetail = NewTaskDetailPane(a)
	a.footer = newFooter()

	// Register pane chrome controllers so we can update focus styling.
	// Each tview widget exposes its embedded *Box via the promoted .Box field.
	a.paneStylers[paneTree] = newPaneStyler(a.tree.Box, "Hierarchy")
	a.paneStylers[paneTaskList] = newPaneStyler(a.taskList.Box, "Tasks")
	a.paneStylers[paneTaskDetail] = newPaneStyler(a.taskDetail.Box, "Detail")

	// Give the tree pane access to its own styler so it can update the title
	// when the selected list changes.
	a.tree.styler = a.paneStylers[paneTree]
	// Give the task list pane access to its styler for dynamic title updates.
	a.taskList.styler = a.paneStylers[paneTaskList]

	// Apply initial border + title styles.
	a.paneStylers[paneTree].SetFocused()
	a.paneStylers[paneTaskList].SetInactive()
	a.paneStylers[paneTaskDetail].SetInactive()

	// Create filter overlays for the tree and task list panes.
	a.treeFilter = NewFilterOverlay(
		func(text string) {
			a.paneStylers[paneTree].SetFilterText(text)
			// Apply hierarchy filter: filter the cached tree nodes and rebuild.
			if text == "" {
				a.tree.ClearFilter()
			} else {
				filtered := app.FilterHierarchy(a.tree.cachedNodes, text)
				a.tree.ApplyFilter(filtered)
			}
		},
		func() {
			a.paneStylers[paneTree].SetFilterText("")
			// Hide the filter input row.
			a.treeFilterRow.ResizeItem(a.treeFilter.InputField(), 0, 0)
			a.restoreDefaultHelp()
			// Restore unfiltered hierarchy.
			a.tree.ClearFilter()
		},
		func() {
			// Return focus and restore help when filter is applied (Enter).
			a.tviewApp.SetFocus(a.tree)
			if !a.treeFilter.IsActive() {
				a.treeFilterRow.ResizeItem(a.treeFilter.InputField(), 0, 0)
			}
			a.restoreDefaultHelp()
		},
	)

	a.taskListFilter = NewFilterOverlay(
		func(text string) {
			a.paneStylers[paneTaskList].SetFilterText(text)
			// Apply task filter: parse query and filter the full task set.
			if text == "" {
				a.taskList.ClearFilter()
			} else {
				query := app.ParseTaskQuery(text)
				filtered := app.FilterTasks(a.taskList.allTasks, query)
				a.taskList.ApplyFilter(filtered, query)
			}
		},
		func() {
			a.paneStylers[paneTaskList].SetFilterText("")
			// Hide the filter input row.
			a.taskListFilterRow.ResizeItem(a.taskListFilter.InputField(), 0, 0)
			a.restoreDefaultHelp()
			// Restore unfiltered task list.
			a.taskList.ClearFilter()
		},
		func() {
			// Return focus and restore help when filter is applied (Enter).
			a.tviewApp.SetFocus(a.taskList)
			if !a.taskListFilter.IsActive() {
				a.taskListFilterRow.ResizeItem(a.taskListFilter.InputField(), 0, 0)
			}
			a.restoreDefaultHelp()
		},
	)

	// Wrap each filterable pane in a vertical Flex: pane (grows) + filter bar
	// (1 row, hidden by default). The filter input height is 1 when visible.
	a.treeFilterRow = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.tree, 0, 1, true).
		AddItem(a.treeFilter.InputField(), 0, 0, false)

	a.taskListFilterRow = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.taskList, 0, 1, true).
		AddItem(a.taskListFilter.InputField(), 0, 0, false)

	// Layout: hierarchy is narrower than task list; detail gets a good share.
	// Proportions 3:5:4 give the hierarchy enough room for deep names while the
	// task list has the most space for scanning rows.
	a.panes = tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(a.treeFilterRow, 0, 3, true).
		AddItem(a.taskListFilterRow, 0, 5, false).
		AddItem(a.taskDetail, 0, 4, false)

	mainLayout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.panes, 0, 1, true).
		AddItem(a.footer, 1, 0, false)

	a.pages = tview.NewPages().AddPage(pageMain, mainLayout, true, true)

	a.focusOrder = []tview.Primitive{
		a.tree,
		a.taskList,
		a.taskDetail,
	}

	// Set global help that applies when no task is selected.
	a.footer.SetHelp(
		"Tab:next pane",
		"Shift+Tab:prev pane",
		"Enter:select",
		"/:filter",
		"[:toggle tree",
		"s:update status",
		"y:copy id",
		"q:quit",
	)

	a.tviewApp.SetRoot(a.pages, true)
	a.tviewApp.SetInputCapture(a.globalInputHandler)
}

func (a *App) globalInputHandler(event *tcell.EventKey) *tcell.EventKey {
	// When a filter overlay is in edit mode, only allow Esc and Enter
	// (handled by the overlay itself). Block all other global keybindings
	// so they don't propagate.
	if a.isFilterEditing() {
		return event
	}

	switch event.Key() {
	case tcell.KeyTab:
		a.cycleFocus(1)
		return nil
	case tcell.KeyBacktab:
		a.cycleFocus(-1)
		return nil
	case tcell.KeyRune:
		switch event.Rune() {
		case '/':
			a.activateFilter()
			return nil
		case 'q':
			a.tviewApp.Stop()
			return nil
		case '[':
			a.toggleTree()
			return nil
		case 'y':
			a.yankID()
			return nil
		}
	}
	return event
}

func (a *App) cycleFocus(delta int) {
	next := paneID((a.focusIdx + delta + len(a.focusOrder)) % len(a.focusOrder))
	// Skip the tree pane when it is collapsed.
	if !a.treeVisible && next == paneTree {
		next = paneID((int(next) + delta + len(a.focusOrder)) % len(a.focusOrder))
	}
	a.setFocusPane(next)
}

// setFocusPane moves focus to the given pane, updates chrome on both the
// outgoing and incoming pane, and keeps focusIdx in sync.  Use this instead of
// calling tviewApp.SetFocus directly so that border colours always match the
// actual focus state.
func (a *App) setFocusPane(id paneID) {
	prev := paneID(a.focusIdx)
	if prev != id {
		a.paneStylers[prev].SetInactive()
	}
	a.focusIdx = int(id)
	a.paneStylers[id].SetFocused()
	a.tviewApp.SetFocus(a.focusOrder[id])
}

// toggleTree shows or hides the tree pane.
func (a *App) toggleTree() {
	a.setTreeVisible(!a.treeVisible)
}

// yankID copies the ID of the currently focused entity to the system clipboard.
// The entity is determined by which pane currently holds focus:
//   - Tree pane: the selected hierarchy node's ID
//   - Task list pane: the selected task's ID
//   - Task detail pane: the currently displayed task's ID
func (a *App) yankID() {
	var id string
	switch paneID(a.focusIdx) {
	case paneTree:
		id = a.tree.SelectedNodeID()
	case paneTaskList:
		id = a.taskList.SelectedTaskID()
	case paneTaskDetail:
		id = a.taskDetail.CurrentTaskID()
	}
	if id == "" {
		a.footer.SetStatusReady("Nothing to copy")
		return
	}
	if err := writeClipboard(id); err != nil {
		a.footer.SetStatusError("copy failed: %v", err)
		return
	}
	a.footer.SetStatusReady("Copied: #" + id)
}

// setTreeVisible controls tree pane visibility. When hiding, focus moves to
// the task list pane. When showing, the tree regains its original proportional
// width.
func (a *App) setTreeVisible(visible bool) {
	if a.treeVisible == visible {
		return
	}
	a.treeVisible = visible

	// Rebuild the panes flex — tview does not support hiding individual items.
	a.panes.Clear()
	if visible {
		a.panes.AddItem(a.treeFilterRow, 0, a.treeMinWidth, false)
	}
	a.panes.AddItem(a.taskListFilterRow, 0, 5, !visible)
	a.panes.AddItem(a.taskDetail, 0, 4, false)

	if !visible && paneID(a.focusIdx) == paneTree {
		a.setFocusPane(paneTaskList)
	}
}

// ── Filter overlay management ────────────────────────────────────────────────

// isFilterEditing reports whether any filter overlay is currently in editing
// mode (accepting keystrokes).
func (a *App) isFilterEditing() bool {
	return a.treeFilter.IsEditing() || a.taskListFilter.IsEditing()
}

// activateFilter shows the filter overlay for the currently focused pane.
// Only the tree and task list panes support filtering.
func (a *App) activateFilter() {
	focusedPane := paneID(a.focusIdx)

	var overlay *FilterOverlay
	var filterRow *tview.Flex

	switch focusedPane {
	case paneTree:
		overlay = a.treeFilter
		filterRow = a.treeFilterRow
	case paneTaskList:
		overlay = a.taskListFilter
		filterRow = a.taskListFilterRow
	default:
		// Detail pane — no filtering.
		return
	}

	// Show the filter input row (1-line height).
	filterRow.ResizeItem(overlay.InputField(), 1, 0)
	overlay.Show()
	a.tviewApp.SetFocus(overlay.InputField())

	// Update footer to show filter-mode keybindings.
	a.footer.SetHelp("Enter:apply filter", "Esc:clear filter")
}

// restoreDefaultHelp resets the footer help to the standard keybinding set.
func (a *App) restoreDefaultHelp() {
	a.footer.SetHelp(
		"Tab:next pane",
		"Shift+Tab:prev pane",
		"Enter:select",
		"/:filter",
		"[:toggle tree",
		"s:update status",
		"y:copy id",
		"q:quit",
	)
}

// Run starts the TUI event loop. It blocks until the application exits.
//
// The initial data load runs in a goroutine launched just before the blocking
// tviewApp.Run() call. This avoids a deadlock: the load helpers call
// QueueUpdateDraw which blocks until the event loop is running, so the
// goroutine must be started before Run() takes over the main goroutine.
func (a *App) Run(ctx context.Context) error {
	// Set the loading status directly (no event loop needed for this).
	switch {
	case a.launch.WorkspaceID != "" && a.launch.SpaceID != "" && a.launch.ListID != "":
		a.footer.SetStatusLoading("Navigating to list…")
	case a.launch.WorkspaceID != "" && a.launch.SpaceID != "":
		a.footer.SetStatusLoading("Navigating to space…")
	case a.launch.WorkspaceID != "":
		a.footer.SetStatusLoading("Loading spaces…")
	default:
		a.footer.SetStatusLoading("Loading workspaces…")
	}

	// Launch initial load in a background goroutine. The QueueUpdateDraw
	// calls inside will block briefly until tviewApp.Run() starts below.
	go a.initialLoad(ctx)

	return a.tviewApp.Run()
}

// setStatusLoading shows a yellow loading message in the footer.
func (a *App) setStatusLoading(format string, args ...any) {
	a.footer.SetStatusLoading(format, args...)
}

// setError shows a red error message in the footer.
func (a *App) setError(format string, args ...any) {
	a.footer.SetStatusError(format, args...)
}

// initialLoad dispatches the correct initial data load based on launch options.
// It runs synchronously (blocking on API calls + QueueUpdateDraw) and must be
// called from a goroutine — never from the main goroutine before tviewApp.Run().
func (a *App) initialLoad(ctx context.Context) {
	switch {
	case a.launch.WorkspaceID != "" && a.launch.SpaceID != "" && a.launch.ListID != "":
		a.doAutoNavToList(ctx, a.launch.WorkspaceID, a.launch.SpaceID, a.launch.ListID)
	case a.launch.WorkspaceID != "" && a.launch.SpaceID != "":
		a.doAutoNavToSpace(ctx, a.launch.WorkspaceID, a.launch.SpaceID)
	case a.launch.WorkspaceID != "":
		a.doAutoNavToWorkspace(ctx, a.launch.WorkspaceID)
	default:
		a.doLoadWorkspaces(ctx)
	}
}

// doLoadWorkspaces fetches workspaces and updates the tree. It blocks until the
// API call and UI update are complete.
func (a *App) doLoadWorkspaces(ctx context.Context) {
	nodes, err := a.hierarchy.LoadWorkspaces(ctx)
	a.tviewApp.QueueUpdateDraw(func() {
		if err != nil {
			a.logger.Error("load workspaces", "error", err)
			a.setError("failed to load workspaces: %v", err)
			return
		}
		a.tree.SetWorkspaces(ctx, nodes)
		a.footer.SetStatusReady("Ready")
	})
}

// doAutoNavToWorkspace fetches spaces for a workspace and updates the tree.
// It blocks until the API call and UI update are complete.
func (a *App) doAutoNavToWorkspace(ctx context.Context, workspaceID string) {
	spaces, err := a.hierarchy.LoadSpaces(ctx, workspaceID)
	a.tviewApp.QueueUpdateDraw(func() {
		if err != nil {
			a.logger.Error("auto-nav: load spaces", "workspace", workspaceID, "error", err)
			a.setError("load spaces: %v", err)
			return
		}
		a.tree.SetSpaces(ctx, workspaceID, spaces)
		a.footer.SetStatusReady("Ready")
	})
}

// doAutoNavToSpace loads spaces and a specific space's contents, then updates
// the tree with the space expanded and sets focus to the tree pane so the user
// can select a folder or list. It blocks until all API calls and UI updates are
// complete.
func (a *App) doAutoNavToSpace(ctx context.Context, workspaceID, spaceID string) {
	// Load spaces to find the target space name, then load its contents.
	spaces, err := a.hierarchy.LoadSpaces(ctx, workspaceID)
	if err != nil {
		a.tviewApp.QueueUpdateDraw(func() {
			a.logger.Error("auto-nav: load spaces", "workspace", workspaceID, "error", err)
			a.setError("load spaces: %v", err)
		})
		return
	}

	contents, err := a.hierarchy.LoadSpaceContents(ctx, spaceID)
	if err != nil {
		a.tviewApp.QueueUpdateDraw(func() {
			a.logger.Error("auto-nav: load space contents", "space", spaceID, "error", err)
			a.setError("load space contents: %v", err)
		})
		return
	}

	a.tviewApp.QueueUpdateDraw(func() {
		a.tree.SetSpacesAndExpand(ctx, workspaceID, spaces, spaceID, contents)
		a.setFocusPane(paneTree)
		a.footer.SetStatusReady("Ready")
	})
}

// doAutoNavToList loads spaces and the target space's contents, populates the
// tree, and immediately loads the task list for the given list ID. Focus is set
// to the task list pane since the user already knows which list they want.
// The tree remains visible for context. It blocks until all API calls and UI
// updates are complete.
func (a *App) doAutoNavToList(ctx context.Context, workspaceID, spaceID, listID string) {
	spaces, err := a.hierarchy.LoadSpaces(ctx, workspaceID)
	if err != nil {
		a.tviewApp.QueueUpdateDraw(func() {
			a.logger.Error("auto-nav: load spaces", "workspace", workspaceID, "error", err)
			a.setError("load spaces: %v", err)
		})
		return
	}

	contents, err := a.hierarchy.LoadSpaceContents(ctx, spaceID)
	if err != nil {
		a.tviewApp.QueueUpdateDraw(func() {
			a.logger.Error("auto-nav: load space contents", "space", spaceID, "error", err)
			a.setError("load space contents: %v", err)
		})
		return
	}

	// Find the list name from the loaded hierarchy so the task list title is
	// meaningful. Fall back to the raw ID when the list is not found.
	listName := findListName(contents, listID)
	if listName == "" {
		listName = listID
	}

	a.tviewApp.QueueUpdateDraw(func() {
		a.tree.SetSpacesAndExpand(ctx, workspaceID, spaces, spaceID, contents)
		a.taskList.LoadTasks(listID, listName)
		a.setFocusPane(paneTaskList)
		a.footer.SetStatusReady("Ready")
	})
}

// findListName searches the hierarchy nodes (and their children) for a list
// node matching listID and returns its name. Returns "" when not found.
func findListName(nodes []*app.HierarchyNode, listID string) string {
	for _, n := range nodes {
		if n.ID == listID && n.Kind == app.NodeList {
			return n.Name
		}
		if name := findListName(n.Children, listID); name != "" {
			return name
		}
	}
	return ""
}
