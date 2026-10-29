package audit

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

	"davish/internal/apperr"
	"davish/internal/shellx"
)

// ClientSource hands out a server's live SSH client.
type ClientSource interface {
	Client(serverID string) (*ssh.Client, error)
}

// Result is what a finished command printed and how it exited.
type Result struct {
	Stdout []byte
	Stderr []byte
	Exit   int
}

// Runner runs one-shot commands on servers and records each one in the
// log, so services cannot run a command the user does not get to see.
type Runner struct {
	conns ClientSource
	log   *Log
	// slog, when set, gets every command too, for the support log file.
	slog *slog.Logger
}

// LogTo also writes every command, its exit and duration to l. Only the
// command line is written: never its input or output.
func (r *Runner) LogTo(l *slog.Logger) { r.slog = l }

// NewRunner creates a runner that records into log.
func NewRunner(conns ClientSource, log *Log) *Runner {
	return &Runner{conns: conns, log: log}
}

// Run executes cmd on serverID for app. A non-zero exit is not an error:
// it is in Result.Exit for the caller to interpret. The error is set only
// when the command could not run or finish.
func (r *Runner) Run(ctx context.Context, serverID, app string, cmd shellx.Command) (Result, error) {
	return r.RunLine(ctx, serverID, app, cmd.String())
}

// RunLine is Run for a command line already rendered by shellx, such as
// shellx.InDir or a few commands joined with &&. It must never contain
// text that did not go through shellx.
func (r *Runner) RunLine(ctx context.Context, serverID, app, line string) (Result, error) {
	return r.RunInput(ctx, serverID, app, line, nil)
}

// RunInput is RunLine with stdin, such as a new crontab for "crontab -".
// Only the command line is recorded, never the input.
func (r *Runner) RunInput(ctx context.Context, serverID, app, line string, stdin []byte) (Result, error) {
	client, err := r.conns.Client(serverID)
	if err != nil {
		return Result{}, err
	}
	start := time.Now()
	// Lines are written for sh. sshd hands them to the user's login shell,
	// which may be zsh (no word splitting of $vars) or fish (other syntax
	// altogether), so they run under sh whatever that shell is. The log
	// keeps the line as written.
	res, err := run(ctx, client, shellx.Cmd("sh", "-c", line).String(), stdin)
	exit := res.Exit
	if err != nil {
		exit = -1
	}
	if r.slog != nil {
		attrs := []any{"server", serverID, "app", app, "exit", exit, "ms", time.Since(start).Milliseconds(), "cmd", clip(line)}
		if err != nil {
			attrs = append(attrs, "err", err.Error())
			r.slog.Warn("command failed", attrs...)
		} else {
			r.slog.Info("command", attrs...)
		}
	}
	r.log.Add(Entry{
		Time:     start.UnixMilli(),
		ServerID: serverID,
		App:      app,
		Command:  line,
		Exit:     exit,
		DurMs:    time.Since(start).Milliseconds(),
	})
	return res, err
}

func clip(s string) string {
	if len(s) > 300 {
		return s[:300] + "…"
	}
	return s
}

// DefaultTimeout bounds a command whose caller set no deadline: nothing a
// service runs should take longer, and a server that never answers must
// not leave a window loading for ever. Longer work sets its own deadline.
const DefaultTimeout = 3 * time.Minute

// run executes line and returns when it finishes or ctx ends, whichever
// comes first. On a timeout it does not wait for the server: closing the
// session can hang as long as the command does (a stuck command, or a
// shell startup file that never returns), so the session is closed in the
// background and the caller gets CMD_TIMEOUT at once.
func run(ctx context.Context, client *ssh.Client, line string, stdin []byte) (Result, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultTimeout)
		defer cancel()
	}
	type outcome struct {
		res Result
		err error
	}
	done := make(chan outcome, 1)
	var (
		mu        sync.Mutex
		sess      *ssh.Session
		abandoned bool
	)
	go func() {
		// Opening a session is a round trip too, and can hang the same way.
		s, err := client.NewSession()
		if err != nil {
			done <- outcome{err: apperr.Wrap(apperr.ConnNotConnected, "cannot open a session", err)}
			return
		}
		mu.Lock()
		if abandoned {
			mu.Unlock()
			s.Close() //nolint:errcheck,gosec // nobody waits for it
			return
		}
		sess = s
		mu.Unlock()
		defer s.Close() //nolint:errcheck // closed either way
		var stdout, stderr bytes.Buffer
		if stdin != nil {
			s.Stdin = bytes.NewReader(stdin)
		}
		s.Stdout = &stdout
		s.Stderr = &stderr
		err = s.Run(line)
		res := Result{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}
		var exit *ssh.ExitError
		switch {
		case err == nil:
			done <- outcome{res: res}
		case errors.As(err, &exit):
			res.Exit = exit.ExitStatus()
			done <- outcome{res: res}
		default:
			done <- outcome{err: apperr.Wrap(apperr.ConnNotConnected, "command did not finish", err)}
		}
	}()

	select {
	case o := <-done:
		return o.res, o.err
	case <-ctx.Done():
		mu.Lock()
		abandoned = true
		if sess != nil {
			go sess.Close() //nolint:errcheck,gosec // may block until the server lets go
		}
		mu.Unlock()
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return Result{}, apperr.Wrap(apperr.CmdTimeout, "the server did not finish the command in time", ctx.Err())
		}
		return Result{}, ctx.Err()
	}
}
