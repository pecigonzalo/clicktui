// Package tui — URL opening helper.
//
// openURL launches the user's default browser/handler for a URL using
// platform-native commands (open on macOS, xdg-open on Linux, cmd /c start
// on Windows). This avoids adding an external dependency.
package tui

import (
	"fmt"
	"os/exec"
	"runtime"
)

// openURL opens url in the user's default browser.
// Returns an error when the platform is unsupported or the command fails to
// start.
func openURL(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open url: %w", err)
	}
	return nil
}
