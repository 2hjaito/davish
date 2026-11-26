//go:build windows

package conn

import "davish/internal/apperr"

// checkControlSocket always fails: OpenSSH for Windows has no ControlMaster.
func checkControlSocket(string) error {
	return apperr.New(apperr.ConnControlSocket, "control sockets are not supported on Windows")
}
