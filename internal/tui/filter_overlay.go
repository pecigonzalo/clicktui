// Package tui — filter overlay component.
//
// FilterOverlay wraps a tview.InputField to provide an inline filter bar that
// can be shown at the bottom of any pane. It supports two states:
//   - Editing: the input field has focus and the user is typing.
//   - Applied: the filter text is active but focus has returned to the pane.
//
// The overlay fires callbacks as the user types (onChanged) and when the
// filter is cleared via Esc (onCleared).
package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// FilterOverlay is a reusable inline filter input that sits at the bottom of
// a pane.
type FilterOverlay struct {
	input *tview.InputField

	// onChanged is called with the current text every time the user types.
	onChanged func(text string)
	// onCleared is called when the user presses Esc to dismiss the filter.
	onCleared func()
	// returnFocus is called to return focus to the owning pane.
	returnFocus func()

	// editing is true when the input field is focused and accepting keystrokes.
	editing bool
	// active is true when a non-empty filter text is applied (editing or not).
	active bool
}

// NewFilterOverlay creates a FilterOverlay. The caller must supply callbacks:
//   - onChanged: fired on every keystroke with the current input text.
//   - onCleared: fired when Esc clears the filter.
//   - returnFocus: called to return focus to the owning pane.
func NewFilterOverlay(onChanged func(string), onCleared func(), returnFocus func()) *FilterOverlay {
	fo := &FilterOverlay{
		onChanged:   onChanged,
		onCleared:   onCleared,
		returnFocus: returnFocus,
	}

	input := tview.NewInputField()
	input.SetLabel("/ ")
	input.SetLabelColor(ColorFilterPrompt)
	input.SetFieldBackgroundColor(tcell.ColorDefault)
	input.SetFieldTextColor(tcell.ColorWhite)

	input.SetChangedFunc(func(text string) {
		fo.active = text != ""
		if fo.onChanged != nil {
			fo.onChanged(text)
		}
	})

	input.SetInputCapture(fo.inputCapture)
	fo.input = input
	return fo
}

// InputField returns the underlying tview.InputField for layout embedding.
func (fo *FilterOverlay) InputField() *tview.InputField {
	return fo.input
}

// Show activates the overlay in editing mode.
func (fo *FilterOverlay) Show() {
	fo.editing = true
	fo.active = fo.input.GetText() != ""
}

// Hide deactivates the overlay entirely, clearing text and state.
func (fo *FilterOverlay) Hide() {
	fo.editing = false
	fo.active = false
	fo.input.SetText("")
}

// IsActive reports whether a filter is currently applied (editing or applied).
func (fo *FilterOverlay) IsActive() bool {
	return fo.active
}

// IsEditing reports whether the overlay is in editing mode (input focused).
func (fo *FilterOverlay) IsEditing() bool {
	return fo.editing
}

// Text returns the current filter text.
func (fo *FilterOverlay) Text() string {
	return fo.input.GetText()
}

// SetApplied transitions from editing to applied state (filter stays active
// but input loses focus).
func (fo *FilterOverlay) SetApplied() {
	fo.editing = false
	fo.active = fo.input.GetText() != ""
}

func (fo *FilterOverlay) inputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEnter:
		// Keep filter active, return focus to pane.
		fo.SetApplied()
		if fo.returnFocus != nil {
			fo.returnFocus()
		}
		return nil
	case tcell.KeyEscape:
		// Clear filter and return focus.
		fo.Hide()
		if fo.onCleared != nil {
			fo.onCleared()
		}
		if fo.returnFocus != nil {
			fo.returnFocus()
		}
		return nil
	}
	return event
}
