//go:build !windows

package conn

import (
	"os/exec"
	"syscall"
)

// proxyShell is how OpenSSH runs a ProxyCommand: through the system shell,
// so pipes, redirections and arguments behave exactly as they do in ssh.
func proxyShell() (name, flag string) { return "sh", "-c" }

// inOwnGroup puts the command in a process group of its own, so that
// killProxy can end it and everything it started. dash, the shell of Debian
// and Ubuntu, does not exec the last command of `sh -c`: it forks it, and
// killing only the shell left the tunnel client running.
func inOwnGroup(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} }

// killProxy ends the proxy command and everything it started.
func killProxy(cmd *exec.Cmd) {
	syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) //nolint:errcheck,gosec // the group may be gone already
	cmd.Process.Kill()                              //nolint:errcheck,gosec // ErrProcessDone is fine
}
