package audit

import (
	"bytes"
	"context"
	"errors"
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
}

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
	res, err := run(ctx, client, line, stdin)
	exit := res.Exit
	if err != nil {
		exit = -1
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

func run(ctx context.Context, client *ssh.Client, line string, stdin []byte) (Result, error) {
	sess, err := client.NewSession()
	if err != nil {
		return Result{}, apperr.Wrap(apperr.ConnNotConnected, "cannot open a session", err)
	}
	defer sess.Close() //nolint:errcheck // closed either way

	stop := context.AfterFunc(ctx, func() { sess.Close() }) //nolint:errcheck,gosec // unblocks Run
	defer stop()

	var stdout, stderr bytes.Buffer
	if stdin != nil {
		sess.Stdin = bytes.NewReader(stdin)
	}
	sess.Stdout = &stdout
	sess.Stderr = &stderr
	err = sess.Run(line)
	if ctx.Err() != nil {
		return Result{}, ctx.Err()
	}
	res := Result{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}
	var exit *ssh.ExitError
	switch {
	case err == nil:
		return res, nil
	case errors.As(err, &exit):
		res.Exit = exit.ExitStatus()
		return res, nil
	}
	return Result{}, apperr.Wrap(apperr.ConnNotConnected, "command did not finish", err)
}
