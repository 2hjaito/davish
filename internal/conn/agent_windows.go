//go:build windows

package conn

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"strings"
)

// InitAgentEnv is a no-op on Windows, where GUI apps inherit user environment
// variables directly and the default agent listens on a named pipe.
func InitAgentEnv(*slog.Logger) {}

// windowsAgentPipe is where the OpenSSH Authentication Agent service of
// Windows listens, and the 1Password and Bitwarden agents too. Windows sets
// no SSH_AUTH_SOCK for it.
const windowsAgentPipe = `\\.\pipe\openssh-ssh-agent`

// dialAgent connects to the agent: SSH_AUTH_SOCK first when it is set (a
// named pipe, or a Unix socket such as WSL's), else the Windows agent's
// pipe. A named pipe opens as a file, so no extra package is needed.
func dialAgent() (io.ReadWriteCloser, error) {
	pipe := windowsAgentPipe
	var unixErr error
	if sock := strings.TrimSpace(os.Getenv("SSH_AUTH_SOCK")); sock != "" {
		if isPipe(sock) {
			pipe = sock
		} else {
			c, err := net.DialTimeout("unix", expandHome(sock), agentWait()) //nolint:gosec // SSH_AUTH_SOCK is how the user's agent is found
			if err == nil {
				return c, nil
			}
			unixErr = err
		}
	}
	c, err := os.OpenFile(pipe, os.O_RDWR, 0) //nolint:gosec // the agent's named pipe
	if err != nil && unixErr != nil {
		return nil, fmt.Errorf("SSH_AUTH_SOCK: %w; default pipe: %w", unixErr, err)
	}
	return c, err
}

func isPipe(path string) bool {
	return strings.HasPrefix(path, `\\.\pipe\`) ||
		strings.HasPrefix(path, `//./pipe/`) ||
		strings.HasPrefix(path, `\\?\pipe\`) ||
		strings.HasPrefix(path, `//?/pipe/`)
}
