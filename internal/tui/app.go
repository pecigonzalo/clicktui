// Package tui implements the terminal user interface using tview/tcell.
//
// The TUI uses a 3-pane layout:
//   - Left:   workspace/space/folder/list hierarchy tree
//   - Center: task list for the selected list
//   - Right:  task details for the selected task
package tui

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/pecigonzalo/clicktui/internal/app"
)

// App is the main TUI application.
type App struct {
	tviewApp   *tview.Application
	hierarchy  *app.HierarchyService
	tasks      *app.TaskService
	logger     *slog.Logger
	tree       *TreePane
	taskList   *TaskListPane
	taskDetail *TaskDetailPane
	statusBar  *tview.TextView
	layout     *tview.Flex
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
	a.taskDetail = NewTaskDetailPane()
	a.statusBar = tview.NewTextView().
		SetDynamicColors(true).
		SetText("[yellow]Loading workspaces...")

	panes := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(a.tree.TreeView, 0, 1, true).
		AddItem(a.taskList.Table, 0, 2, false).
		AddItem(a.taskDetail.TextView, 0, 2, false)

	a.layout = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(panes, 0, 1, true).
		AddItem(a.statusBar, 1, 0, false)

	a.focusOrder = []tview.Primitive{
		a.tree.TreeView,
		a.taskList.Table,
		a.taskDetail.TextView,
	}

	a.tviewApp.SetRoot(a.layout, true)
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
	a.focusIdx = (a.focusIdx + delta + len(a.focusOrder)) % len(a.focusOrder)
	a.tviewApp.SetFocus(a.focusOrder[a.focusIdx])
}

// Run starts the TUI event loop. It blocks until the application exits.
func (a *App) Run(ctx context.Context) error {
	a.loadWorkspaces(ctx)
	return a.tviewApp.Run()
}

// setStatus updates the status bar text (must be called from the UI goroutine
// or via QueueUpdateDraw).
func (a *App) setStatus(format string, args ...any) {
	a.statusBar.SetText(fmt.Sprintf("[yellow]"+format, args...))
}

// setError shows an error in the status bar.
func (a *App) setError(format string, args ...any) {
	a.statusBar.SetText(fmt.Sprintf("[red]Error: "+format, args...))
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
			a.setStatus("Ready — select a list to browse tasks. Press Tab to switch panes, q to quit.")
		})
	}()
}
