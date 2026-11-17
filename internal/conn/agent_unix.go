//go:build !windows

package conn

import (
	"errors"
	"io"
	"net"
	"os"
)

// dialAgent connects to the agent named by SSH_AUTH_SOCK.
func dialAgent() (io.ReadWriteCloser, error) {
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		return nil, errors.New("SSH_AUTH_SOCK is not set")
	}
	return net.Dial("unix", sock) //nolint:gosec // SSH_AUTH_SOCK is how the user's agent is found
}
