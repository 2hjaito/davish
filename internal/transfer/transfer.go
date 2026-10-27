// Package transfer moves files and folders between this machine and a
// server over SFTP, with progress events and cancellation.
package transfer

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/pkg/sftp"

	"davish/internal/apperr"
	"davish/internal/events"
)

// Conns hands out a server's shared SFTP client.
type Conns interface {
	SFTP(serverID string) (*sftp.Client, error)
}

// Emitter pushes events to the frontend.
type Emitter interface {
	Emit(name string, payload any)
}

// Job kinds.
const (
	KindUpload   = "upload"
	KindDownload = "download"
)

// Job is one file or folder being moved.
type Job struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	ServerID string `json:"serverID"`
	// Name is the final name at the destination, which differs from the
	// source when the user chose to keep both.
	Name string `json:"name"`
	Src  string `json:"src"`
	Dst  string `json:"dst"`
	Dir  bool   `json:"dir"`
}

// Progress is the transfer:progress payload. Total is 0 until a folder
// has been walked.
type Progress struct {
	JobID string `json:"jobID"`
	Done  int64  `json:"done"`
	Total int64  `json:"total"`
	Bps   int64  `json:"bps"`
}

// Done is the transfer:done payload.
type Done struct {
	JobID    string        `json:"jobID"`
	Canceled bool          `json:"canceled,omitempty"`
	Error    *apperr.Error `json:"error,omitempty"`
}

// Options tunes the service; zero values take the defaults.
type Options struct {
	// DownloadDir receives downloads; default ~/Downloads.
	DownloadDir string
	// Parallel is how many jobs run at once; the rest wait. Default 2.
	Parallel int
	// ProgressEvery throttles transfer:progress; default 250ms (4 per second).
	ProgressEvery time.Duration
}

// Service runs transfer jobs in the background.
type Service struct {
	conns Conns
	emit  Emitter
	log   *slog.Logger
	opts  Options
	slots chan struct{}

	root context.Context
	stop context.CancelFunc
	wg   sync.WaitGroup

	// mu guards cancels.
	mu      sync.Mutex
	cancels map[string]context.CancelFunc
}

// New creates the service; call Close on shutdown.
func New(conns Conns, emit Emitter, log *slog.Logger, opts Options) (*Service, error) {
	if opts.DownloadDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		opts.DownloadDir = home + string(os.PathSeparator) + "Downloads"
	}
	if opts.Parallel <= 0 {
		opts.Parallel = 2
	}
	if opts.ProgressEvery <= 0 {
		opts.ProgressEvery = 250 * time.Millisecond
	}
	root, stop := context.WithCancel(context.Background())
	return &Service{
		conns: conns, emit: emit, log: log, opts: opts,
		slots:   make(chan struct{}, opts.Parallel),
		root:    root,
		stop:    stop,
		cancels: map[string]context.CancelFunc{},
	}, nil
}

// Cancel stops a job; it ends with transfer:done {canceled: true} once
// its partial files are removed.
func (s *Service) Cancel(jobID string) error {
	s.mu.Lock()
	cancel, ok := s.cancels[jobID]
	s.mu.Unlock()
	if !ok {
		return apperr.New(apperr.NotFound, "no such transfer")
	}
	cancel()
	return nil
}

// Close cancels every job and waits for their cleanup.
func (s *Service) Close() {
	s.stop()
	s.wg.Wait()
}

// start queues run as job. run reports progress through the meter and
// must remove whatever it left behind when it fails.
func (s *Service) start(job Job, run func(ctx context.Context, m *meter) error) {
	ctx, cancel := context.WithCancel(s.root)
	s.mu.Lock()
	s.cancels[job.ID] = cancel
	s.mu.Unlock()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer func() {
			cancel()
			s.mu.Lock()
			delete(s.cancels, job.ID)
			s.mu.Unlock()
		}()

		var err error
		select {
		case s.slots <- struct{}{}:
			m := newMeter(ctx, job.ID, s.emit, s.opts.ProgressEvery)
			err = run(ctx, m)
			<-s.slots
			if err == nil {
				m.flush()
			}
		case <-ctx.Done():
			err = ctx.Err()
		}
		s.finish(job, err)
	}()
}

func (s *Service) finish(job Job, err error) {
	done := Done{JobID: job.ID}
	switch {
	case err == nil:
		s.log.Info("transfer done", "server", job.ServerID, "app", "transfer", "op", job.Kind)
	case errors.Is(err, context.Canceled):
		done.Canceled = true
		s.log.Info("transfer canceled", "server", job.ServerID, "app", "transfer", "op", job.Kind)
	default:
		if f, ok := apperr.Format(err).(*apperr.Error); ok {
			done.Error = f
		}
		s.log.Warn("transfer failed", "server", job.ServerID, "app", "transfer", "op", job.Kind, "err", err)
	}
	s.emit.Emit(events.TransferDone, done)
}

func newID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// meter counts bytes and throttles transfer:progress.
type meter struct {
	ctx   context.Context
	id    string
	emit  Emitter
	every time.Duration

	// mu guards the fields below.
	mu       sync.Mutex
	done     int64
	total    int64
	lastAt   time.Time
	lastDone int64
}

func newMeter(ctx context.Context, id string, emit Emitter, every time.Duration) *meter {
	return &meter{ctx: ctx, id: id, emit: emit, every: every, lastAt: time.Now()}
}

func (m *meter) setTotal(total int64) {
	m.mu.Lock()
	m.total = total
	p := m.progress(time.Now())
	m.mu.Unlock()
	m.emit.Emit(events.TransferProgress, p)
}

// add records n bytes and fails once the job is canceled, which is how a
// cancel interrupts a copy in progress.
func (m *meter) add(n int) error {
	now := time.Now()
	m.mu.Lock()
	m.done += int64(n)
	var p *Progress
	if now.Sub(m.lastAt) >= m.every {
		pr := m.progress(now)
		p = &pr
	}
	m.mu.Unlock()
	if p != nil {
		m.emit.Emit(events.TransferProgress, *p)
	}
	return m.ctx.Err()
}

// flush sends the final count.
func (m *meter) flush() {
	m.mu.Lock()
	p := m.progress(time.Now())
	m.mu.Unlock()
	m.emit.Emit(events.TransferProgress, p)
}

// progress must be called with mu held.
func (m *meter) progress(now time.Time) Progress {
	var bps int64
	if dt := now.Sub(m.lastAt).Seconds(); dt > 0 {
		bps = int64(float64(m.done-m.lastDone) / dt)
	}
	m.lastAt, m.lastDone = now, m.done
	return Progress{JobID: m.id, Done: m.done, Total: m.total, Bps: bps}
}
