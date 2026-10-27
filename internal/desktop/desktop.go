// Package desktop talks to the user's own machine: showing a local file in
// the system file manager. It never touches a server.
package desktop

import (
	"context"
	"os/exec"
	"path/filepath"
	"runtime"
)

// Reveal shows path in Finder, Explorer or the Linux file manager, with the
// file selected where the platform allows it. path must be absolute and
// already validated by the caller; it is passed as one argument, never
// through a shell.
func Reveal(ctx context.Context, path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.CommandContext(ctx, "open", "-R", path) //nolint:gosec // validated path, no shell
	case "windows":
		cmd = exec.CommandContext(ctx, "explorer", "/select,"+path) //nolint:gosec // validated path, no shell
	default:
		// No portable way to select a file; open its folder.
		cmd = exec.CommandContext(ctx, "xdg-open", filepath.Dir(path)) //nolint:gosec // validated path, no shell
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	// The file manager stays open on its own; only reap the launcher.
	go cmd.Wait() //nolint:errcheck // explorer exits 1 even when it worked
	return nil
}
