// Package tui — single/multi-select list modal.
package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const pageSelectModal = "select_modal"

// SelectOption is one item in a SelectModal.
type SelectOption struct {
	// Label is the display text shown in the list.
	Label string
	// Value is the opaque string returned to OnSubmit when this option is chosen.
	Value string
	// Selected marks this option as pre-selected for multi-select mode.
	Selected bool
}

// SelectModalConfig configures a single or multi-select list modal.
type SelectModalConfig struct {
	// Title is displayed in the modal border.
	Title string
	// Options is the list of items to choose from.
	Options []SelectOption
	// Multi enables multi-select mode: Space toggles, Enter confirms all selections.
	Multi bool
	// OnSubmit is called with the values of all selected items.
	OnSubmit func(selected []string)
	// OnCancel is called when the user presses Esc.
	OnCancel func()
}

// ShowSelectModal presents a centred list-selection dialog over the current
// layout.  In single-select mode, Enter immediately confirms the highlighted
// item.  In multi-select mode, Space toggles items and Enter confirms.  Esc
// cancels.  It calls SetModalActive(true) on show and SetModalActive(false)
// when dismissed.
func ShowSelectModal(a *App, cfg SelectModalConfig) {
	dismiss := func() {
		a.SetModalActive(false)
		a.pages.RemovePage(pageSelectModal)
		a.restoreDefaultHelp()
	}

	// selected tracks which option indices are chosen (multi-select).
	selected := make([]bool, len(cfg.Options))
	for i, opt := range cfg.Options {
		selected[i] = opt.Selected
	}

	list := tview.NewList()
	list.SetBorder(false)
	list.ShowSecondaryText(false)
	list.SetMainTextStyle(tcell.StyleDefault.Foreground(ColorDetailValue))
	list.SetSelectedStyle(tcell.StyleDefault.Background(ColorBorderFocused).Foreground(tcell.ColorWhite))

	// renderItems rebuilds the list rows, updating selection indicators.
	// Declared as a variable so it can be referenced recursively from closures.
	var renderItems func()
	renderItems = func() {
		list.Clear()
		for i, opt := range cfg.Options {

			var prefix string
			if cfg.Multi {
				if selected[i] {
					prefix = tagColor(ColorBorderFocused) + "[x] " + "[-]"
				} else {
					prefix = tagColor(ColorTextMuted) + "[ ] " + "[-]"
				}
			}

			list.AddItem(prefix+tview.Escape(opt.Label), "", 0, func() {
				if cfg.Multi {
					// In multi-select, activation via mouse/shortcut behaves
					// like Space — toggling rather than confirming.
					selected[i] = !selected[i]
					renderItems()
					return
				}
				// Single-select: confirm immediately.
				dismiss()
				if cfg.OnSubmit != nil {
					cfg.OnSubmit([]string{opt.Value})
				}
			})
		}
	}
	renderItems()

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			dismiss()
			if cfg.OnCancel != nil {
				cfg.OnCancel()
			}
			return nil
		case tcell.KeyEnter:
			if cfg.Multi {
				// Confirm all currently selected items.
				var vals []string
				for i, sel := range selected {
					if sel {
						vals = append(vals, cfg.Options[i].Value)
					}
				}
				dismiss()
				if cfg.OnSubmit != nil {
					cfg.OnSubmit(vals)
				}
				return nil
			}
			// Single-select: Enter is handled by the list's item activation.
			return event
		case tcell.KeyRune:
			if event.Rune() == ' ' && cfg.Multi {
				idx := list.GetCurrentItem()
				if idx >= 0 && idx < len(selected) {
					selected[idx] = !selected[idx]
					renderItems()
					// Restore cursor position after re-render.
					list.SetCurrentItem(idx)
				}
				return nil
			}
		}
		return event
	})

	frame := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(list, 0, 1, true)
	frame.SetBorder(true).
		SetTitle(" " + cfg.Title + " ").
		SetBorderColor(ColorBorderFocused).
		SetTitleColor(ColorTitleFocused)

	// Height: cap at 20 rows; allow at least 6.
	height := len(cfg.Options) + 4 // +4 for borders + padding
	height = min(height, 20)
	height = max(height, 6)

	modal := centreModal(frame, 48, height)

	if cfg.Multi {
		a.footer.SetHelp("Space:toggle", "Enter:confirm", "Esc:cancel")
	} else {
		a.footer.SetHelp("Enter:select", "Esc:cancel")
	}
	a.SetModalActive(true)
	a.pages.AddPage(pageSelectModal, modal, true, true)
	a.tviewApp.SetFocus(list)
}

// SelectedValues returns the values from opts where Selected is true.
// This is a pure helper suitable for use in tests.
func SelectedValues(opts []SelectOption) []string {
	var out []string
	for _, o := range opts {
		if o.Selected {
			out = append(out, o.Value)
		}
	}
	return out
}
