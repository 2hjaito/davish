//go:build windows

package conn

import "os/exec"

// proxyShell is the closest Windows has to ssh's "/bin/sh -c": run the
// command through cmd.exe, as Windows OpenSSH itself does.
func proxyShell() (name, flag string) { return "cmd", "/c" }

// inOwnGroup does nothing on Windows.
func inOwnGroup(*exec.Cmd) {}

// killProxy ends the cmd.exe that runs the proxy command. A program it
// started may outlive it: ending a whole tree on Windows needs a job object,
// which is not done here and has not been tried.
func killProxy(cmd *exec.Cmd) { cmd.Process.Kill() } //nolint:errcheck,gosec // ErrProcessDone is fine
