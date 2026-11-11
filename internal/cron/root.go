package cron

import (
	"context"
	"errors"
	"strings"

	"davish/internal/apperr"
	"davish/internal/audit"
	"davish/internal/shellx"
	"davish/internal/sudo"
)

// SourceRoot is root's crontab, read and edited through sudo.
const SourceRoot = "root"

// rootHome is where root's crontab is backed up before a change: root's
// home on every Linux distribution Davish supports.
const rootHome = "/root"

// Root runs a command as root for a user who is not (sudo.Keeper).
type Root interface {
	Run(ctx context.Context, serverID, app, line string, stdin []byte) (audit.Result, error)
	State(ctx context.Context, serverID string) (string, error)
}

// UseRoot lets Cron read what the user may not, and show and edit root's
// crontab, through r (ADR 0018). Jobs in /etc/crontab and /etc/cron.d
// stay read-only here; they open in Editor.
func (s *Service) UseRoot(r Root) { s.root = r }

// rootAccess is how Cron may reach root on a server right now.
type rootAccess int

const (
	// noRoot: no sudo, or the user is root already and root's crontab is
	// theirs.
	noRoot rootAccess = iota
	// rootLocked: sudo asks for a password Davish does not have yet.
	rootLocked
	// rootOpen: commands run as root without asking.
	rootOpen
)

// asRoot runs line as root, with stdin. The access tells how it went;
// err is only for a failure that is neither "no sudo" nor "password
// needed", such as a dropped connection or a wrong password.
func (s *Service) asRoot(ctx context.Context, serverID, line string, stdin []byte) (audit.Result, rootAccess, error) {
	if s.root == nil {
		return audit.Result{}, noRoot, nil
	}
	res, err := s.root.Run(ctx, serverID, app, line, stdin)
	switch codeOf(err) {
	case "":
		if err != nil {
			return res, noRoot, err
		}
		return res, rootOpen, nil
	case apperr.SudoPasswordRequired, apperr.SudoPasswordWrong:
		// A wrong password is forgotten, so it is asked again.
		return res, rootLocked, nil
	case apperr.CmdNeedsRoot:
		return res, noRoot, nil
	}
	return res, noRoot, err
}

// rootCrontabCmd reads root's crontab.
var rootCrontabCmd = shellx.Cmd("crontab", "-u", "root", "-l")

// rootCrontab reads root's crontab as root, and whether there is one.
// The access is noRoot when the user is root (their own crontab is
// root's) or cannot use sudo.
func (s *Service) rootCrontab(ctx context.Context, serverID string) (text string, exists bool, access rootAccess, err error) {
	if s.root == nil {
		return "", false, noRoot, nil
	}
	// Root's crontab is the user's own for root. Users in no admin group
	// are not offered the prompt: sudo would most likely refuse them.
	state, err := s.root.State(ctx, serverID)
	if err != nil {
		return "", false, noRoot, err
	}
	switch state {
	case sudo.Password:
		return "", false, rootLocked, nil
	case sudo.NoPassword, sudo.Ready:
	default:
		return "", false, noRoot, nil
	}
	res, access, err := s.asRoot(ctx, serverID, rootCrontabCmd.String(), nil)
	if err != nil || access != rootOpen {
		return "", false, access, err
	}
	switch {
	case res.Exit == 0:
		return string(res.Stdout), true, rootOpen, nil
	case strings.Contains(string(res.Stderr), "no crontab for"):
		return "", false, rootOpen, nil
	}
	// sudo may run some commands only: root's crontab stays out of sight.
	return "", false, noRoot, nil
}

// readAsRoot reads a system crontab the user may not read. A symlink is
// not followed, so a link in /etc/cron.d cannot show another root file,
// and at most 1 MiB is read, as over SFTP.
func (s *Service) readAsRoot(ctx context.Context, serverID, path string) (string, rootAccess, error) {
	res, access, err := s.asRoot(ctx, serverID, readRootLine(path), nil)
	if err != nil || access != rootOpen {
		return "", access, err
	}
	if res.Exit != 0 {
		return "", noRoot, nil
	}
	return string(res.Stdout), rootOpen, nil
}

// rootError is what an edit of root's crontab gets when root cannot be
// reached: the UI asks for the password, or says sudo is not allowed.
func rootError(a rootAccess) error {
	if a == rootLocked {
		return apperr.New(apperr.SudoPasswordRequired, "sudo asks for a password")
	}
	return apperr.New(apperr.CmdNeedsRoot, "root's crontab needs sudo")
}

func readRootLine(path string) string {
	return shellx.Cmd("test", "-f", path).String() + " && " + shellx.Cmd("test", "!", "-L", path).String() +
		" && " + shellx.Cmd("head", "-c", "1048576", "--", path).String()
}

// errUnread marks a system crontab root could not read either.
var errUnread = errors.New("unreadable")

func codeOf(err error) apperr.Code {
	var e *apperr.Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}
