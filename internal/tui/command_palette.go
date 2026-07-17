// Package tui — command palette ("jump to list").
//
// ShowCommandPalette is the K9s-style ':' command bar: type a few letters of
// any list's name, from any workspace/space/folder, and jump straight to its
// tasks without navigating the hierarchy tree. The full cross-workspace list
// index is fetched once (in the background) and cached by HierarchyService,
// so the palette stays instant on repeat opens within the same session.
package tui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/pecigonzalo/clicktui/internal/app"
)

const pageCommandPalette = "command_palette"

// ShowCommandPalette opens the ':' jump-to-list command bar. It calls
// SetModalActive(true) on show and SetModalActive(false) on dismiss. Must be
// called from the UI goroutine.
func ShowCommandPalette(a *App) {
	input := tview.NewInputField()
	input.SetLabel(": ")
	input.SetLabelColor(ColorFilterPrompt)
	input.SetFieldBackgroundColor(tcell.ColorDefault)
	input.SetFieldTextColor(tcell.ColorWhite)

	results := tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false)

	frame := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(input, 1, 0, true).
		AddItem(results, 0, 1, false)
	frame.SetBorder(true).
		SetTitle(" Jump to List ").
		SetBorderColor(ColorBorderFocused).
		SetTitleColor(ColorTitleFocused)

	dismiss := func() {
		a.SetModalActive(false)
		a.pages.RemovePage(pageCommandPalette)
		a.restoreDefaultHelp()
	}

	var (
		allRefs  []app.ListRef
		filtered []app.ListRef
		selected int
	)

	renderResults := func(placeholder string) {
		results.Clear()
		if placeholder != "" {
			results.SetCell(0, 0, tview.NewTableCell(emptyText(placeholder)).
				SetSelectable(false).
				SetExpansion(1))
			return
		}
		if len(filtered) == 0 {
			results.SetCell(0, 0, tview.NewTableCell(emptyText("No matching lists")).
				SetSelectable(false).
				SetExpansion(1))
			return
		}
		for i, ref := range filtered {
			nameText := icons.List + " " + tview.Escape(ref.Name)
			results.SetCell(i, 0, tview.NewTableCell(nameText).
				SetTextColor(ColorText).
				SetExpansion(3))
			results.SetCell(i, 1, tview.NewTableCell(tview.Escape(breadcrumbFor(ref))).
				SetTextColor(ColorTextMuted).
				SetExpansion(2))
		}
		if selected >= len(filtered) {
			selected = len(filtered) - 1
		}
		if selected < 0 {
			selected = 0
		}
		results.Select(selected, 0)
	}

	applyFilter := func(text string) {
		filtered = app.FilterListRefs(allRefs, text)
		selected = 0
		renderResults("")
	}

	jumpTo := func(ref app.ListRef) {
		dismiss()
		a.taskList.LoadTasks(ref.ID, ref.Name)
		a.setFocusPane(paneTaskList)
	}

	input.SetChangedFunc(func(text string) {
		if allRefs != nil {
			applyFilter(text)
		}
	})

	input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			dismiss()
			return nil
		case tcell.KeyDown:
			if selected < len(filtered)-1 {
				selected++
				results.Select(selected, 0)
			}
			return nil
		case tcell.KeyUp:
			if selected > 0 {
				selected--
				results.Select(selected, 0)
			}
			return nil
		case tcell.KeyEnter:
			if selected >= 0 && selected < len(filtered) {
				jumpTo(filtered[selected])
			}
			return nil
		}
		return event
	})

	renderResults("Loading lists…")
	a.footer.SetHelp("↑↓:select", "Enter:jump", "Esc:cancel")
	a.SetModalActive(true)

	w, h := modalScreenSize(a)
	width := clamp(w*70/100, 50, 96)
	height := clamp(h-6, 10, 24)
	modal := centreModal(frame, width, height)

	a.pages.AddPage(pageCommandPalette, modal, true, true)
	a.tviewApp.SetFocus(input)

	go func() {
		refs, err := a.hierarchy.LoadAllLists(a.ctx)
		a.tviewApp.QueueUpdateDraw(func() {
			if err != nil {
				a.logger.Error("command palette: load all lists", "error", err)
				renderResults(fmt.Sprintf("Error loading lists: %v", err))
				return
			}
			allRefs = refs
			applyFilter(input.GetText())
		})
	}()
}

// breadcrumbFor formats a ListRef's parent path for display alongside its
// name in the command palette results (e.g. "Space › Folder").
func breadcrumbFor(ref app.ListRef) string {
	if ref.FolderName == "" {
		return ref.SpaceName
	}
	return ref.SpaceName + " " + icons.Breadcrumb + " " + ref.FolderName
}
