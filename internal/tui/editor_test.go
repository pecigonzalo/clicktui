// Package tui — unit tests for external editor helpers.
package tui

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveEditor_Visual(t *testing.T) {
	t.Setenv("VISUAL", "nvim")
	t.Setenv("EDITOR", "")
	assert.Equal(t, []string{"nvim"}, resolveEditor())
}

func TestResolveEditor_Editor(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "nano")
	assert.Equal(t, []string{"nano"}, resolveEditor())
}

func TestResolveEditor_VisualTakesPrecedence(t *testing.T) {
	t.Setenv("VISUAL", "vim")
	t.Setenv("EDITOR", "nano")
	assert.Equal(t, []string{"vim"}, resolveEditor())
}

func TestResolveEditor_NeitherSet(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "")
	assert.Nil(t, resolveEditor())
}

func TestResolveEditor_MultiWordCommand(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "code --wait")
	assert.Equal(t, []string{"code", "--wait"}, resolveEditor())
}

func TestOpenInEditor_Unchanged(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-script editor fixture is unix-only")
	}

	editor := writeEditorScript(t, "#!/bin/sh\nexit 0\n")
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", editor)

	tviewApp, stop := startTestTviewApp(t)
	defer stop()

	updated, err := openInEditor(tviewApp, "same content")
	assert.Empty(t, updated)
	assert.ErrorIs(t, err, errUnchanged)
}

func TestOpenInEditor_Changed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-script editor fixture is unix-only")
	}

	editor := writeEditorScript(t, "#!/bin/sh\nprintf 'new content' > \"$1\"\n")
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", editor)

	tviewApp, stop := startTestTviewApp(t)
	defer stop()

	updated, err := openInEditor(tviewApp, "same content")
	require.NoError(t, err)
	assert.Equal(t, "new content", updated)
}

func TestOpenInEditor_EditorError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-script editor fixture is unix-only")
	}

	editor := writeEditorScript(t, "#!/bin/sh\nexit 7\n")
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", editor)
	t.Setenv("TMPDIR", t.TempDir())

	tviewApp, stop := startTestTviewApp(t)
	defer stop()

	updated, err := openInEditor(tviewApp, "same content")
	assert.Empty(t, updated)
	require.Error(t, err)
	assert.False(t, errors.Is(err, errUnchanged))

	entries, readErr := os.ReadDir(os.Getenv("TMPDIR"))
	require.NoError(t, readErr)
	for _, entry := range entries {
		assert.NotContains(t, entry.Name(), "clicktui-")
	}
}

func writeEditorScript(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "editor.sh")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o755))
	return path
}

func startTestTviewApp(t *testing.T) (*tview.Application, func()) {
	t.Helper()

	tviewApp := tview.NewApplication()
	screen := tcell.NewSimulationScreen("UTF-8")
	tviewApp.SetScreen(screen)
	tviewApp.SetRoot(tview.NewBox(), true)

	runErr := make(chan error, 1)
	go func() {
		runErr <- tviewApp.Run()
	}()

	ready := make(chan struct{})
	tviewApp.QueueUpdateDraw(func() {
		close(ready)
	})
	<-ready

	return tviewApp, func() {
		tviewApp.Stop()
		require.NoError(t, <-runErr)
	}
}
