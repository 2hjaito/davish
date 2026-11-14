//go:build windows

package conn

import (
	"io"
	"net"
	"os"
	"strings"
)

// windowsAgentPipe is where the OpenSSH Authentication Agent service of
// Windows listens, and the 1Password and Bitwarden agents too. Windows sets
// no SSH_AUTH_SOCK for it.
const windowsAgentPipe = `\\.\pipe\openssh-ssh-agent`

// dialAgent connects to the agent: SSH_AUTH_SOCK first when it is set (a
// named pipe, or a Unix socket such as WSL's), else the Windows agent's
// pipe. A named pipe opens as a file, so no extra package is needed.
func dialAgent() (io.ReadWriteCloser, error) {
	pipe := windowsAgentPipe
	if sock := os.Getenv("SSH_AUTH_SOCK"); sock != "" {
		if isPipe(sock) {
			pipe = sock
		} else if c, err := net.Dial("unix", sock); err == nil { //nolint:gosec // SSH_AUTH_SOCK is how the user's agent is found
			return c, nil
		}
	}
	return os.OpenFile(pipe, os.O_RDWR, 0) //nolint:gosec // the agent's named pipe
}

func isPipe(path string) bool {
	return strings.HasPrefix(path, `\\.\pipe\`) || strings.HasPrefix(path, `//./pipe/`)
}
