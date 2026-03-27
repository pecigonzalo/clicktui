package tui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFilterOverlay(t *testing.T) {
	changedCalls := 0
	clearedCalls := 0
	focusCalls := 0
	lastText := ""

	fo := NewFilterOverlay(
		func(text string) {
			changedCalls++
			lastText = text
		},
		func() { clearedCalls++ },
		func() { focusCalls++ },
	)

	require.NotNil(t, fo)
	require.NotNil(t, fo.InputField())
	assert.False(t, fo.IsEditing())
	assert.False(t, fo.IsActive())
	assert.Equal(t, "", fo.Text())

	fo.InputField().SetText("status:open")
	assert.Equal(t, "status:open", lastText)
	assert.GreaterOrEqual(t, changedCalls, 1)
	assert.True(t, fo.IsActive())
	assert.Equal(t, 0, clearedCalls)
	assert.Equal(t, 0, focusCalls)
}

func TestFilterOverlay_ShowHideIsEditingState(t *testing.T) {
	fo := NewFilterOverlay(nil, nil, nil)

	assert.False(t, fo.IsEditing())
	assert.False(t, fo.IsActive())

	fo.Show()
	assert.True(t, fo.IsEditing())
	assert.False(t, fo.IsActive())

	fo.InputField().SetText("abc")
	assert.True(t, fo.IsActive())

	fo.Hide()
	assert.False(t, fo.IsEditing())
	assert.False(t, fo.IsActive())
	assert.Equal(t, "", fo.Text())
}

func TestFilterOverlay_EnterAppliesAndReturnsFocus(t *testing.T) {
	focusCalls := 0
	fo := NewFilterOverlay(nil, nil, func() { focusCalls++ })
	fo.InputField().SetText("foo")
	fo.Show()

	fo.InputField().InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), nil)

	assert.False(t, fo.IsEditing())
	assert.True(t, fo.IsActive())
	assert.Equal(t, 1, focusCalls)
}

func TestFilterOverlay_EscapeClearsAndReturnsFocus(t *testing.T) {
	focusCalls := 0
	clearedCalls := 0
	fo := NewFilterOverlay(nil, func() { clearedCalls++ }, func() { focusCalls++ })
	fo.InputField().SetText("foo")
	fo.Show()

	fo.InputField().InputHandler()(tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone), nil)

	assert.False(t, fo.IsEditing())
	assert.False(t, fo.IsActive())
	assert.Equal(t, "", fo.Text())
	assert.Equal(t, 1, clearedCalls)
	assert.Equal(t, 1, focusCalls)
}

func TestFilterOverlay_SetApplied(t *testing.T) {
	fo := NewFilterOverlay(nil, nil, nil)
	fo.InputField().SetText("kind:list")
	fo.Show()

	fo.SetApplied()

	assert.False(t, fo.IsEditing())
	assert.True(t, fo.IsActive())
	assert.Equal(t, "kind:list", fo.Text())
}
