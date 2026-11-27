//go:build !windows

package conn

// proxyShell is how OpenSSH runs a ProxyCommand: through the system shell,
// so pipes, redirections and arguments behave exactly as they do in ssh.
func proxyShell() (name, flag string) { return "sh", "-c" }
