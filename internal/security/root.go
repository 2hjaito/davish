package security

import (
	"context"
	"errors"

	"davish/internal/apperr"
	"davish/internal/audit"
	"davish/internal/shellx"
	"davish/internal/sudo"
)

// Root runs a command as root for a user who is not (sudo.Keeper).
type Root interface {
	Run(ctx context.Context, serverID, app, line string, stdin []byte) (audit.Result, error)
	State(ctx context.Context, serverID string) (string, error)
}

// UseRoot lets the Security app read what needs root, and block, unblock,
// ban and unban, through r (ADR 0018), not only when sudo asks nothing.
// What could lock the user out of the server (turning ufw on, password
// logins off) stays in a terminal.
func (s *Service) UseRoot(r Root) { s.root = r }

// rootSections are the parts of reportLine that need root to be whole.
var rootSections = []string{"journal", "authlog", "lastb", "ufw", "f2b"}

// rootReportLine reads rootSections, run as root.
const rootReportLine = `echo --journal--; journalctl -u ssh -u sshd --since '-7 days' --no-pager -o short-iso 2>/dev/null | grep -E 'Accepted |Failed password|Invalid user' | tail -n 5000` +
	`; echo --authlog--; [ -r /var/log/auth.log ] && grep -hE 'sshd.*(Accepted |Failed password|Invalid user)' /var/log/auth.log | tail -n 5000` +
	`; echo --lastb--; lastb -n 20 -i -w 2>&1 | grep -v '^$' | grep -v '^btmp begins'` +
	`; echo --ufw--; command -v ufw >/dev/null && { echo INSTALLED; ufw status 2>&1 | head -n 1000; }` +
	`; echo --f2b--; command -v fail2ban-client >/dev/null && { echo INSTALLED; fail2ban-client status sshd 2>&1 | head -n 30; }` +
	`; true`

// SudoReadCommand is exactly what reading as root runs, for the prompt.
var SudoReadCommand = shellx.Cmd("sudo", "sh", "-c", rootReportLine).String()

// needsRoot tells whether the report as the user misses what root sees.
func (r Report) needsRoot() bool {
	return !r.Root && (r.Source == "" || r.LastbDenied || r.UFW.NeedsRoot || r.Fail2ban.NeedsRoot)
}

// canSudo tells whether an action may run through sudo: right away, or
// once the session's password is given. Users in no admin group are not
// offered it: sudo would most likely refuse them.
func canSudo(state string) bool {
	return state == sudo.NoPassword || state == sudo.Ready || state == sudo.Password
}

// readAsRoot reads rootSections as root when sudo needs no prompt. ok is
// false otherwise; locked when the session's password would do it.
func (s *Service) readAsRoot(ctx context.Context, serverID, state string) (sec map[string][]string, ok, locked bool, err error) {
	switch state {
	case sudo.Password:
		return nil, false, true, nil
	case sudo.NoPassword, sudo.Ready:
	default:
		return nil, false, false, nil
	}
	res, err := s.root.Run(ctx, serverID, app, rootReportLine, nil)
	switch codeOf(err) {
	case "":
		if err != nil {
			return nil, false, false, err
		}
		return sections(res.Stdout), true, false, nil
	case apperr.SudoPasswordRequired, apperr.SudoPasswordWrong:
		return nil, false, true, nil
	}
	return nil, false, false, nil
}

func codeOf(err error) apperr.Code {
	var e *apperr.Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}
