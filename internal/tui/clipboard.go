// Package tui — clipboard helper.
//
// writeClipboard writes text to the system clipboard using platform-native
// tools (pbcopy on macOS, xclip/xsel on Linux, clip on Windows) via os/exec.
// This avoids adding an external dependency for a simple one-way write.
package tui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// writeClipboard writes s to the system clipboard.
// Returns an error when no suitable clipboard tool is found or the write fails.
func writeClipboard(s string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		// Prefer xclip; fall back to xsel when xclip is not installed.
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else if _, err := exec.LookPath("xsel"); err == nil {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		} else {
			return fmt.Errorf("clipboard: neither xclip nor xsel found; install one to enable copy")
		}
	case "windows":
		cmd = exec.Command("clip")
	default:
		return fmt.Errorf("clipboard: unsupported platform %s", runtime.GOOS)
	}
	cmd.Stdin = strings.NewReader(s)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("clipboard write: %w", err)
	}
	return nil
}
