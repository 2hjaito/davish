//go:build windows

package conn

// proxyShell is the closest Windows has to ssh's "/bin/sh -c": run the
// command through cmd.exe, as Windows OpenSSH itself does.
func proxyShell() (name, flag string) { return "cmd", "/c" }
