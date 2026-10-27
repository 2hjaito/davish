// Package term runs interactive shells on a PTY over the server's shared
// SSH connection and streams their output to the frontend.
package term

import (
	"context"
	"crypto/rand"
	_ "embed"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

	"davish/internal/apperr"
	"davish/internal/events"
	"davish/internal/shellx"
)

//go:embed prompt.bash
var promptScript string

// promptCommand starts bash with Davish's prompt. The script travels as one
// quoted argument and bash reads it through process substitution, so nothing
// is written on the server (docs/adr/0004). The login shell runs this line,
// which is why it is only used when that shell is bash.
var promptCommand = "exec bash --rcfile <(printf '%s\\n' " + shellx.Quote(promptScript) + ") -i"

// ClientSource hands out the live SSH client of a server.
type ClientSource interface {
	Client(serverID string) (*ssh.Client, error)
}

// Emitter pushes events to the frontend.
type Emitter interface {
	Emit(name string, payload any)
}

// Data is the term:data payload. Output is base64 because a PTY can emit
// bytes that are not valid UTF-8, or split a character across reads.
type Data struct {
	ID  string `json:"id"`
	B64 string `json:"b64"`
}

// Exit is the term:exit payload; Code is -1 when the shell died without one.
type Exit struct {
	ID   string `json:"id"`
	Code int    `json:"code"`
}

// flushEvery batches output so `cat bigfile` does not flood the webview.
const flushEvery = 16 * time.Millisecond

// Service owns every terminal session.
type Service struct {
	clients ClientSource
	emit    Emitter
	log     *slog.Logger

	// mu guards sessions.
	mu       sync.Mutex
	sessions map[string]*session
	wg       sync.WaitGroup
}

type session struct {
	serverID string
	ssh      *ssh.Session
	stdin    io.WriteCloser
}

// New creates the service; call Close on shutdown.
func New(clients ClientSource, emit Emitter, log *slog.Logger) *Service {
	return &Service{clients: clients, emit: emit, log: log, sessions: map[string]*session{}}
}

// Start opens a login shell on serverID sized cols×rows and returns its ID.
// With prompt, a bash login shell gets Davish's powerline prompt; other
// shells open untouched.
func (s *Service) Start(ctx context.Context, serverID string, cols, rows int, prompt bool) (string, error) {
	if err := checkSize(cols, rows); err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	client, err := s.clients.Client(serverID)
	if err != nil {
		return "", err
	}
	command := ""
	if prompt && loginShellIsBash(client) {
		command = promptCommand
	}
	sess, err := client.NewSession()
	if err != nil {
		return "", apperr.Wrap(apperr.ConnNotConnected, "cannot open a session", err)
	}
	stdin, stdout, err := startShell(sess, cols, rows, command)
	if err != nil {
		sess.Close() //nolint:errcheck,gosec // failed to start
		return "", err
	}

	id, err := newID()
	if err != nil {
		sess.Close() //nolint:errcheck,gosec // failed to start
		return "", err
	}
	s.mu.Lock()
	s.sessions[id] = &session{serverID: serverID, ssh: sess, stdin: stdin}
	s.mu.Unlock()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.pump(id, stdout)
		code := exitCode(sess.Wait())
		s.mu.Lock()
		delete(s.sessions, id)
		s.mu.Unlock()
		s.emit.Emit(events.TermExit, Exit{ID: id, Code: code})
		s.log.Info("terminal closed", "server", serverID, "code", code)
	}()
	s.log.Info("terminal opened", "server", serverID)
	return id, nil
}

// loginShellIsBash asks the server for the user's login shell. Any failure
// counts as no, which falls back to the plain shell.
func loginShellIsBash(client *ssh.Client) bool {
	sess, err := client.NewSession()
	if err != nil {
		return false
	}
	defer sess.Close() //nolint:errcheck // probe only
	out, err := sess.Output(`printf %s "$SHELL"`)
	return err == nil && path.Base(strings.TrimSpace(string(out))) == "bash"
}

// startShell runs command on a PTY, or the login shell when it is empty.
func startShell(sess *ssh.Session, cols, rows int, command string) (io.WriteCloser, io.Reader, error) {
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if err := sess.RequestPty("xterm-256color", rows, cols, modes); err != nil {
		return nil, nil, apperr.Wrap(apperr.ConnNotConnected, "cannot allocate a PTY", err)
	}
	stdin, err := sess.StdinPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("stdin pipe: %w", err)
	}
	// With a PTY the shell's stderr arrives on stdout too.
	stdout, err := sess.StdoutPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("stdout pipe: %w", err)
	}
	start := sess.Shell
	if command != "" {
		start = func() error { return sess.Start(command) }
	}
	if err := start(); err != nil {
		return nil, nil, apperr.Wrap(apperr.ConnNotConnected, "cannot start the shell", err)
	}
	return stdin, stdout, nil
}

// pump forwards output in batches of at most flushEvery until EOF.
func (s *Service) pump(id string, r io.Reader) {
	chunks := make(chan []byte, 64)
	go func() {
		defer close(chunks)
		buf := make([]byte, 32*1024)
		for {
			n, err := r.Read(buf)
			if n > 0 {
				chunks <- append([]byte(nil), buf[:n]...)
			}
			if err != nil {
				return
			}
		}
	}()

	var pending []byte
	flush := func() {
		if len(pending) > 0 {
			s.emit.Emit(events.TermData, Data{ID: id, B64: base64.StdEncoding.EncodeToString(pending)})
			pending = pending[:0]
		}
	}
	tick := time.NewTicker(flushEvery)
	defer tick.Stop()
	for {
		select {
		case chunk, ok := <-chunks:
			if !ok {
				flush()
				return
			}
			pending = append(pending, chunk...)
		case <-tick.C:
			flush()
		}
	}
}

func (s *Service) get(id string) (*session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[id]
	if !ok {
		return nil, apperr.New(apperr.TermClosed, "terminal session is closed")
	}
	return sess, nil
}

// Write sends keystrokes or pasted text to the shell.
func (s *Service) Write(id, data string) error {
	sess, err := s.get(id)
	if err != nil {
		return err
	}
	if _, err := io.WriteString(sess.stdin, data); err != nil {
		return apperr.Wrap(apperr.TermClosed, "terminal session is closed", err)
	}
	return nil
}

// Resize tells the remote PTY the new window size.
func (s *Service) Resize(id string, cols, rows int) error {
	if err := checkSize(cols, rows); err != nil {
		return err
	}
	sess, err := s.get(id)
	if err != nil {
		return err
	}
	if err := sess.ssh.WindowChange(rows, cols); err != nil {
		return apperr.Wrap(apperr.TermClosed, "terminal session is closed", err)
	}
	return nil
}

// Close ends the session; term:exit follows.
func (s *Service) Close(id string) error {
	sess, err := s.get(id)
	if err != nil {
		return err
	}
	sess.ssh.Close() //nolint:errcheck,gosec // ends the session either way
	return nil
}

// CloseAll ends every session and waits for their goroutines.
func (s *Service) CloseAll() {
	s.mu.Lock()
	for _, sess := range s.sessions {
		sess.ssh.Close() //nolint:errcheck,gosec // shutting down
	}
	s.mu.Unlock()
	s.wg.Wait()
}

func checkSize(cols, rows int) error {
	if cols < 1 || cols > 1000 || rows < 1 || rows > 1000 {
		return apperr.New(apperr.InvalidInput, "terminal size out of range")
	}
	return nil
}

func exitCode(err error) int {
	var ee *ssh.ExitError
	switch {
	case err == nil:
		return 0
	case errors.As(err, &ee):
		return ee.ExitStatus()
	}
	return -1
}

func newID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate session id: %w", err)
	}
	return hex.EncodeToString(b), nil
}
