// Package tui — unit tests for app-level state logic and helpers.
package tui

import (
	"testing"

	"github.com/gdamore/tcell/v2"

	"github.com/pecigonzalo/clicktui/internal/app"
)

// ── cycleFocusIndex — pure focus-cycling calculation ─────────────────────────
// cycleFocus calls setFocusPane which requires a live tview.Application, so we
// test the core arithmetic that cycleFocus uses. The formula is:
//   next = (focusIdx + delta + count) % count
//   if !treeVisible && next == paneTree: skip by applying delta again

func cycleFocusIndex(focusIdx, delta, count int, treeVisible bool) int {
	next := (focusIdx + delta + count) % count
	if !treeVisible && paneID(next) == paneTree {
		next = (next + delta + count) % count
	}
	return next
}

func TestCycleFocusIndex_ForwardWraps(t *testing.T) {
	cases := []struct {
		name     string
		focusIdx int
		delta    int
		count    int
		want     int
	}{
		{"from_tree_to_tasklist", 0, 1, 3, 1},
		{"from_tasklist_to_detail", 1, 1, 3, 2},
		{"from_detail_wraps_to_tree", 2, 1, 3, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := cycleFocusIndex(tc.focusIdx, tc.delta, tc.count, true)
			if got != tc.want {
				t.Errorf("cycleFocusIndex(%d, %d, %d, true) = %d, want %d",
					tc.focusIdx, tc.delta, tc.count, got, tc.want)
			}
		})
	}
}

func TestCycleFocusIndex_BackwardWraps(t *testing.T) {
	cases := []struct {
		name     string
		focusIdx int
		want     int
	}{
		{"from_tree_wraps_to_detail", 0, 2},
		{"from_detail_to_tasklist", 2, 1},
		{"from_tasklist_to_tree", 1, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := cycleFocusIndex(tc.focusIdx, -1, 3, true)
			if got != tc.want {
				t.Errorf("cycleFocusIndex(%d, -1, 3, true) = %d, want %d",
					tc.focusIdx, got, tc.want)
			}
		})
	}
}

func TestCycleFocusIndex_SkipsTreeWhenHidden(t *testing.T) {
	cases := []struct {
		name     string
		focusIdx int
		delta    int
		want     int
	}{
		// Tree hidden: forward from detail (2) should skip tree (0) → tasklist (1)
		{"forward_skips_tree", 2, 1, 1},
		// Tree hidden: backward from tasklist (1) should skip tree (0) → detail (2)
		{"backward_skips_tree", 1, -1, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := cycleFocusIndex(tc.focusIdx, tc.delta, 3, false)
			if got != tc.want {
				t.Errorf("cycleFocusIndex(%d, %d, 3, false) = %d, want %d",
					tc.focusIdx, tc.delta, got, tc.want)
			}
		})
	}
}

// ── findListName — pure recursive search ─────────────────────────────────────

func TestFindListName_FoundAtTopLevel(t *testing.T) {
	nodes := []*app.HierarchyNode{
		{ID: "list1", Name: "Sprint Tasks", Kind: app.NodeList},
		{ID: "list2", Name: "Backlog", Kind: app.NodeList},
	}
	got := findListName(nodes, "list2")
	if got != "Backlog" {
		t.Errorf("findListName(nodes, 'list2') = %q, want 'Backlog'", got)
	}
}

func TestFindListName_FoundNested(t *testing.T) {
	nodes := []*app.HierarchyNode{
		{
			ID:   "folder1",
			Name: "Engineering",
			Kind: app.NodeFolder,
			Children: []*app.HierarchyNode{
				{ID: "list-deep", Name: "Deep List", Kind: app.NodeList},
			},
		},
	}
	got := findListName(nodes, "list-deep")
	if got != "Deep List" {
		t.Errorf("findListName(nodes, 'list-deep') = %q, want 'Deep List'", got)
	}
}

func TestFindListName_NotFound(t *testing.T) {
	nodes := []*app.HierarchyNode{
		{ID: "list1", Name: "Sprint Tasks", Kind: app.NodeList},
	}
	got := findListName(nodes, "nonexistent")
	if got != "" {
		t.Errorf("findListName(nodes, 'nonexistent') = %q, want empty", got)
	}
}

func TestFindListName_SkipsNonListNodes(t *testing.T) {
	// A folder node with the matching ID should not be returned because
	// findListName only matches NodeList nodes.
	nodes := []*app.HierarchyNode{
		{ID: "id1", Name: "A Folder", Kind: app.NodeFolder},
	}
	got := findListName(nodes, "id1")
	if got != "" {
		t.Errorf("findListName should skip non-list nodes, got %q", got)
	}
}

func TestFindListName_EmptyNodes(t *testing.T) {
	got := findListName(nil, "anything")
	if got != "" {
		t.Errorf("findListName(nil, 'anything') = %q, want empty", got)
	}
}

func TestFindListName_DeeplyNested(t *testing.T) {
	nodes := []*app.HierarchyNode{
		{
			ID:   "ws",
			Name: "Workspace",
			Kind: app.NodeWorkspace,
			Children: []*app.HierarchyNode{
				{
					ID:   "space",
					Name: "Space",
					Kind: app.NodeSpace,
					Children: []*app.HierarchyNode{
						{
							ID:   "folder",
							Name: "Folder",
							Kind: app.NodeFolder,
							Children: []*app.HierarchyNode{
								{ID: "deep-list", Name: "Deep List", Kind: app.NodeList},
							},
						},
					},
				},
			},
		},
	}
	got := findListName(nodes, "deep-list")
	if got != "Deep List" {
		t.Errorf("findListName(nodes, 'deep-list') = %q, want 'Deep List'", got)
	}
}

// ── globalInputHandler — key routing ─────────────────────────────────────────
// We cannot fully test globalInputHandler without a live tview.Application, but
// we can verify the routing logic by checking which events are consumed (nil)
// vs passed through. The filter editing guard blocks all global keys, so we
// test via the FilterOverlay state.

func TestGlobalInputHandler_FilterEditingPassesThrough(t *testing.T) {
	// When a filter overlay is in editing mode, globalInputHandler should
	// pass through all events (return the event unchanged).
	fo := NewFilterOverlay(func(string) {}, func() {}, func() {})
	fo.Show() // editing = true

	a := &App{
		treeFilter:     fo,
		taskListFilter: NewFilterOverlay(func(string) {}, func() {}, func() {}),
	}

	// Tab should normally be consumed, but with filter editing it passes through.
	event := tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone)
	result := a.globalInputHandler(event)
	if result == nil {
		t.Error("globalInputHandler(Tab) with filter editing should pass through, got nil")
	}

	// 'q' should normally quit, but with filter editing it passes through.
	qEvent := tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModNone)
	result = a.globalInputHandler(qEvent)
	if result == nil {
		t.Error("globalInputHandler('q') with filter editing should pass through, got nil")
	}
}

// ── Modal state management ────────────────────────────────────────────────────

func TestIsModalActive_FalseByDefault(t *testing.T) {
	a := &App{}
	if a.IsModalActive() {
		t.Error("IsModalActive() = true on zero-value App, want false")
	}
}

func TestSetModalActive_True(t *testing.T) {
	a := &App{}
	a.SetModalActive(true)
	if !a.IsModalActive() {
		t.Error("IsModalActive() = false after SetModalActive(true), want true")
	}
}

func TestSetModalActive_False(t *testing.T) {
	a := &App{modalActive: true}
	a.SetModalActive(false)
	if a.IsModalActive() {
		t.Error("IsModalActive() = true after SetModalActive(false), want false")
	}
}

func TestGlobalInputHandler_ModalActive_SuppressesAllKeys(t *testing.T) {
	// When a modal is active, globalInputHandler must pass all events through
	// unchanged (return the event, not nil) so the modal handles input.
	a := &App{
		modalActive:    true,
		treeFilter:     NewFilterOverlay(func(string) {}, func() {}, func() {}),
		taskListFilter: NewFilterOverlay(func(string) {}, func() {}, func() {}),
	}

	cases := []struct {
		name  string
		event *tcell.EventKey
	}{
		{"Tab", tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone)},
		{"BackTab", tcell.NewEventKey(tcell.KeyBacktab, 0, tcell.ModNone)},
		{"q", tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModNone)},
		{"S", tcell.NewEventKey(tcell.KeyRune, 'S', tcell.ModNone)},
		{"slash", tcell.NewEventKey(tcell.KeyRune, '/', tcell.ModNone)},
		{"bracket", tcell.NewEventKey(tcell.KeyRune, '[', tcell.ModNone)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := a.globalInputHandler(tc.event)
			if got == nil {
				t.Errorf("globalInputHandler(%s) with modal active = nil (consumed), want event passed through", tc.name)
			}
		})
	}
}

// ── paneID constants ─────────────────────────────────────────────────────────

func TestPaneIDConstants_AreSequential(t *testing.T) {
	// Verify the pane IDs are sequential starting from 0 — the focus cycling
	// logic depends on this property.
	if paneTree != 0 {
		t.Errorf("paneTree = %d, want 0", paneTree)
	}
	if paneTaskList != 1 {
		t.Errorf("paneTaskList = %d, want 1", paneTaskList)
	}
	if paneTaskDetail != 2 {
		t.Errorf("paneTaskDetail = %d, want 2", paneTaskDetail)
	}
}
