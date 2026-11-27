//go:build !windows

package conn

import (
	"os/exec"
	"syscall"
)

// proxyShell is how OpenSSH runs a ProxyCommand: through the system shell,
// so pipes, redirections and arguments behave exactly as they do in ssh.
func proxyShell() (name, flag string) { return "sh", "-c" }

// detach starts the command in a session of its own. That gives it a process
// group that killGroup can end together with everything the shell started:
// dash, the shell of Debian and Ubuntu, does not exec the last command of
// `sh -c`, it forks it, and killing only the shell left the tunnel client
// running. A session and not just a group (Setpgid), because a proxy that
// reads /dev/tty (ssh asking for a password, when Davish was started from a
// terminal) would be stopped by SIGTTIN as a background group and hang until
// the handshake timeout; with no controlling terminal it fails at once.
func detach(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true} }

// killGroup ends every process in the command's group.
func killGroup(cmd *exec.Cmd) {
	syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) //nolint:errcheck,gosec // the group may be gone already
}

// groupEmpty tells whether the command's group has no process left. Once it
// has none, its number is free for the system to give to an unrelated
// process, and signalling it could hit that one, so it must not be signalled
// again (POSIX keeps the number reserved only while the group exists).
func groupEmpty(cmd *exec.Cmd) bool {
	return syscall.Kill(-cmd.Process.Pid, 0) == syscall.ESRCH
}
