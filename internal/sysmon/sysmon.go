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

	// alertsMu guards alerts. Outlives watchers so that watching a server
	// again does not repeat a notification it already gave.
	alertsMu sync.Mutex
	alerts   map[string]*alerter
}

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
		alerts: map[string]*alerter{},
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
			if err := s.collect(ctx, serverID, client); err != nil && ctx.Err() == nil {
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
	sess.Stdin = strings.NewReader(collectScript)
	stdout, err := sess.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	if err := sess.Start("sh -s"); err != nil {
		return fmt.Errorf("start collector: %w", err)
	}
	stop := context.AfterFunc(ctx, func() { sess.Close() }) //nolint:errcheck,gosec // unblocks the reader
	defer stop()

	s.resumeAlerts(serverID)
	r := NewReader(stdout)
	var prev *Snapshot
	for {
		snap, err := r.Next()
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
