//go:build !windows

package conn

import (
	"os"
	"syscall"

	"davish/internal/apperr"
)

// checkControlSocket refuses anything but a socket that only the current
// user can reach: whoever can open it can use the master's login.
func checkControlSocket(path string) error {
	fi, err := os.Lstat(path)
	if err != nil {
		return apperr.Wrap(apperr.ConnControlSocket, "control socket not found; start ssh -M -S <path> -fN first", err)
	}
	if fi.Mode()&os.ModeSocket == 0 {
		return apperr.New(apperr.ConnControlSocket, "the control path is not a socket")
	}
	if st, ok := fi.Sys().(*syscall.Stat_t); ok && int(st.Uid) != os.Getuid() {
		return apperr.New(apperr.ConnControlSocket, "the control socket belongs to another user")
	}
	if fi.Mode().Perm()&0o077 != 0 {
		return apperr.New(apperr.ConnControlSocket, "the control socket is reachable by other users; chmod 600 it")
	}
	return nil
}
