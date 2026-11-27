//go:build windows

package conn

import "os/exec"

// proxyShell is the closest Windows has to ssh's "/bin/sh -c": run the
// command through cmd.exe, as Windows OpenSSH itself does.
func proxyShell() (name, flag string) { return "cmd", "/c" }

// detach does nothing on Windows.
func detach(*exec.Cmd) {}

// killGroup does nothing on Windows: Close kills the cmd.exe that runs the
// proxy command, and a program it started may outlive it. Ending a whole
// tree there needs a job object, which is not done here and has not been
// tried.
func killGroup(*exec.Cmd) {}

// groupEmpty is always true on Windows: there is no group to signal.
func groupEmpty(*exec.Cmd) bool { return true }
