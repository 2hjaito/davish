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
}

// pendingKey is a host key shown to the user and not yet accepted.
type pendingKey struct {
	addr string
	key  ssh.PublicKey
}

// NewManager wires the manager; call Close on shutdown.
func NewManager(servers ServerSource, secrets SecretSource, hostKeys *HostKeys, emit Emitter, log *slog.Logger, opts Options) *Manager {
	root, stop := context.WithCancel(context.Background())
	return &Manager{
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
	l := &link{state: StateConnecting, cancel: cancel}
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
	m.mu.Unlock()
	m.publish(id, StateConnected, nil)
	m.log.Info("connected", "server", id)

	m.wg.Add(1)
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
		m.watch(ctx, client)
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
		m.mu.Unlock()
		m.publish(id, StateConnected, nil)
		m.log.Info("reconnected", "server", id)
	}
}

// watch returns when the client closes, misses too many keepalives, or ctx
// is canceled.
func (m *Manager) watch(ctx context.Context, client *ssh.Client) {
	closed := make(chan struct{})
	go func() {
		client.Wait() //nolint:errcheck,gosec // only the end matters
		close(closed)
	}()
	tick := time.NewTicker(m.opts.KeepaliveInterval)
	defer tick.Stop()
	misses := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-closed:
			return
		case <-tick.C:
			if keepalive(client, m.opts.KeepaliveInterval) {
				misses = 0
				continue
			}
			if misses++; misses >= m.opts.KeepaliveMisses {
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

// dialVia connects to server id on behalf of target: a hop's unknown host
// key is shown as target's prompt, and approving it reconnects target.
// Every hop is a fresh connection with its own host key check and auth.
func (m *Manager) dialVia(ctx context.Context, target, id string, depth int) (*ssh.Client, error) {
	srv, err := m.servers.Server(ctx, id)
	if err != nil {
		return nil, err
	}
	auth, release, err := m.authMethods(srv)
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
	if srv.JumpID != "" {
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
	} else {
		d := net.Dialer{Timeout: m.opts.DialTimeout}
		if nc, err = d.DialContext(ctx, "tcp", addr); err != nil {
			return nil, apperr.Wrap(apperr.ConnUnreachable, "cannot reach the server", err)
		}
	}
	// Bound the handshake too; a silent server must not hang the UI. A
	// tunneled conn has no deadlines, so close it instead.
	timer := time.AfterFunc(m.opts.DialTimeout, func() { nc.Close() }) //nolint:errcheck,gosec // aborting
	c, chans, reqs, err := ssh.NewClientConn(nc, addr, cfg)
	if !timer.Stop() && err == nil { // timed out just as the handshake finished
		c.Close() //nolint:errcheck,gosec // already closed underneath
		err = errors.New("ssh handshake timed out")
	}
	if err != nil {
		nc.Close() //nolint:errcheck,gosec // handshake failed
		if jump != nil {
			jump.Close() //nolint:errcheck,gosec // unusable hop
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
	if err != nil {
		if f, ok := apperr.Format(err).(*apperr.Error); ok {
			st.Error = f
		}
		m.log.Info("connection state", "server", id, "state", state, "err", fmt.Sprint(st.Error.Code))
	}
	m.emit.Emit(events.ConnStatus, st)
}
