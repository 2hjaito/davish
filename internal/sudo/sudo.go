// Package sudo runs commands as root for a user who is not, with the sudo
// password typed once per session and kept in memory only (ADR 0018).
//
// The password goes to sudo on stdin, never on a command line, so it is in
// neither the audit log nor ps. It is wiped when the connection drops, the
// app locks, the server is removed, or the user asks.
package sudo

import (
	"bytes"
	"context"
	"strings"
	"sync"

	"davish/internal/apperr"
	"davish/internal/audit"
	"davish/internal/conn"
	"davish/internal/events"
	"davish/internal/lock"
	"davish/internal/shellx"
)

// Runner runs a command on a server and records it in the audit log.
type Runner interface {
	RunInput(ctx context.Context, serverID, app, line string, stdin []byte) (audit.Result, error)
}

// How a server's user reaches root.
const (
	// Root: the user is root, commands run as they are.
	Root = "root"
	// NoPassword: sudo runs without asking.
	NoPassword = "nopassword"
	// Password: sudo asks, and Davish has no password for it yet.
	Password = "password"
	// Ready: sudo asks, and Davish has the password for this session.
	Ready = "ready"
	// None: no sudo, or the user may not use it; root's commands go to a
	// terminal.
	None = "none"
)

// Keeper holds each server's sudo password for the session.
type Keeper struct {
	run Runner
	mu  sync.Mutex
	pw  map[string][]byte
	// root caches whether a server's user is root.
	root map[string]bool
}

// New creates a keeper that runs through run.
func New(run Runner) *Keeper {
	return &Keeper{run: run, pw: map[string][]byte{}, root: map[string]bool{}}
}

func wipe(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// Forget wipes the server's password.
func (k *Keeper) Forget(serverID string) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if p, ok := k.pw[serverID]; ok {
		wipe(p)
		delete(k.pw, serverID)
	}
	delete(k.root, serverID)
}

// ForgetAll wipes every password.
func (k *Keeper) ForgetAll() {
	k.mu.Lock()
	defer k.mu.Unlock()
	for id, p := range k.pw {
		wipe(p)
		delete(k.pw, id)
	}
	k.root = map[string]bool{}
}

// Observe forgets passwords on the events that end a session: a server
// that is no longer connected, and the app locking. The app passes every
// event it emits.
func (k *Keeper) Observe(name string, payload any) {
	switch name {
	case events.ConnStatus:
		if st, ok := payload.(conn.Status); ok && st.State != conn.StateConnected {
			k.Forget(st.ServerID)
		}
	case events.LockState:
		if st, ok := payload.(lock.State); ok && st.Locked {
			k.ForgetAll()
		}
	}
}

func (k *Keeper) password(serverID string) []byte {
	k.mu.Lock()
	defer k.mu.Unlock()
	p, ok := k.pw[serverID]
	if !ok {
		return nil
	}
	return append([]byte(nil), p...)
}

// isRoot tells whether the server's user is root, asking once.
func (k *Keeper) isRoot(ctx context.Context, serverID, app string) (bool, error) {
	k.mu.Lock()
	r, ok := k.root[serverID]
	k.mu.Unlock()
	if ok {
		return r, nil
	}
	res, err := k.run.RunInput(ctx, serverID, app, "id -u", nil)
	if err != nil {
		return false, err
	}
	r = strings.TrimSpace(string(res.Stdout)) == "0"
	k.mu.Lock()
	k.root[serverID] = r
	k.mu.Unlock()
	return r, nil
}

// Lines that run line through sudo: without a password, and with the
// password as the first line of stdin. -k makes sudo read the password
// every time, so it never reaches the command's stdin when sudo still
// remembers an earlier one; -p ” keeps the prompt out of the output.
func noPasswordLine(line string) string {
	return shellx.Cmd("sudo", "-n", "--", "sh", "-c", line).String()
}

func passwordLine(line string) string {
	return shellx.Cmd("sudo", "-S", "-k", "-p", "", "--", "sh", "-c", line).String()
}

// withPassword is pw's line then stdin, in a buffer of its own so wiping
// it touches nothing else.
func withPassword(pw, stdin []byte) []byte {
	in := make([]byte, 0, len(pw)+1+len(stdin))
	in = append(in, pw...)
	in = append(in, '\n')
	return append(in, stdin...)
}

func needsPassword(stderr []byte) bool {
	return bytes.Contains(stderr, []byte("sudo: a password is required"))
}

func noSudo(res audit.Result) bool {
	e := string(res.Stderr)
	return (res.Exit == 127 && strings.Contains(e, "sudo")) ||
		strings.Contains(e, "is not in the sudoers file") ||
		strings.Contains(e, "may not run sudo") ||
		strings.Contains(e, "is not allowed to execute")
}

func wrongPassword(stderr []byte) bool {
	return bytes.Contains(stderr, []byte("incorrect password attempt"))
}

// Run runs line as root on the server: as it is when the user is root,
// else through sudo, without a password when sudo allows, else with the
// session's password. A non-zero exit of line itself is in Result, as
// audit.Runner does.
//
// Errors: SUDO_PASSWORD_REQUIRED when sudo asks and Davish has no password
// yet (the UI asks, then calls Run again); SUDO_PASSWORD_WRONG when the
// password no longer works (it is forgotten); CMD_NEEDS_ROOT when the user
// cannot use sudo at all (the UI offers a terminal).
func (k *Keeper) Run(ctx context.Context, serverID, app, line string, stdin []byte) (audit.Result, error) {
	root, err := k.isRoot(ctx, serverID, app)
	if err != nil {
		return audit.Result{}, err
	}
	if root {
		return k.run.RunInput(ctx, serverID, app, line, stdin)
	}
	res, err := k.run.RunInput(ctx, serverID, app, noPasswordLine(line), stdin)
	if err != nil {
		return res, err
	}
	switch {
	case noSudo(res):
		return res, apperr.New(apperr.CmdNeedsRoot, "this user cannot use sudo")
	case !needsPassword(res.Stderr):
		return res, nil
	}
	pw := k.password(serverID)
	if pw == nil {
		return res, apperr.New(apperr.SudoPasswordRequired, "sudo asks for a password")
	}
	in := withPassword(pw, stdin)
	defer wipe(in)
	wipe(pw)
	res, err = k.run.RunInput(ctx, serverID, app, passwordLine(line), in)
	if err != nil {
		return res, err
	}
	if wrongPassword(res.Stderr) {
		k.Forget(serverID)
		return res, apperr.New(apperr.SudoPasswordWrong, "sudo did not accept the password")
	}
	return res, nil
}

// State tells how the server's user reaches root, asking sudo without a
// password; it never prompts.
func (k *Keeper) State(ctx context.Context, serverID string) (string, error) {
	root, err := k.isRoot(ctx, serverID, "sudo")
	if err != nil {
		return "", err
	}
	if root {
		return Root, nil
	}
	res, err := k.run.RunInput(ctx, serverID, "sudo", "sudo -n true", nil)
	if err != nil {
		return "", err
	}
	switch {
	case res.Exit == 0:
		return NoPassword, nil
	case noSudo(res):
		return None, nil
	case k.password(serverID) != nil:
		return Ready, nil
	}
	return Password, nil
}

// SetPassword checks pw with sudo -v and keeps it for the session only
// when sudo accepts it.
func (k *Keeper) SetPassword(ctx context.Context, serverID string, pw []byte) error {
	if len(pw) == 0 || bytes.ContainsAny(pw, "\n\r") {
		return apperr.New(apperr.InvalidInput, "not a password")
	}
	in := withPassword(pw, nil)
	defer wipe(in)
	res, err := k.run.RunInput(ctx, serverID, "sudo", shellx.Cmd("sudo", "-S", "-k", "-p", "", "-v").String(), in)
	if err != nil {
		return err
	}
	switch {
	case noSudo(res):
		return apperr.New(apperr.CmdNeedsRoot, "this user cannot use sudo")
	case res.Exit != 0:
		return apperr.New(apperr.SudoPasswordWrong, "sudo did not accept the password")
	}
	k.mu.Lock()
	if old, ok := k.pw[serverID]; ok {
		wipe(old)
	}
	k.pw[serverID] = append([]byte(nil), pw...)
	k.mu.Unlock()
	return nil
}
