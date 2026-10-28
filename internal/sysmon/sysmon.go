// Package sysmon collects system metrics by reading /proc over a long-lived
// SSH session, and lists processes and system details on request.
package sysmon

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/ssh"

	"davish/internal/events"
)

//go:embed collect.sh
var collectScript string

// ClientSource hands out the live SSH client of a server.
type ClientSource interface {
	Client(serverID string) (*ssh.Client, error)
}

// Emitter pushes events to the frontend.
type Emitter interface {
	Emit(name string, payload any)
}

// Options tunes the service; zero values take the defaults.
type Options struct {
	// RetryEvery is how long to wait before collecting again after the
	// server was unreachable or the collector ended.
	RetryEvery time.Duration
	// Thresholds trigger notify:new.
	Thresholds Thresholds
}

// Service runs one collector per watched server.
type Service struct {
	clients ClientSource
	emit    Emitter
	log     *slog.Logger
	opts    Options

	// mu guards watchers.
	mu       sync.Mutex
	watchers map[string]*watcher
	wg       sync.WaitGroup

	// procsMu guards procs, the last process sample per server.
	procsMu sync.Mutex
	procs   map[string]procSample

	// alertsMu guards alerts and opts.Thresholds. Alerts outlive watchers
	// so that watching a server again does not repeat a notification.
	alertsMu sync.Mutex
	alerts   map[string]*alerter

	// idle slows every collector down while nobody looks at Davish; the
	// alerts still need readings. paceMu guards paced, closed and
	// replaced when idle changes so running collectors restart.
	idle   atomic.Bool
	paceMu sync.Mutex
	paced  chan struct{}
}

// Collection pace: every 2s while Davish is on screen, every 30s while it
// is hidden or locked, with failed units and restarts read every 30s either
// way.
const (
	activeEvery = 2
	idleEvery   = 30
	healthSecs  = 30
)

// errRepace ends a collector so it restarts at the new pace.
var errRepace = errors.New("pace changed")

type watcher struct {
	refs   int
	cancel context.CancelFunc
}

// New creates the service; call Close on shutdown.
func New(clients ClientSource, emit Emitter, log *slog.Logger, opts Options) *Service {
	if opts.RetryEvery <= 0 {
		opts.RetryEvery = 3 * time.Second
	}
	if opts.Thresholds == (Thresholds{}) {
		opts.Thresholds = DefaultThresholds
	}
	return &Service{
		clients: clients, emit: emit, log: log, opts: opts,
		watchers: map[string]*watcher{}, procs: map[string]procSample{},
		alerts: map[string]*alerter{}, paced: make(chan struct{}),
	}
}

// SetIdle slows collection down while Davish is hidden or locked, and
// brings it back to full speed when it is seen again.
func (s *Service) SetIdle(idle bool) {
	if s.idle.Swap(idle) == idle {
		return
	}
	s.paceMu.Lock()
	close(s.paced)
	s.paced = make(chan struct{})
	s.paceMu.Unlock()
}

// script is collect.sh at the current pace.
func (s *Service) script() (string, <-chan struct{}) {
	every := activeEvery
	if s.idle.Load() {
		every = idleEvery
	}
	s.paceMu.Lock()
	ch := s.paced
	s.paceMu.Unlock()
	return fmt.Sprintf("every=%d\nhealthEvery=%d\n", every, max(1, healthSecs/every)) + collectScript, ch
}

// SetThresholds changes when RAM and disk readings notify, for every
// server, from the next tick on (Settings → Cảnh báo).
func (s *Service) SetThresholds(th Thresholds) {
	s.alertsMu.Lock()
	defer s.alertsMu.Unlock()
	s.opts.Thresholds = th
	for _, a := range s.alerts {
		a.th = th
	}
}

// Watch starts emitting sysmon:tick for serverID until every Watch has a
// matching Unwatch. The menu bar and each Monitor window watch separately.
func (s *Service) Watch(serverID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if w, ok := s.watchers[serverID]; ok {
		w.refs++
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.watchers[serverID] = &watcher{refs: 1, cancel: cancel}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.run(ctx, serverID)
	}()
}

// Unwatch drops one Watch; the collector stops with the last one.
func (s *Service) Unwatch(serverID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	w, ok := s.watchers[serverID]
	if !ok {
		return
	}
	if w.refs--; w.refs == 0 {
		w.cancel()
		delete(s.watchers, serverID)
	}
}

// Close stops every collector and waits for them.
func (s *Service) Close() {
	s.mu.Lock()
	for id, w := range s.watchers {
		w.cancel()
		delete(s.watchers, id)
	}
	s.mu.Unlock()
	s.wg.Wait()
}

// run keeps a collector going while ctx lives. The connection may be down
// or come back later; retrying picks the new client up after a reconnect.
func (s *Service) run(ctx context.Context, serverID string) {
	for {
		if client, err := s.clients.Client(serverID); err == nil {
			err := s.collect(ctx, serverID, client)
			if errors.Is(err, errRepace) {
				continue
			}
			if err != nil && ctx.Err() == nil {
				s.log.Warn("sysmon collector ended", "server", serverID, "err", err)
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(s.opts.RetryEvery):
		}
	}
}

// collect runs collect.sh in one session and emits a tick per snapshot
// after the first, until the session ends or ctx is cancelled.
func (s *Service) collect(ctx context.Context, serverID string, client *ssh.Client) error {
	sess, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("open session: %w", err)
	}
	defer sess.Close() //nolint:errcheck // closed either way

	// The script goes in on stdin, so no command line is built and the
	// user's login shell (fish, csh…) does not matter.
	script, repace := s.script()
	sess.Stdin = strings.NewReader(script)
	stdout, err := sess.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	if err := sess.Start("sh -s"); err != nil {
		return fmt.Errorf("start collector: %w", err)
	}
	// Reading goes through a pipe this side can close: closing the session
	// alone waits for the server, which may be in a 30s sleep.
	pr, pw := io.Pipe()
	go func() {
		_, err := io.Copy(pw, stdout)
		pw.CloseWithError(err) //nolint:errcheck,gosec // always nil
	}()
	abort := func() {
		pw.Close()               //nolint:errcheck,gosec // unblocks the reader now
		sess.Signal(ssh.SIGTERM) //nolint:errcheck,gosec // older sshd ignore it; SIGPIPE ends it later
		sess.Close()             //nolint:errcheck,gosec // closed either way
	}
	stop := context.AfterFunc(ctx, abort)
	defer stop()
	var repaced atomic.Bool
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-repace:
			repaced.Store(true)
			abort() // restarts at the new pace
		case <-done:
		}
	}()

	s.resumeAlerts(serverID)
	r := NewReader(pr)
	var prev *Snapshot
	for {
		snap, err := r.Next()
		if err != nil && repaced.Load() {
			return errRepace
		}
		if errors.Is(err, io.EOF) {
			return errors.New("collector output ended")
		}
		if errors.Is(err, ErrMalformed) {
			s.log.Warn("sysmon snapshot skipped", "server", serverID, "err", err)
			prev = nil
			continue
		}
		if err != nil {
			return err
		}
		if snap.Health != nil {
			for _, a := range s.checkHealth(serverID, snap.Uptime, snap.Health) {
				s.emit.Emit(events.NotifyNew, a)
			}
		}
		if prev != nil {
			tick := Rate(serverID, *prev, snap)
			s.emit.Emit(events.SysmonTick, tick)
			for _, a := range s.checkAlerts(tick) {
				s.emit.Emit(events.NotifyNew, a)
			}
		}
		prev = &snap
	}
}

// resumeAlerts readies the server's alerter for a new collector. Two
// collectors can briefly overlap after a quick Unwatch and Watch, hence the
// lock around every use.
func (s *Service) resumeAlerts(serverID string) {
	s.alertsMu.Lock()
	defer s.alertsMu.Unlock()
	if a, ok := s.alerts[serverID]; ok {
		a.resume()
		return
	}
	s.alerts[serverID] = newAlerter(s.opts.Thresholds)
}

func (s *Service) checkAlerts(t Tick) []Alert {
	s.alertsMu.Lock()
	defer s.alertsMu.Unlock()
	a, ok := s.alerts[t.ServerID]
	if !ok {
		return nil
	}
	return a.check(t)
}

func (s *Service) checkHealth(serverID string, uptime float64, h *Health) []Alert {
	s.alertsMu.Lock()
	defer s.alertsMu.Unlock()
	a, ok := s.alerts[serverID]
	if !ok {
		return nil
	}
	return a.health.check(serverID, uptime, h)
}
