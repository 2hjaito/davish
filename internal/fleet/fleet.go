// Package fleet runs one command on several servers at once, each in its
// own terminal session on a PTY, at most Parallel at a time
// (docs/adr/0021). The user answers any prompt in that server's tab;
// Davish never types a sudo password for a run.
package fleet

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"strings"
	"sync"
	"time"

	"davish/internal/apperr"
	"davish/internal/audit"
	"davish/internal/events"
)

// Parallel is how many servers run at once; the rest wait their turn.
const Parallel = 4

// Limits on a run.
const (
	MaxServers = 100
	MaxCommand = 8 << 10
)

// app is the audit app name: the All servers app.
const app = "servers"

// Terms runs command sessions; term.Service in the app.
type Terms interface {
	Run(ctx context.Context, serverID string, cols, rows int, command string, onStart func(id string)) (string, <-chan int, error)
	Close(id string) error
}

// Auditor records what ran; audit.Log in the app.
type Auditor interface {
	Add(e audit.Entry) audit.Entry
}

// Emitter sends events to the frontend.
type Emitter interface {
	Emit(name string, payload any)
}

// Item is one server and the command to run on it.
type Item struct {
	ServerID string `json:"serverID"`
	Command  string `json:"command"`
}

// Job states.
const (
	Queued  = "queued"
	Running = "running"
	Done    = "done"    // exit 0
	Failed  = "failed"  // another exit code, a lost connection, or no start
	Stopped = "stopped" // stopped by the user
)

// Job is one server's part of a run, sent as fleet:job on every change.
type Job struct {
	RunID    string `json:"runID"`
	ServerID string `json:"serverID"`
	State    string `json:"state"`
	// TermID is the terminal session, set once it runs.
	TermID string `json:"termID,omitempty"`
	// Exit is the exit code once finished; -1 when it never finished.
	Exit int `json:"exit"`
	// Error is an apperr code when the session could not start.
	Error string `json:"error,omitempty"`
}

// Service runs and tracks runs.
type Service struct {
	terms Terms
	audit Auditor
	emit  Emitter
	log   *slog.Logger

	mu   sync.Mutex
	runs map[string]*run

	// stopGrace is how long a stopped job may take to end.
	stopGrace time.Duration
}

type run struct {
	cancel context.CancelFunc
}

// New creates the service.
func New(terms Terms, audit Auditor, emit Emitter, log *slog.Logger) *Service {
	return &Service{terms: terms, audit: audit, emit: emit, log: log, runs: map[string]*run{}, stopGrace: 5 * time.Second}
}

// afterCancel fires grace after ctx is canceled, never before.
func afterCancel(ctx context.Context, grace time.Duration) <-chan struct{} {
	c := make(chan struct{})
	context.AfterFunc(ctx, func() {
		time.AfterFunc(grace, func() { close(c) })
	})
	return c
}

// Start begins a run and returns its ID and the jobs, all queued. Each
// session opens cols×rows. Progress arrives as fleet:job events.
func (s *Service) Start(ctx context.Context, items []Item, cols, rows int) (string, []Job, error) {
	if err := validate(items); err != nil {
		return "", nil, err
	}
	id, err := newID()
	if err != nil {
		return "", nil, err
	}
	rctx, cancel := context.WithCancel(ctx)
	r := &run{cancel: cancel}
	s.mu.Lock()
	s.runs[id] = r
	s.mu.Unlock()

	jobs := make([]Job, len(items))
	for i, it := range items {
		jobs[i] = Job{RunID: id, ServerID: it.ServerID, State: Queued, Exit: -1}
	}
	go s.schedule(rctx, id, r, items, cols, rows)
	return id, jobs, nil
}

func validate(items []Item) error {
	if len(items) == 0 || len(items) > MaxServers {
		return apperr.New(apperr.InvalidInput, "choose between 1 and 100 servers")
	}
	seen := map[string]bool{}
	for _, it := range items {
		if it.ServerID == "" || seen[it.ServerID] {
			return apperr.New(apperr.InvalidInput, "each server may appear once")
		}
		seen[it.ServerID] = true
		if strings.TrimSpace(it.Command) == "" || len(it.Command) > MaxCommand || strings.ContainsRune(it.Command, 0) {
			return apperr.New(apperr.InvalidInput, "the command is empty or too long")
		}
	}
	return nil
}

// schedule runs the items in order, Parallel at a time.
func (s *Service) schedule(ctx context.Context, id string, r *run, items []Item, cols, rows int) {
	slots := make(chan struct{}, Parallel)
	var wg sync.WaitGroup
	for _, it := range items {
		select {
		case slots <- struct{}{}:
		case <-ctx.Done():
		}
		if ctx.Err() != nil {
			s.send(Job{RunID: id, ServerID: it.ServerID, State: Stopped, Exit: -1})
			continue
		}
		wg.Add(1)
		go func(it Item) {
			defer wg.Done()
			defer func() { <-slots }()
			s.runOne(ctx, id, it, cols, rows)
		}(it)
	}
	wg.Wait()
	r.cancel()
	s.mu.Lock()
	delete(s.runs, id)
	s.mu.Unlock()
}

func (s *Service) runOne(ctx context.Context, id string, it Item, cols, rows int) {
	start := time.Now()
	job := Job{RunID: id, ServerID: it.ServerID, Exit: -1}
	termID, done, err := s.terms.Run(ctx, it.ServerID, cols, rows, it.Command, func(tid string) {
		// Sent before any output, so the frontend knows the session.
		s.send(Job{RunID: id, ServerID: it.ServerID, State: Running, TermID: tid, Exit: -1})
	})
	if err != nil {
		job.State = Failed
		if ctx.Err() != nil {
			job.State = Stopped
		}
		job.Error = code(err)
		s.record(it, start, -1)
		s.send(job)
		return
	}
	// A stop while it runs ends the session; its exit code follows. A
	// server that keeps the command going anyway does not hold the run:
	// after stopGrace the job counts as stopped.
	stop := context.AfterFunc(ctx, func() { _ = s.terms.Close(termID) })
	select {
	case job.Exit = <-done:
	case <-afterCancel(ctx, s.stopGrace):
	}
	stop()

	job.TermID = termID
	switch {
	case job.Exit == 0:
		job.State = Done
	case ctx.Err() != nil:
		job.State = Stopped
	default:
		job.State = Failed
	}
	s.record(it, start, job.Exit)
	s.send(job)
}

// record writes one audit line per server: the command and its exit
// code, never the output or what the user typed into the tab.
func (s *Service) record(it Item, start time.Time, exit int) {
	s.audit.Add(audit.Entry{
		Time:     start.UnixMilli(),
		ServerID: it.ServerID,
		App:      app,
		Command:  it.Command,
		Exit:     exit,
		DurMs:    time.Since(start).Milliseconds(),
		// Typed to act on the servers: exported like any change (docs/adr/0022).
		Mutating: true,
	})
	s.log.Info("fleet command", "server", it.ServerID, "exit", exit)
}

func (s *Service) send(j Job) { s.emit.Emit(events.FleetJob, j) }

// Stop ends a run: queued servers do not start, running sessions close.
func (s *Service) Stop(runID string) {
	s.mu.Lock()
	r, ok := s.runs[runID]
	s.mu.Unlock()
	if ok {
		r.cancel()
	}
}

// StopAll ends every run, on shutdown.
func (s *Service) StopAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, r := range s.runs {
		r.cancel()
	}
}

func newID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", apperr.Wrap(apperr.Internal, "cannot make an ID", err)
	}
	return hex.EncodeToString(b), nil
}
