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

// App is the main TUI application.
type App struct {
	tviewApp   *tview.Application
	hierarchy  *app.HierarchyService
	tasks      *app.TaskService
	logger     *slog.Logger
	pages      *tview.Pages
	tree       *TreePane
	taskList   *TaskListPane
	taskDetail *TaskDetailPane
	footer     *Footer
	// paneStylers maps paneID to the chrome controller for that pane.
	paneStylers [3]*PaneStyler
	// focusOrder tracks the panes for Tab cycling.
	focusOrder []tview.Primitive
	focusIdx   int
}

// New creates a TUI application wired to the given services.
func New(hierarchy *app.HierarchyService, tasks *app.TaskService, logger *slog.Logger) *App {
	a := &App{
		tviewApp:  tview.NewApplication(),
		hierarchy: hierarchy,
		tasks:     tasks,
		logger:    logger,
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

	// Layout: hierarchy is narrower than task list; detail gets a good share.
	// Proportions 3:5:4 give the hierarchy enough room for deep names while the
	// task list has the most space for scanning rows.
	panes := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(a.tree, 0, 3, true).
		AddItem(a.taskList, 0, 5, false).
		AddItem(a.taskDetail, 0, 4, false)

	mainLayout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(panes, 0, 1, true).
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
		"s:update status",
		"q:quit",
	)

	a.tviewApp.SetRoot(a.pages, true)
	a.tviewApp.SetInputCapture(a.globalInputHandler)
}

func (a *App) globalInputHandler(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyTab:
		a.cycleFocus(1)
		return nil
	case tcell.KeyBacktab:
		a.cycleFocus(-1)
		return nil
	case tcell.KeyRune:
		if event.Rune() == 'q' {
			a.tviewApp.Stop()
			return nil
		}
	}
	return event
}

func (a *App) cycleFocus(delta int) {
	prev := paneID(a.focusIdx)
	a.focusIdx = (a.focusIdx + delta + len(a.focusOrder)) % len(a.focusOrder)
	next := paneID(a.focusIdx)

	a.paneStylers[prev].SetInactive()
	a.paneStylers[next].SetFocused()

	a.tviewApp.SetFocus(a.focusOrder[a.focusIdx])
}

// Run starts the TUI event loop. It blocks until the application exits.
func (a *App) Run(ctx context.Context) error {
	a.footer.SetStatusLoading("Loading workspaces…")
	a.loadWorkspaces(ctx)
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

func (a *App) loadWorkspaces(ctx context.Context) {
	go func() {
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
	}()
}
