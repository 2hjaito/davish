package disk

import (
	"context"
	"errors"
	"path"
	"strconv"
	"strings"

	"davish/internal/apperr"
	"davish/internal/audit"
	"davish/internal/shellx"
)

// Root runs a command as root for a user who is not (sudo.Keeper).
type Root interface {
	Run(ctx context.Context, serverID, app, line string, stdin []byte) (audit.Result, error)
}

// UseRoot lets Disk measure, clean and empty what needs root through r
// (ADR 0018): measuring as root whenever sudo needs no password or the
// session has it, and running the confirmed cleanup or truncate as root.
func (s *Service) UseRoot(r Root) { s.root = r }

// measureAsRoot runs a read-only line as root when that needs no prompt.
// ok is false when sudo needs a password Davish does not have (locked) or
// the user cannot use sudo, so the caller measures as the user instead.
func (s *Service) measureAsRoot(ctx context.Context, serverID, line string) (res audit.Result, ok, locked bool, err error) {
	if s.root == nil {
		return audit.Result{}, false, false, nil
	}
	res, err = s.root.Run(ctx, serverID, app, line, nil)
	switch codeOf(err) {
	case "":
		return res, err == nil, false, err
	case apperr.SudoPasswordRequired, apperr.SudoPasswordWrong:
		return audit.Result{}, false, true, nil
	case apperr.CmdNeedsRoot:
		return audit.Result{}, false, false, nil
	}
	return audit.Result{}, false, false, err
}

// untrusted is what Disk answers when emptying a file as root could be
// sent elsewhere.
func untrusted() error {
	return apperr.New(apperr.InvalidInput, "Disk does not empty this file as root")
}

// checkTruncateAsRoot refuses to empty p as root unless no one but root
// and the login user could swap it for a link: p is a plain file and not
// a link, it really is at p, and every folder on its way belongs to root
// or the login user, its own folder not writable by everyone (as /tmp is;
// a system group such as syslog's on /var/log is the admin's). truncate
// follows links, so a link planted at /tmp/big.log, pointing at
// /etc/shadow, would otherwise empty that.
func (s *Service) checkTruncateAsRoot(ctx context.Context, serverID, p string) error {
	var dirs []string
	for d := path.Dir(p); ; d = path.Dir(d) {
		dirs = append(dirs, d)
		if d == "/" {
			break
		}
	}
	line := "id -u; readlink -f -- " + shellx.Quote(p) + "; " +
		shellx.Cmd("stat", append([]string{"-c", "%u %a %F", "--", p}, dirs...)...).String()
	res, err := s.run.RunLine(ctx, serverID, app, line)
	if err != nil {
		return err
	}
	out := strings.Split(strings.TrimRight(string(res.Stdout), "\n"), "\n")
	if res.Exit != 0 || len(out) != 3+len(dirs) {
		return untrusted()
	}
	login, resolved, file := out[0], out[1], strings.SplitN(out[2], " ", 3)
	if resolved != p || len(file) != 3 || !strings.HasPrefix(file[2], "regular") || protectedFile(resolved) {
		return untrusted()
	}
	for i, l := range out[3:] {
		f := strings.SplitN(l, " ", 3)
		if len(f) != 3 || (f[0] != "0" && f[0] != login) {
			return untrusted()
		}
		mode, err := strconv.ParseUint(f[1], 8, 32)
		if err != nil || (i == 0 && mode&0o002 != 0) {
			return untrusted()
		}
	}
	return nil
}

func codeOf(err error) apperr.Code {
	var e *apperr.Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}
