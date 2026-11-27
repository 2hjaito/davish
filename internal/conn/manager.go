// Package conn keeps one SSH client per server, shared by every app, with
// host key pinning, keepalive and automatic reconnects.
package conn

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"davish/internal/apperr"
	"davish/internal/events"
	"davish/internal/platform"
	"davish/internal/store"
)

// State is a server's connection state, as shown in the UI.
type State string

// Connection states sent in conn:status.
const (
	StateDisconnected State = "disconnected"
	StateConnecting   State = "connecting"
	StateConnected    State = "connected"
	StateReconnecting State = "reconnecting"
	StateFailed       State = "failed"
)

// Status is the conn:status payload.
type Status struct {
	ServerID string        `json:"serverID"`
	State    State         `json:"state"`
	Error    *apperr.Error `json:"error,omitempty"`
	// Detail is the technical cause of Error, home directory hidden, for
	// the error dialog's "copy" box. Never a secret: causes here are SSH,
	// key parsing and file errors.
	Detail string `json:"detail,omitempty"`
	// Platform is the server's OS once it is known; the UI greys out the
	// apps that do not run there.
	Platform platform.OS `json:"platform,omitempty"`
	// Distro is the logo key of the server's system ("ubuntu", "windows"…)
	// once it is known (platform.Distro).
	Distro string `json:"distro,omitempty"`
}

// HostKeyPrompt is the conn:hostkey-prompt payload: the UI shows it and
// answers with AcceptHostKey.
type HostKeyPrompt struct {
	ServerID    string `json:"serverID"`
	Host        string `json:"host"`
	KeyType     string `json:"keyType"`
	Fingerprint string `json:"fingerprint"`
}

// ServerSource looks up saved servers.
type ServerSource interface {
	Server(ctx context.Context, id string) (store.Server, error)
}

// SecretSource reads passwords and passphrases.
type SecretSource interface {
	Get(serverID string, kind store.SecretKind) (string, error)
}

// Emitter pushes events to the frontend.
type Emitter interface {
	Emit(name string, payload any)
}

// Options tunes timing; zero values take the defaults from ARCHITECTURE.md.
type Options struct {
	DialTimeout       time.Duration // default 10s
	KeepaliveInterval time.Duration // default 15s
	KeepaliveMisses   int           // default 3
	BackoffMin        time.Duration // default 1s
	BackoffMax        time.Duration // default 30s
	// DetectPlatform asks a connected server what it runs; nil takes the
	// real probe. Tests set it to make detection slow or instant.
	DetectPlatform func(ctx context.Context, client *ssh.Client) platform.OS
	// DetectDistro gives the logo key of a connected server whose OS is
	// known; nil takes a read of /etc/os-release.
	DetectDistro func(ctx context.Context, client *ssh.Client, os platform.OS) string
}

func (o Options) withDefaults() Options {
	def := func(d *time.Duration, v time.Duration) {
		if *d == 0 {
			*d = v
		}
	}
	def(&o.DialTimeout, 10*time.Second)
	def(&o.KeepaliveInterval, 15*time.Second)
	def(&o.BackoffMin, time.Second)
	def(&o.BackoffMax, 30*time.Second)
	if o.KeepaliveMisses == 0 {
		o.KeepaliveMisses = 3
	}
	return o
}

// Manager owns every server connection.
type Manager struct {
	servers  ServerSource
	secrets  SecretSource
	hostKeys *HostKeys
	emit     Emitter
	log      *slog.Logger
	opts     Options

	root context.Context
	stop context.CancelFunc
	wg   sync.WaitGroup

	// mu guards conns and pending.
	mu      sync.Mutex
	conns   map[string]*link
	pending map[string]pendingKey
}

// link is one server's connection and the goroutine supervising it.
type link struct {
	state  State
	client *ssh.Client // nil unless connected
	cancel context.CancelFunc
	// sftp is opened on first use over client and dropped with it.
	sftp *sftp.Client
	// plat holds the server's OS. Detection runs in the background after
	// the connection is reported up, and the answer survives reconnects,
	// since the server stays the same machine.
	plat *platform.Cell
	// distro is the logo key found with plat; empty until it is known.
	distro string
}

// pendingKey is a host key shown to the user and not yet accepted.
type pendingKey struct {
	addr string
	key  ssh.PublicKey
}

// NewManager wires the manager; call Close on shutdown.
func NewManager(servers ServerSource, secrets SecretSource, hostKeys *HostKeys, emit Emitter, log *slog.Logger, opts Options) *Manager {
	root, stop := context.WithCancel(context.Background())
	m := &Manager{
		servers:  servers,
		secrets:  secrets,
		hostKeys: hostKeys,
		emit:     emit,
		log:      log,
		opts:     opts.withDefaults(),
		root:     root,
		stop:     stop,
		conns:    map[string]*link{},
		pending:  map[string]pendingKey{},
	}
	if m.opts.DetectPlatform == nil {
		m.opts.DetectPlatform = m.detectPlatform
	}
	if m.opts.DetectDistro == nil {
		m.opts.DetectDistro = m.detectDistro
	}
	return m
}

// Connect opens the connection if it is not already up. It returns once
// the first attempt succeeds or fails; later drops reconnect on their own.
// The connection's lifetime hangs off the manager, not ctx: it must outlive
// the call that opened it.
func (m *Manager) Connect(ctx context.Context, id string) error { //nolint:contextcheck // see above
	m.mu.Lock()
	if l, ok := m.conns[id]; ok && l.state != StateFailed {
		m.mu.Unlock()
		return nil
	}
	lctx, cancel := context.WithCancel(m.root)
	l := &link{state: StateConnecting, cancel: cancel, plat: platform.NewCell()}
	m.conns[id] = l
	m.mu.Unlock()
	m.publish(id, StateConnecting, nil)

	client, err := m.dial(ctx, id)
	if err != nil {
		cancel()
		m.setFailed(id, l, err)
		return err
	}

	m.mu.Lock()
	if m.conns[id] != l { // disconnected while dialing
		m.mu.Unlock()
		client.Close() //nolint:errcheck,gosec // unwanted connection
		return apperr.New(apperr.ConnNotConnected, "disconnected while connecting")
	}
	l.state, l.client = StateConnected, client
	cell := l.plat
	m.mu.Unlock()
	// Connected first: a probe that hangs on a slow login shell must not
	// hold up every connect. The OS follows in a second conn:status.
	m.publish(id, StateConnected, nil)
	m.log.Info("connected", "server", id)

	m.wg.Add(2)
	go m.detect(lctx, id, l, client, cell) //nolint:contextcheck // the link's context, not the caller's, which ends when Connect returns
	go func() {
		defer m.wg.Done()
		m.supervise(lctx, id, l, client)
	}()
	return nil
}

// Disconnect closes the connection and stops reconnecting.
func (m *Manager) Disconnect(id string) {
	m.mu.Lock()
	l, ok := m.conns[id]
	delete(m.conns, id)
	m.mu.Unlock()
	if !ok {
		return
	}
	l.cancel()
	if l.client != nil {
		l.client.Close() //nolint:errcheck,gosec // closing anyway
	}
	m.publish(id, StateDisconnected, nil)
}

// Client returns the live client, or CONN_NOT_CONNECTED.
func (m *Manager) Client(id string) (*ssh.Client, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if l, ok := m.conns[id]; ok && l.client != nil && l.state == StateConnected {
		return l.client, nil
	}
	return nil, apperr.New(apperr.ConnNotConnected, "server is not connected")
}

// SFTP returns the server's shared SFTP client, opening it on first use.
// A reconnect replaces the SSH client, and the next call opens a new one.
func (m *Manager) SFTP(id string) (*sftp.Client, error) {
	m.mu.Lock()
	l, ok := m.conns[id]
	if !ok || l.client == nil || l.state != StateConnected {
		m.mu.Unlock()
		return nil, apperr.New(apperr.ConnNotConnected, "server is not connected")
	}
	if l.sftp != nil {
		sc := l.sftp
		m.mu.Unlock()
		return sc, nil
	}
	client := l.client
	m.mu.Unlock()

	// Opening the subsystem is a round trip; do it outside the lock.
	// Concurrent writes can land out of order if a transfer fails midway;
	// that is safe because every write goes to a temporary file first.
	sc, err := openSFTP(client, m.opts.DialTimeout*2)
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	switch {
	case l.client != client || m.conns[id] != l: // reconnected or dropped meanwhile
		m.mu.Unlock()
		sc.Close() //nolint:errcheck,gosec // unwanted
		return nil, apperr.New(apperr.ConnNotConnected, "server is not connected")
	case l.sftp != nil: // another caller won the race
		other := l.sftp
		m.mu.Unlock()
		sc.Close() //nolint:errcheck,gosec // duplicate
		return other, nil
	}
	l.sftp = sc
	m.mu.Unlock()

	// The subsystem can end on its own (sftp-server killed); forget it so
	// the next call opens a fresh one.
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		sc.Wait() //nolint:errcheck,gosec // only the end matters
		m.mu.Lock()
		if l.sftp == sc {
			l.sftp = nil
		}
		m.mu.Unlock()
	}()
	return sc, nil
}

// openSFTP starts the sftp subsystem within limit. sshd starts it through
// the user's login shell, so a startup file that hangs would block it for
// ever; past limit the caller gets CMD_TIMEOUT and a late client is closed.
func openSFTP(client *ssh.Client, limit time.Duration) (*sftp.Client, error) {
	type result struct {
		sc  *sftp.Client
		err error
	}
	done := make(chan result, 1)
	go func() {
		sc, err := sftp.NewClient(client, sftp.UseConcurrentWrites(true))
		done <- result{sc, err}
	}()
	select {
	case r := <-done:
		if r.err != nil {
			return nil, apperr.Wrap(apperr.ConnNotConnected, "cannot open sftp", r.err)
		}
		return r.sc, nil
	case <-time.After(limit):
		go func() {
			if r := <-done; r.sc != nil {
				r.sc.Close() //nolint:errcheck,gosec // nobody wants it any more
			}
		}()
		return nil, apperr.New(apperr.CmdTimeout, "the server did not start sftp in time")
	}
}

// Platform reports the server's OS without waiting: Unknown until detection
// has finished (or when it learned nothing).
func (m *Manager) Platform(id string) platform.OS {
	m.mu.Lock()
	l, ok := m.conns[id]
	var cell *platform.Cell
	if ok {
		cell = l.plat
	}
	m.mu.Unlock()
	if cell == nil {
		return platform.Unknown
	}
	return cell.Get()
}

// WaitPlatform is Platform for callers that must choose commands by OS: it
// waits for a detection still under way, at most as long as detection can
// take. A server that is not connected, or whose OS stays unknown, gives
// Unknown.
func (m *Manager) WaitPlatform(ctx context.Context, id string) platform.OS {
	m.mu.Lock()
	l, ok := m.conns[id]
	var cell *platform.Cell
	if ok {
		cell = l.plat
	}
	m.mu.Unlock()
	if cell == nil {
		return platform.Unknown
	}
	ctx, cancel := context.WithTimeout(ctx, platform.DetectTimeout+time.Second)
	defer cancel()
	return cell.Wait(ctx)
}

// distroSaver is what a ServerSource may also do: remember the logo key, so
// the lists show it before the server has connected again. store.Store does.
type distroSaver interface {
	SetDistro(ctx context.Context, id, distro string) error
}

// detect runs in the background after a connection is reported up. It
// always answers the cell, so no waiter is left hanging, and announces the
// OS and the logo in a second conn:status when the connection it asked
// about is still the current one.
func (m *Manager) detect(ctx context.Context, id string, l *link, client *ssh.Client, cell *platform.Cell) {
	defer m.wg.Done()
	os := m.opts.DetectPlatform(ctx, client)
	cell.Set(os)
	if os == platform.Unknown {
		return
	}
	distro := m.opts.DetectDistro(ctx, client, os)

	m.mu.Lock()
	current := m.conns[id] == l && l.client == client && l.plat == cell
	if current {
		l.distro = distro
	}
	m.mu.Unlock()
	if !current {
		return
	}
	m.log.Info("platform detected", "server", id, "platform", os, "distro", distro)
	if saver, ok := m.servers.(distroSaver); ok && distro != "" {
		if err := saver.SetDistro(ctx, id, distro); err != nil {
			m.log.Warn("could not save the distribution", "server", id, "err", err)
		}
	}
	m.publish(id, StateConnected, nil)
}

// detectDistro gives the logo key: Windows and macOS are known from the OS,
// and a Linux says its distribution in os-release. A server that does not
// answer in time is plain Linux, and still usable.
func (m *Manager) detectDistro(ctx context.Context, client *ssh.Client, os platform.OS) string {
	switch os {
	case platform.Windows:
		return platform.DistroWindows
	case platform.Darwin:
		return platform.DistroMacOS
	case platform.Linux:
		ctx, cancel := context.WithTimeout(ctx, platform.DetectTimeout)
		defer cancel()
		out, err := execOutput(ctx, client, platform.DistroCommand)
		if err != nil {
			return platform.DistroLinux
		}
		return platform.Distro(out)
	}
	return ""
}

// detectPlatform asks the server what it runs. It never fails: a server
// that does not answer in time is Unknown, and still usable.
func (m *Manager) detectPlatform(ctx context.Context, client *ssh.Client) platform.OS {
	return platform.Detect(ctx, platform.Probe{
		Uname: func(ctx context.Context) (string, error) { return execOutput(ctx, client, "uname -s") },
		Home: func(context.Context) (string, error) {
			sc, err := openSFTP(client, platform.DetectTimeout)
			if err != nil {
				return "", err
			}
			defer sc.Close() //nolint:errcheck // probe only
			return sc.RealPath(".")
		},
	})
}

// execOutput runs one command and returns its stdout, giving up when ctx
// ends: closing the session unblocks a shell that never answers.
func execOutput(ctx context.Context, client *ssh.Client, cmd string) (string, error) {
	sess, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer sess.Close()                                      //nolint:errcheck // probe only
	stop := context.AfterFunc(ctx, func() { sess.Close() }) //nolint:errcheck,gosec // unblocks Output
	defer stop()
	out, err := sess.Output(cmd)
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	return string(out), err
}

// State reports a server's connection state.
func (m *Manager) State(id string) State {
	m.mu.Lock()
	defer m.mu.Unlock()
	if l, ok := m.conns[id]; ok {
		return l.state
	}
	return StateDisconnected
}

// AcceptHostKey pins the key the user approved and connects. fingerprint
// must match the prompt, so a key swapped in the meantime is not trusted.
func (m *Manager) AcceptHostKey(ctx context.Context, id, fingerprint string) error {
	m.mu.Lock()
	p, ok := m.pending[id]
	delete(m.pending, id)
	m.mu.Unlock()
	if !ok || Fingerprint(p.key) != fingerprint {
		return apperr.New(apperr.ConnHostKeyUnknown, "no matching host key awaiting approval")
	}
	if err := m.hostKeys.Add(p.addr, p.key); err != nil {
		return err
	}
	m.log.Info("host key accepted", "server", id)
	return m.Connect(ctx, id)
}

// Close disconnects everything and waits for background work to stop.
func (m *Manager) Close() {
	m.stop()
	m.mu.Lock()
	for _, l := range m.conns {
		if l.client != nil {
			l.client.Close() //nolint:errcheck,gosec // shutting down
		}
	}
	m.conns = map[string]*link{}
	m.mu.Unlock()
	m.wg.Wait()
}

// supervise watches a live client and reconnects it when it drops, until
// ctx is canceled or a reconnect fails in a way retrying cannot fix.
func (m *Manager) supervise(ctx context.Context, id string, l *link, client *ssh.Client) {
	for {
		m.watch(ctx, client, !m.usesControl(ctx, id))
		client.Close() //nolint:errcheck,gosec // already dead or unwanted
		if ctx.Err() != nil {
			return
		}
		m.log.Warn("connection lost", "server", id)
		m.mu.Lock()
		l.state, l.client, l.sftp = StateReconnecting, nil, nil
		m.mu.Unlock()
		m.publish(id, StateReconnecting, nil)

		var err error
		client, err = m.redial(ctx, id)
		if err != nil {
			if ctx.Err() == nil {
				m.setFailed(id, l, err)
			}
			return
		}
		m.mu.Lock()
		if ctx.Err() != nil {
			m.mu.Unlock()
			client.Close() //nolint:errcheck,gosec // disconnected meanwhile
			return
		}
		l.state, l.client = StateConnected, client
		// A known OS carries over; an unknown one is asked again.
		var cell *platform.Cell
		if l.plat.Get() == platform.Unknown {
			cell = platform.NewCell()
			l.plat = cell
		}
		m.mu.Unlock()
		m.publish(id, StateConnected, nil)
		if cell != nil {
			m.wg.Add(1)
			go m.detect(ctx, id, l, client, cell)
		}
		m.log.Info("reconnected", "server", id)
	}
}

// usesControl reports whether server id connects through an ssh control
// socket (#240).
func (m *Manager) usesControl(ctx context.Context, id string) bool {
	srv, err := m.servers.Server(ctx, id)
	return err == nil && srv.ControlPath != ""
}

// watch returns when the client closes, misses too many keepalives, or ctx
// is canceled. With keepalives false it only waits for the client to close:
// through a control socket OpenSSH's mux proxy does not forward
// keepalive@openssh.com and drops the connection that sent it, so each
// keepalive would end the connection (#301). The master looks after the
// server's liveness (ServerAliveInterval), and one that goes away closes
// the socket, which ends the client.
func (m *Manager) watch(ctx context.Context, client *ssh.Client, keepalives bool) {
	closed := make(chan struct{})
	go func() {
		client.Wait() //nolint:errcheck,gosec // only the end matters
		close(closed)
	}()
	watchLoop(ctx, closed, keepalives, m.opts.KeepaliveInterval, m.opts.KeepaliveMisses,
		func() bool { return keepalive(client, m.opts.KeepaliveInterval) })
}

// watchLoop waits for closed or ctx, and, when keepalives is set, calls ping
// every interval and returns after misses failures in a row.
func watchLoop(ctx context.Context, closed <-chan struct{}, keepalives bool, interval time.Duration, misses int, ping func() bool) {
	// A nil channel never fires: no keepalives.
	var ticks <-chan time.Time
	if keepalives {
		tick := time.NewTicker(interval)
		defer tick.Stop()
		ticks = tick.C
	}
	missed := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-closed:
			return
		case <-ticks:
			if ping() {
				missed = 0
				continue
			}
			if missed++; missed >= misses {
				return
			}
		}
	}
}

// keepalive sends keepalive@openssh.com and reports whether it was answered
// in time. A reply of "unsupported" still proves the server is there.
func keepalive(client *ssh.Client, timeout time.Duration) bool {
	done := make(chan error, 1)
	go func() {
		_, _, err := client.SendRequest("keepalive@openssh.com", true, nil)
		done <- err
	}()
	select {
	case err := <-done:
		return err == nil
	case <-time.After(timeout):
		return false
	}
}

// redial retries with exponential backoff. Errors that retrying cannot fix
// (auth, host key, missing secret) stop it.
func (m *Manager) redial(ctx context.Context, id string) (*ssh.Client, error) {
	wait := m.opts.BackoffMin
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(wait):
		}
		client, err := m.dial(ctx, id)
		if err == nil {
			return client, nil
		}
		var ae *apperr.Error
		if !errors.As(err, &ae) || ae.Code != apperr.ConnUnreachable {
			return nil, err
		}
		wait = min(wait*2, m.opts.BackoffMax)
	}
}

// maxJumps bounds a ProxyJump chain and stops servers that jump through
// each other.
const maxJumps = 8

// dial opens one SSH connection to the server, through its jump chain if it
// has one.
func (m *Manager) dial(ctx context.Context, id string) (*ssh.Client, error) {
	return m.dialVia(ctx, id, id, 0)
}

// CheckKeyLogin logs in to server id over a new connection with its key
// only, never a password (a key server's auth methods hold none), and closes it
// at once. Turning password logins
// off uses it before and after the change (ADR 0029). It does not touch
// the open connection.
func (m *Manager) CheckKeyLogin(ctx context.Context, id string) error {
	srv, err := m.servers.Server(ctx, id)
	if err != nil {
		return err
	}
	if srv.Auth == store.AuthPassword {
		return apperr.New(apperr.InvalidInput, "this server logs in with a password, not a key")
	}
	if srv.ControlPath != "" {
		return apperr.New(apperr.InvalidInput, "a control socket reuses an open connection, so a key login cannot be tried")
	}
	c, err := m.dial(ctx, id)
	if err != nil {
		return err
	}
	return c.Close()
}

// dialVia connects to server id on behalf of target: a hop's unknown host
// key is shown as target's prompt, and approving it reconnects target.
// Every hop is a fresh connection with its own host key check and auth.
func (m *Manager) dialVia(ctx context.Context, target, id string, depth int) (*ssh.Client, error) {
	srv, err := m.servers.Server(ctx, id)
	if err != nil {
		return nil, err
	}
	if srv.ControlPath != "" {
		return m.dialControl(ctx, srv)
	}
	watch := &signWatch{}
	auth, release, err := m.authMethods(srv, watch)
	if err != nil {
		return nil, err
	}
	defer release()

	addr := net.JoinHostPort(srv.Host, strconv.Itoa(srv.Port))
	var res hostKeyResult
	check, err := m.hostKeys.callback(&res)
	if err != nil {
		return nil, err
	}
	algos, err := m.hostKeys.algorithms(addr)
	if err != nil {
		return nil, err
	}
	cfg := &ssh.ClientConfig{
		User:              srv.User,
		Auth:              auth,
		HostKeyCallback:   check,
		HostKeyAlgorithms: algos,
		Timeout:           m.opts.DialTimeout,
	}

	var jump *ssh.Client
	var nc net.Conn
	var proxy *proxyConn
	switch {
	case srv.JumpID != "":
		if depth >= maxJumps {
			return nil, apperr.New(apperr.InvalidInput, "jump chain is too long or loops")
		}
		if jump, err = m.dialVia(ctx, target, srv.JumpID, depth+1); err != nil {
			return nil, err
		}
		if nc, err = jump.DialContext(ctx, "tcp", addr); err != nil {
			jump.Close() //nolint:errcheck,gosec // unusable hop
			return nil, apperr.Wrap(apperr.ConnUnreachable, "cannot reach the server through its jump host", err)
		}
	case srv.ProxyCommand != "":
		// The command owns how the server is reached; the handshake that
		// follows is the normal one, over its stdin and stdout.
		if proxy, err = dialProxy(srv); err != nil {
			// A refusal keeps its own code (unconfirmed, unsafe value); only
			// a command that could not start is "unreachable".
			var ae *apperr.Error
			if errors.As(err, &ae) {
				return nil, err
			}
			return nil, apperr.Wrap(apperr.ConnUnreachable, "cannot start the proxy command", err)
		}
		nc = proxy
	default:
		d := net.Dialer{Timeout: m.opts.DialTimeout}
		if nc, err = d.DialContext(ctx, "tcp", addr); err != nil {
			return nil, apperr.Wrap(apperr.ConnUnreachable, "cannot reach the server", err)
		}
	}
	// Bound the handshake too; a silent server must not hang the UI. A
	// tunneled conn has no deadlines, so close it instead. If the user
	// raised DAVISH_AGENT_WAIT for a vault or Touch ID prompt, let the
	// handshake wait at least that long too.
	handshakeTimeout := m.opts.DialTimeout
	if srv.Auth == store.AuthAgent || srv.Auth == store.AuthKey {
		if wait, ok := agentWaitOverride(); ok && wait > handshakeTimeout {
			handshakeTimeout = wait
		}
	}
	// While the SSH agent signs the login the person may be answering its
	// prompt (Bitwarden's "Confirm SSH key usage"), which takes longer than
	// a server needs to answer: the clock waits for that (#168).
	signMax := max(agentSignWait, handshakeTimeout)
	stopTimer := watch.guard(handshakeTimeout, signMax, func() { nc.Close() }) //nolint:errcheck,gosec // aborting
	c, chans, reqs, err := ssh.NewClientConn(nc, addr, cfg)
	if !stopTimer() && err == nil { // timed out just as the handshake finished
		c.Close() //nolint:errcheck,gosec // already closed underneath
		err = errors.New("ssh handshake timed out")
	}
	if err != nil {
		nc.Close() //nolint:errcheck,gosec // handshake failed
		if jump != nil {
			jump.Close() //nolint:errcheck,gosec // unusable hop
		}
		if proxy != nil {
			// A tunnel that died takes the blame when it has something to
			// confess; a live one stays out of the report.
			if why := proxy.explain(); why != "" {
				err = fmt.Errorf("%w (%s)", err, why)
			}
		}
		if watch.timedOut() {
			m.log.Warn("ssh agent did not sign in time", "server", target, "waited", signMax.String())
			return nil, apperr.New(apperr.ConnAgentSignTimeout, "the SSH agent did not sign the login in time")
		}
		return nil, m.handshakeError(target, addr, &res, err)
	}
	client := ssh.NewClient(c, chans, reqs)
	if jump != nil {
		// The hop lives as long as the client; a dead hop ends the client,
		// and the reconnect rebuilds the whole chain.
		go func() {
			client.Wait() //nolint:errcheck,gosec // only the end matters
			jump.Close()  //nolint:errcheck,gosec // no longer needed
		}()
	}
	return client, nil
}

func (m *Manager) handshakeError(id, addr string, res *hostKeyResult, err error) error {
	switch {
	case res.changed:
		m.log.Warn("host key changed, refusing", "server", id)
		return apperr.New(apperr.ConnHostKeyChanged, "host key changed")
	case res.unknown != nil:
		m.mu.Lock()
		m.pending[id] = pendingKey{addr: addr, key: res.unknown}
		m.mu.Unlock()
		m.emit.Emit(events.ConnHostKeyPrompt, HostKeyPrompt{
			ServerID:    id,
			Host:        addr,
			KeyType:     res.unknown.Type(),
			Fingerprint: Fingerprint(res.unknown),
		})
		return apperr.New(apperr.ConnHostKeyUnknown, "host key needs approval")
	case strings.Contains(err.Error(), "unable to authenticate"):
		m.log.Warn("auth failed", "server", id)
		return apperr.Wrap(apperr.ConnAuthFailed, "authentication failed", err)
	}
	// Without this line a handshake that fails for any other reason leaves
	// nothing in the log (#168).
	m.log.Warn("ssh handshake failed", "server", id, "err", err)
	return apperr.Wrap(apperr.ConnUnreachable, "ssh handshake failed", err)
}

func (m *Manager) setFailed(id string, l *link, err error) {
	m.mu.Lock()
	if m.conns[id] == l {
		l.state, l.client, l.sftp = StateFailed, nil, nil
	}
	m.mu.Unlock()
	m.publish(id, StateFailed, err)
}

func (m *Manager) publish(id string, state State, err error) {
	st := Status{ServerID: id, State: state}
	if state == StateConnected {
		if os := m.Platform(id); os != platform.Unknown {
			st.Platform = os
		}
		m.mu.Lock()
		if l, ok := m.conns[id]; ok {
			st.Distro = l.distro
		}
		m.mu.Unlock()
	}
	if err != nil {
		if f, ok := apperr.Format(err).(*apperr.Error); ok {
			st.Error = f
		}
		st.Detail = apperr.Detail(err)
		// A failed login logs only its result (.claude/docs/SECURITY.md); other
		// causes are kept, or a support log could not explain them.
		if st.Error.Code == apperr.ConnAuthFailed {
			m.log.Info("connection state", "server", id, "state", state, "err", fmt.Sprint(st.Error.Code))
		} else {
			m.log.Warn("connection state", "server", id, "state", state, "err", fmt.Sprint(st.Error.Code), "detail", st.Detail)
		}
	}
	m.emit.Emit(events.ConnStatus, st)
}
