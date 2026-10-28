package logs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

	"davish/internal/apperr"
	"davish/internal/audit"
	"davish/internal/events"
	"davish/internal/shellx"
)

// ClientSource hands out a server's live SSH client.
type ClientSource interface {
	Client(serverID string) (*ssh.Client, error)
}

// Emitter pushes events to the frontend.
type Emitter interface {
	Emit(name string, payload any)
}

// Lines is the logs:line payload.
type Lines struct {
	StreamID string   `json:"streamID"`
	Lines    []string `json:"lines"`
}

// Started tells the frontend which stream is its own, and how to read the
// same source as root: an empty journal may only mean the account cannot
// read the system journal, which journalctl does not report.
type Started struct {
	StreamID string `json:"streamID"`
	Sudo     string `json:"sudo"`
}

// End is the logs:end payload. Error is nil when the user stopped the
// stream; Sudo is set with CMD_NEEDS_ROOT.
type End struct {
	StreamID string        `json:"streamID"`
	Error    *apperr.Error `json:"error,omitempty"`
	Sudo     string        `json:"sudo,omitempty"`
}

// flushEvery batches lines, as ARCHITECTURE.md §4.2 asks of logs:line.
const flushEvery = 100 * time.Millisecond

// maxBatch caps one event; a flood beyond it waits for the next flush.
const maxBatch = 2000

// Service owns every log stream.
type Service struct {
	clients ClientSource
	audit   *audit.Log
	emit    Emitter
	log     *slog.Logger

	// mu guards streams.
	mu      sync.Mutex
	streams map[string]*stream
	wg      sync.WaitGroup
}

type stream struct {
	sess *ssh.Session
	// stopped is set when the user stopped it, so the end carries no error.
	stopped bool
	// denied holds the sudo command once stderr showed a lack of rights.
	denied string
}

// New creates the service; call Close on shutdown.
func New(clients ClientSource, auditLog *audit.Log, emit Emitter, log *slog.Logger) *Service {
	return &Service{clients: clients, audit: auditLog, emit: emit, log: log, streams: map[string]*stream{}}
}

// Start follows src on serverID and returns the stream's ID with its sudo
// form. Lines arrive as logs:line, and logs:end tells when it is over.
func (s *Service) Start(_ context.Context, serverID string, src Source) (Started, error) {
	cmd, err := Command(src)
	if err != nil {
		return Started{}, err
	}
	client, err := s.clients.Client(serverID)
	if err != nil {
		return Started{}, err
	}
	sess, err := client.NewSession()
	if err != nil {
		return Started{}, apperr.Wrap(apperr.ConnNotConnected, "cannot open a session", err)
	}
	stdout, err := sess.StdoutPipe()
	if err != nil {
		sess.Close() //nolint:errcheck,gosec // failed to start
		return Started{}, apperr.Wrap(apperr.Internal, "stdout pipe", err)
	}
	stderr, err := sess.StderrPipe()
	if err != nil {
		sess.Close() //nolint:errcheck,gosec // failed to start
		return Started{}, apperr.Wrap(apperr.Internal, "stderr pipe", err)
	}
	line := cmd.String()
	if err := sess.Start(line); err != nil {
		sess.Close() //nolint:errcheck,gosec // failed to start
		return Started{}, apperr.Wrap(apperr.ConnNotConnected, "cannot start the command", err)
	}
	// A stream has no exit status until the user stops it, so it is
	// recorded when it starts, with -1.
	s.audit.Add(audit.Entry{Time: time.Now().UnixMilli(), ServerID: serverID, App: "logs", Command: line, Exit: -1})

	id := newID()
	st := &stream{sess: sess}
	s.mu.Lock()
	s.streams[id] = st
	s.mu.Unlock()

	sudo := shellx.Cmd("sudo", cmd.Args()...).String()
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.pump(id, src.Kind, st, stdout, stderr, sudo)
		exit := sess.Wait()
		s.mu.Lock()
		delete(s.streams, id)
		end := End{StreamID: id}
		switch {
		case st.denied != "":
			end.Error = apperr.New(apperr.CmdNeedsRoot, "the server requires root to read this log")
			end.Sudo = st.denied
		case st.stopped:
		case exit != nil:
			end.Error = endError(exit)
		}
		s.mu.Unlock()
		s.emit.Emit(events.LogsEnd, end)
		s.log.Info("log stream ended", "server", serverID, "app", "logs")
	}()
	return Started{StreamID: id, Sudo: sudo}, nil
}

func endError(err error) *apperr.Error {
	var exit *ssh.ExitError
	if errors.As(err, &exit) {
		if exit.ExitStatus() == 127 {
			return apperr.New(apperr.CmdMissing, "program not installed on the server")
		}
		return apperr.New(apperr.CmdFailed, "the log command stopped")
	}
	return apperr.New(apperr.ConnNotConnected, "connection lost")
}

// pump reads both outputs into batches until the command ends. A stderr
// line that shows a lack of rights stops the stream: following it would
// only ever print that.
func (s *Service) pump(id, kind string, st *stream, stdout, stderr io.Reader, sudo string) {
	var (
		mu      sync.Mutex
		pending []string
	)
	add := func(line string) {
		mu.Lock()
		pending = append(pending, line)
		mu.Unlock()
	}
	flush := func() {
		mu.Lock()
		batch := pending
		if len(batch) > maxBatch {
			batch, pending = batch[:maxBatch], append([]string(nil), batch[maxBatch:]...)
		} else {
			pending = nil
		}
		mu.Unlock()
		if len(batch) > 0 {
			s.emit.Emit(events.LogsLine, Lines{StreamID: id, Lines: batch})
		}
	}

	var readers sync.WaitGroup
	readers.Add(2)
	go func() {
		defer readers.Done()
		_ = readLines(stdout, add)
	}()
	go func() {
		defer readers.Done()
		_ = readLines(stderr, func(line string) {
			if needsRoot(kind, line) {
				s.mu.Lock()
				if st.denied == "" {
					st.denied = sudo
					st.sess.Close() //nolint:errcheck,gosec // ends the stream
				}
				s.mu.Unlock()
			}
			add(line)
		})
	}()
	done := make(chan struct{})
	go func() {
		readers.Wait()
		close(done)
	}()

	tick := time.NewTicker(flushEvery)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			flush()
		case <-done:
			for {
				mu.Lock()
				left := len(pending)
				mu.Unlock()
				if left == 0 {
					return
				}
				flush()
			}
		}
	}
}

// Stop ends a stream; its logs:end follows. Unknown IDs are ignored, the
// stream may have ended on its own.
func (s *Service) Stop(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if st, ok := s.streams[id]; ok {
		st.stopped = true
		st.sess.Close() //nolint:errcheck,gosec // ends the stream
	}
}

// Close stops every stream and waits for them.
func (s *Service) Close() {
	s.mu.Lock()
	for _, st := range s.streams {
		st.stopped = true
		st.sess.Close() //nolint:errcheck,gosec // ends the stream
	}
	s.mu.Unlock()
	s.wg.Wait()
}

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
