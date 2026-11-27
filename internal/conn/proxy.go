// ssh's ProxyCommand: the transport of an SSH connection carried by a
// command's stdin and stdout instead of a TCP socket. This is what reaches
// servers hidden behind a WebSocket relay, an HTTP tunnel or a host an ssh
// client cannot route to directly.

package conn

import (
	"errors"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"davish/internal/store"
)

// expandProxyTokens applies the substitution OpenSSH does on a ProxyCommand:
// %h and %n are the host, %r the login, %p the port and %% a literal percent.
// A token this build cannot compute yet is left as typed: the command then
// fails in the user's own shell, visibly, instead of connecting elsewhere.
func expandProxyTokens(cmd string, srv store.Server) string {
	var b strings.Builder
	for i := 0; i < len(cmd); i++ {
		if cmd[i] != '%' || i+1 >= len(cmd) {
			b.WriteByte(cmd[i])
			continue
		}
		i++
		switch cmd[i] {
		case '%':
			b.WriteByte('%')
		case 'h', 'n':
			b.WriteString(srv.Host)
		case 'r':
			b.WriteString(srv.User)
		case 'p':
			b.WriteString(strconv.Itoa(srv.Port))
		default:
			b.WriteByte('%')
			b.WriteByte(cmd[i])
		}
	}
	return b.String()
}

// maxStderr keeps a proxy's dying words small; tunnel clients rarely say
// more than a line before exiting, and nothing is ever read after that.
const maxStderr = 2048

// dialProxy starts srv's ProxyCommand and returns its stdin/stdout as a
// connection, ready for the SSH handshake. The command runs unconfined and
// as the user, exactly as ssh would run it from ~/.ssh/config. The process
// lives until Close, which is how the handshake timeout and every
// reconnect end a tunnel nobody needs any more.
func dialProxy(srv store.Server) (*proxyConn, error) {
	name, flag := proxyShell()
	cmd := exec.Command(name, flag, expandProxyTokens(srv.ProxyCommand, srv)) //nolint:gosec // the user's own ProxyCommand, as ssh runs it
	pc := &proxyConn{target: net.JoinHostPort(srv.Host, strconv.Itoa(srv.Port))}

	// Handled pipes, not StdinPipe/StdoutPipe: those are closed by Wait,
	// racing the reads of a proxy that is still alive.
	inR, inW, err := os.Pipe() // we write to inW, the command reads inR
	if err != nil {
		return nil, err
	}
	outR, outW, err := os.Pipe() // the command writes outW, we read outR
	if err != nil {
		inR.Close() //nolint:errcheck,gosec // already failing
		inW.Close() //nolint:errcheck,gosec // already failing
		return nil, err
	}
	cmd.Stdin, cmd.Stdout = inR, outW
	pc.cmd, pc.w, pc.r, pc.errs = cmd, inW, outR, &stderrBuffer{}
	cmd.Stderr = pc.errs
	if err := cmd.Start(); err != nil {
		inR.Close()  //nolint:errcheck,gosec // already failing
		inW.Close()  //nolint:errcheck,gosec // already failing
		outR.Close() //nolint:errcheck,gosec // already failing
		outW.Close() //nolint:errcheck,gosec // already failing
		return nil, err
	}
	go pc.wait()
	return pc, nil
}

// proxyConn adapts a living ProxyCommand process to net.Conn, the shape the
// SSH handshake needs.
type proxyConn struct {
	cmd       *exec.Cmd
	target    string   // the server's host:port, what the library reports it as
	w         *os.File // to the process's stdin
	r         *os.File // from the process's stdout
	closeOnce sync.Once

	// mu guards exited and waitErr.
	mu      sync.Mutex
	exited  bool
	waitErr error
	errs    *stderrBuffer
}

// wait runs in its own goroutine: it reaps the process and remembers how it
// ended, so a command that dies before the handshake can say why.
func (pc *proxyConn) wait() {
	err := pc.cmd.Wait()
	pc.mu.Lock()
	pc.exited, pc.waitErr = true, err
	pc.mu.Unlock()
}

// explain reports why a proxy that exited before answering cannot; a
// process that is still running has nothing to say.
func (pc *proxyConn) explain() string {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	if !pc.exited {
		return ""
	}
	if said := pc.errs.String(); said != "" {
		return "the proxy command said on stderr: " + said
	}
	var exit *exec.ExitError
	switch {
	case errors.As(pc.waitErr, &exit):
		return "the proxy command exited with " + exit.String()
	case pc.waitErr != nil:
		return "the proxy command could not run: " + pc.waitErr.Error()
	}
	return "the proxy command exited before the server answered"
}

func (pc *proxyConn) Read(b []byte) (int, error)  { return pc.r.Read(b) }
func (pc *proxyConn) Write(b []byte) (int, error) { return pc.w.Write(b) }

// Close ends the process and both pipes. Closing stdin first lets a proxy
// that follows ssh's convention (exit on EOF) finish writing; the kill
// then only collects one that ignored it.
func (pc *proxyConn) Close() error {
	pc.closeOnce.Do(func() {
		pc.w.Close()          //nolint:errcheck,gosec // the only writer
		pc.r.Close()          //nolint:errcheck,gosec // the only reader
		pc.cmd.Process.Kill() //nolint:errcheck,gosec // ErrProcessDone is fine
	})
	return nil
}

// RemoteAddr reports the server's host:port — what the known-hosts check
// matches on — and LocalAddr says how the machine really talks: through the
// proxy, not through the network address it stands for.
func (pc *proxyConn) RemoteAddr() net.Addr { return proxyAddr{pc.target} }
func (pc *proxyConn) LocalAddr() net.Addr  { return proxyAddr{"proxy command"} }

// The SSH handshake does not use deadlines: its timeouts come from the
// manager, which closes the whole connection instead.
func (pc *proxyConn) SetDeadline(_ time.Time) error      { return nil }
func (pc *proxyConn) SetReadDeadline(_ time.Time) error  { return nil }
func (pc *proxyConn) SetWriteDeadline(_ time.Time) error { return nil }

// proxyAddr is the network the proxy connection claims: "proxy", with the
// address itself naming what it stands for.
type proxyAddr struct{ addr string }

func (proxyAddr) Network() string  { return "proxy" }
func (a proxyAddr) String() string { return a.addr }

// stderrBuffer keeps a proxy's last words for the connection error, but
// caps them, so a chatty proxy cannot outgrow its welcome.
type stderrBuffer struct {
	mu  sync.Mutex
	buf []byte
}

func (b *stderrBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if room := maxStderr - len(b.buf); room > 0 {
		b.buf = append(b.buf, p[:min(room, len(p))]...)
	}
	return len(p), nil
}

func (b *stderrBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return strings.TrimSpace(string(b.buf))
}

// the handshake takes a net.Conn; prove the adapter is one.
var _ net.Conn = (*proxyConn)(nil)
