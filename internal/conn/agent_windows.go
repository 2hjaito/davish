//go:build windows

package conn

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
)

// InitAgentEnv is a no-op on Windows, where GUI apps inherit user environment
// variables directly and the default agent listens on a named pipe.
func InitAgentEnv(*slog.Logger) {}

// windowsAgentPipe is where the OpenSSH Authentication Agent service of
// Windows listens, and the 1Password and Bitwarden agents too. Windows sets
// no SSH_AUTH_SOCK for it.
const windowsAgentPipe = `\\.\pipe\openssh-ssh-agent`

// AgentProxyFlag starts Davish as a relay between its stdin/stdout and an
// agent's named pipe, with no window (see dialAgentPipe).
const AgentProxyFlag = "--ssh-agent-proxy"

// agentProxyReady is the byte the relay writes once the pipe is open, so the
// app can tell "the agent is there" from "the relay could not open it".
const agentProxyReady = 0x01

// dialAgent connects to the agent: SSH_AUTH_SOCK first when it is set (a
// named pipe, or a Unix socket such as WSL's), else the Windows agent's
// pipe.
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
	c, err := dialAgentPipe(pipe)
	if err != nil && unixErr != nil {
		return nil, fmt.Errorf("SSH_AUTH_SOCK: %w; default pipe: %w", unixErr, err)
	}
	return c, err
}

// dialAgentPipe opens the agent's pipe through a child Davish (#332).
//
// 1Password on Windows does not name the process that opens its pipe: it
// walks up from that process's parent and asks "Allow <app>" for the first
// app it finds below Explorer. ssh.exe started from a terminal is named after
// the terminal. Davish started from Explorer has no parent but Explorer, so
// 1Password logs "process tree is empty", shows no prompt and lets the
// request time out. With a child Davish holding the pipe, the parent it finds
// is Davish itself.
//
// If the child cannot start, the pipe is opened here as before.
func dialAgentPipe(pipe string) (io.ReadWriteCloser, error) {
	if c, err := startAgentProxy(pipe); err == nil || !errors.Is(err, errProxyStart) {
		return c, err
	}
	return os.OpenFile(pipe, os.O_RDWR, 0) //nolint:gosec // the agent's named pipe
}

var errProxyStart = errors.New("ssh agent relay did not start")

// agentProxyExe is the program started as the relay; tests replace it.
var agentProxyExe = os.Executable

func startAgentProxy(pipe string) (io.ReadWriteCloser, error) {
	exe, err := agentProxyExe()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errProxyStart, err)
	}
	cmd := exec.Command(exe, AgentProxyFlag, pipe) //nolint:gosec // Davish itself
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errProxyStart, err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errProxyStart, err)
	}
	var stderr strings.Builder
	cmd.Stderr = &limitedWriter{w: &stderr, n: 512}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("%w: %w", errProxyStart, err)
	}
	p := &agentProxy{cmd: cmd, in: stdin, out: bufio.NewReader(stdout)}

	// The relay answers at once: the ready byte, or an error and exit.
	ready := make(chan error, 1)
	go func() {
		b, err := p.out.ReadByte()
		switch {
		case err == nil && b == agentProxyReady:
			ready <- nil
		case err == nil:
			ready <- fmt.Errorf("%w: unexpected byte %#x", errProxyStart, b)
		default:
			_ = cmd.Wait()
			msg := strings.TrimSpace(stderr.String())
			if msg == "" {
				// Not a relay at all (an old build, or the flag unknown).
				ready <- fmt.Errorf("%w: %w", errProxyStart, err)
				return
			}
			ready <- errors.New(msg)
		}
	}()
	select {
	case err := <-ready:
		if err != nil {
			p.kill()
			return nil, err
		}
		return p, nil
	case <-time.After(agentWait()):
		p.kill()
		return nil, fmt.Errorf("%w: no answer in %s", errProxyStart, agentWait())
	}
}

// createNoWindow keeps Windows from giving the relay a console window.
const createNoWindow = 0x08000000

// agentProxy is the app's end of the relay: writes go to the child's stdin,
// reads come from its stdout.
type agentProxy struct {
	cmd  *exec.Cmd
	in   io.WriteCloser
	out  *bufio.Reader
	once sync.Once
}

func (p *agentProxy) Read(b []byte) (int, error)  { return p.out.Read(b) }
func (p *agentProxy) Write(b []byte) (int, error) { return p.in.Write(b) }

func (p *agentProxy) Close() error {
	p.kill()
	return nil
}

func (p *agentProxy) kill() {
	p.once.Do(func() {
		_ = p.in.Close()
		_ = p.cmd.Process.Kill()
		go p.cmd.Wait() //nolint:errcheck // reaps the child; its exit status says nothing
	})
}

type limitedWriter struct {
	w io.Writer
	n int
}

func (l *limitedWriter) Write(b []byte) (int, error) {
	if l.n > 0 {
		k := min(len(b), l.n)
		_, _ = l.w.Write(b[:k])
		l.n -= k
	}
	return len(b), nil
}

// maxAgentMessage bounds one agent message; x/crypto's client allows 16 MiB.
const maxAgentMessage = 16 << 20

// relayAgent passes one agent message at a time: a request from the app to
// the agent, then the agent's answer back. A pipe opened by os.OpenFile is
// synchronous, so a read pending on it would block a write from another
// goroutine; turn by turn, the pipe is never read and written at once. The
// agent protocol is request then answer, so this loses nothing.
func relayAgent(app io.Reader, toApp io.Writer, agentPipe io.ReadWriter) error {
	for {
		if err := copyAgentMessage(agentPipe, app); err != nil {
			return err
		}
		if err := copyAgentMessage(toApp, agentPipe); err != nil {
			return err
		}
	}
}

// copyAgentMessage copies one length-prefixed agent message.
func copyAgentMessage(dst io.Writer, src io.Reader) error {
	var head [4]byte
	if _, err := io.ReadFull(src, head[:]); err != nil {
		return err
	}
	n := uint32(head[0])<<24 | uint32(head[1])<<16 | uint32(head[2])<<8 | uint32(head[3])
	if n > maxAgentMessage {
		return fmt.Errorf("ssh agent message of %d bytes", n)
	}
	buf := make([]byte, 4+int(n))
	copy(buf, head[:])
	if _, err := io.ReadFull(src, buf[4:]); err != nil {
		return err
	}
	_, err := dst.Write(buf)
	return err
}

// RunAgentProxy is the relay's main: it opens the pipe, says so on stdout,
// then passes messages until either side closes. It returns false when args
// do not ask for it, so main goes on to start the app.
func RunAgentProxy(args []string) bool {
	if len(args) != 3 || args[1] != AgentProxyFlag {
		return false
	}
	pipe := args[2]
	if !isPipe(pipe) {
		fmt.Fprintf(os.Stderr, "not a named pipe: %s\n", pipe)
		os.Exit(2)
	}
	c, err := os.OpenFile(pipe, os.O_RDWR, 0) //nolint:gosec // the agent's named pipe, given by the parent Davish
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if _, err := os.Stdout.Write([]byte{agentProxyReady}); err != nil {
		os.Exit(1)
	}
	_ = relayAgent(os.Stdin, os.Stdout, c) // ends when the app closes its side
	_ = c.Close()
	os.Exit(0)
	return true
}

func isPipe(path string) bool {
	return strings.HasPrefix(path, `\\.\pipe\`) ||
		strings.HasPrefix(path, `//./pipe/`) ||
		strings.HasPrefix(path, `\\?\pipe\`) ||
		strings.HasPrefix(path, `//?/pipe/`)
}
