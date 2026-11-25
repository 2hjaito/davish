package security

import (
	"context"
	"strings"
	"time"

	"davish/internal/apperr"
	"davish/internal/audit"
	"davish/internal/shellx"
)

// Turning password logins off from the app (ADR 0029). The change is a
// drop-in file, so nothing the user wrote is edited. It is first checked
// with sshd, and after the reload a second connection that uses only the
// key must still get in, or the drop-in is removed again.

// KeyChecker logs in to a server with its key only: no password, not even
// keyboard-interactive. conn.Manager does it over a fresh connection.
type KeyChecker interface {
	CheckKeyLogin(ctx context.Context, serverID string) error
}

// UseKeys lets the Security app check that the key still logs in around
// turning password logins off.
func (s *Service) UseKeys(k KeyChecker) { s.keys = k }

// afterReload is how many times, and how far apart, the key is tried after
// sshd reloads: a reload restarts sshd, which refuses connections for a
// moment, and that must not look like a key that stopped working.
var (
	afterReloadTries = 6
	afterReloadWait  = 500 * time.Millisecond
)

const dropIn = "/etc/ssh/sshd_config.d/00-davish-no-password.conf"

// The reasons PasswordStatus gives for not offering the change.
const (
	BlockNoInclude = "noInclude" // sshd_config does not read sshd_config.d
	BlockMatch     = "match"     // a Match block sets password or keyboard-interactive
	BlockMethods   = "methods"   // AuthenticationMethods asks for a password
	BlockControl   = "control"   // the server is reached through a control socket
	BlockPassword  = "password"  // Davish itself logs in with the password
)

// PasswordStatus is what sshd really does now, and whether the change is
// safe to offer.
type PasswordStatus struct {
	// Password and KbdInteractive are sshd -T's values, "yes", "no" or "".
	Password       string `json:"password"`
	KbdInteractive string `json:"kbdInteractive"`
	// Ours is set when Davish's own drop-in is there, so it can be undone.
	Ours bool `json:"ours"`
	// Blocked lists why the change is not offered, as the Block* codes.
	Blocked []string `json:"blocked"`
}

// PasswordPlan is the commands of the change, in the forms of Plan.
type PasswordPlan struct {
	// Status is the read-only command that tells what sshd does now.
	Status  Plan `json:"status"`
	Disable Plan `json:"disable"`
	// Enable undoes it: removes the drop-in and reloads sshd.
	Enable Plan `json:"enable"`
}

const includeCheck = `[ -d /etc/ssh/sshd_config.d ] && grep -qiE "^[[:space:]]*include[[:space:]].*sshd_config\.d" /etc/ssh/sshd_config`

const reload = `{ systemctl reload ssh 2>/dev/null || systemctl reload sshd 2>/dev/null; }`

// statusLine reads, as root, what sshd does now and the files it reads.
const statusLine = `echo --eff--; sshd -T 2>/dev/null | grep -iE "^(passwordauthentication|kbdinteractiveauthentication|authenticationmethods) "` +
	`; echo --include--; ` + includeCheck + ` && echo yes` +
	`; echo --ours--; [ -e ` + dropIn + ` ] && echo yes` +
	`; echo --cfg--; cat /etc/ssh/sshd_config /etc/ssh/sshd_config.d/*.conf 2>/dev/null` +
	`; true`

// disableLine does the change; it prints a DAVISH_ marker and exits with
// a number when it undid itself.
const disableLine = `f=` + dropIn + `
` + includeCheck + ` || { echo DAVISH_NO_INCLUDE; exit 3; }
b=/var/backups/davish-sshd-$(date +%Y%m%d-%H%M%S)
mkdir -p "$b" && cp -a /etc/ssh/sshd_config /etc/ssh/sshd_config.d "$b"/ || { echo DAVISH_BACKUP; exit 3; }
printf "PasswordAuthentication no\nKbdInteractiveAuthentication no\n" > "$f"
sshd -t || { rm -f "$f"; echo DAVISH_SSHD_T; exit 4; }
t=$(sshd -T) && echo "$t" | grep -qix "passwordauthentication no" && ! echo "$t" | grep -qixE "kbdinteractiveauthentication yes" || { rm -f "$f"; echo DAVISH_OVERRIDDEN; exit 5; }
` + reload + ` || { rm -f "$f"; echo DAVISH_RELOAD; exit 6; }
echo "backup in $b"`

// enableLine removes the drop-in and reloads sshd, checked first.
const enableLine = `rm -f ` + dropIn + `
sshd -t && ` + reload

// PlanPassword gives the commands of turning password logins off and on.
func PlanPassword(root bool) PasswordPlan {
	form := func(line string) Plan {
		p := Plan{
			Command:  "sudo -n sh -c " + shellx.Quote(line),
			Terminal: "sudo sh -c " + shellx.Quote(line),
		}
		if root {
			p.Command = "sh -c " + shellx.Quote(line)
		}
		return p
	}
	return PasswordPlan{Status: form(statusLine), Disable: form(disableLine), Enable: form(enableLine)}
}

// ReadPasswordStatus reads what sshd does now and whether the change is
// safe, as root. keyLogin is whether Davish itself logs in with a key,
// and control whether it goes through a control socket.
func (s *Service) ReadPasswordStatus(ctx context.Context, serverID string, keyLogin, control bool) (PasswordStatus, error) {
	var res audit.Result
	var err error
	if s.root != nil {
		res, err = s.root.Run(ctx, serverID, app, statusLine, nil)
	} else {
		res, err = s.run.RunLine(ctx, serverID, app, statusLine)
	}
	if err != nil {
		return PasswordStatus{}, err
	}
	st := ParsePasswordStatus(res.Stdout)
	if !keyLogin {
		st.Blocked = append(st.Blocked, BlockPassword)
	}
	if control {
		st.Blocked = append(st.Blocked, BlockControl)
	}
	return st, nil
}

// ParsePasswordStatus reads statusLine's output.
func ParsePasswordStatus(out []byte) PasswordStatus {
	sec := sections(out)
	st := PasswordStatus{Blocked: []string{}}
	methods := ""
	for _, l := range sec["eff"] {
		f := strings.Fields(l)
		if len(f) < 2 {
			continue
		}
		v := strings.ToLower(strings.Join(f[1:], " "))
		switch strings.ToLower(f[0]) {
		case "passwordauthentication":
			st.Password = v
		case "kbdinteractiveauthentication":
			st.KbdInteractive = v
		case "authenticationmethods":
			methods = v
		}
	}
	st.Ours = first(sec["ours"]) == "yes"
	if first(sec["include"]) != "yes" {
		st.Blocked = append(st.Blocked, BlockNoInclude)
	}
	if strings.Contains(methods, "password") || strings.Contains(methods, "keyboard-interactive") {
		st.Blocked = append(st.Blocked, BlockMethods)
	}
	if matchSetsPassword(sec["cfg"]) {
		st.Blocked = append(st.Blocked, BlockMatch)
	}
	return st
}

// matchSetsPassword tells whether a Match block turns password or
// keyboard-interactive logins on, or asks for them in
// AuthenticationMethods: the drop-in's global "no" would then not reach
// the users it matches. A Match block ends at the next Match line.
func matchSetsPassword(lines []string) bool {
	in := false
	for _, l := range lines {
		f := strings.Fields(strings.TrimSpace(l))
		if len(f) == 0 || strings.HasPrefix(f[0], "#") {
			continue
		}
		key := strings.ToLower(f[0])
		if key == "match" {
			in = true
			continue
		}
		if !in || len(f) < 2 {
			continue
		}
		v := strings.ToLower(strings.Join(f[1:], " "))
		switch key {
		case "passwordauthentication", "kbdinteractiveauthentication":
			if v == "yes" {
				return true
			}
		case "authenticationmethods":
			if strings.Contains(v, "password") || strings.Contains(v, "keyboard-interactive") {
				return true
			}
		}
	}
	return false
}

// runPlan runs one of p's commands the way Apply does: through the
// session's sudo when it is the sudo form, as given otherwise.
func (s *Service) runPlan(ctx context.Context, serverID string, asRoot, withSudo Plan, command string) (audit.Result, error) {
	if command != asRoot.Command && command != withSudo.Command {
		return audit.Result{}, apperr.New(apperr.FSStale, "the command changed since it was confirmed")
	}
	if command == withSudo.Command && s.root != nil {
		return s.root.Run(ctx, serverID, app, asRoot.Command, nil)
	}
	return s.run.RunLine(ctx, serverID, app, command)
}

// DisablePassword turns password logins off. command must be what
// PlanPassword gave. The key is tried before, so that a key that does not
// work stops it, and after, so that a change that locks the key out is
// undone through the session that is still open.
func (s *Service) DisablePassword(ctx context.Context, serverID, command string) error {
	if s.keys == nil {
		return apperr.New(apperr.PlatformUnsupported, "key login cannot be checked here")
	}
	if err := s.keys.CheckKeyLogin(ctx, serverID); err != nil {
		return apperr.Wrap(apperr.SecKeyLoginFailed, "the key could not log in before the change", err)
	}
	asRoot, withSudo := PlanPassword(true), PlanPassword(false)
	res, err := s.runPlan(ctx, serverID, asRoot.Disable, withSudo.Disable, command)
	if err != nil {
		return err
	}
	if res.Exit != 0 {
		return passwordFailure(res)
	}
	if err := s.keyAfterReload(ctx, serverID); err != nil {
		// Undo it over the open session: same sudo form as the change.
		undo := asRoot.Enable
		if command == withSudo.Disable.Command {
			undo = withSudo.Enable
		}
		if _, uerr := s.runPlan(ctx, serverID, asRoot.Enable, withSudo.Enable, undo.Command); uerr != nil {
			return apperr.Wrap(apperr.SecKeyLoginLost, "the key could not log in after the change, and undoing it failed: remove "+dropIn+" and reload sshd", err)
		}
		return apperr.Wrap(apperr.SecKeyLoginLost, "the key could not log in after the change, so it was undone", err)
	}
	return nil
}

// keyAfterReload checks the key, trying again for a few seconds: sshd is
// restarting just after a reload.
func (s *Service) keyAfterReload(ctx context.Context, serverID string) error {
	var err error
	for i := 0; i < afterReloadTries; i++ {
		if i > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(afterReloadWait):
			}
		}
		if err = s.keys.CheckKeyLogin(ctx, serverID); err == nil {
			return nil
		}
	}
	return err
}

// EnablePassword undoes DisablePassword: removes the drop-in and reloads
// sshd. command must be what PlanPassword gave.
func (s *Service) EnablePassword(ctx context.Context, serverID, command string) error {
	asRoot, withSudo := PlanPassword(true), PlanPassword(false)
	res, err := s.runPlan(ctx, serverID, asRoot.Enable, withSudo.Enable, command)
	if err != nil {
		return err
	}
	if res.Exit != 0 {
		return passwordFailure(res)
	}
	return nil
}

// passwordFailure turns what a failed script printed into an error.
func passwordFailure(res audit.Result) error {
	out := strings.TrimSpace(string(res.Stderr) + "\n" + string(res.Stdout))
	switch {
	case strings.Contains(out, "password is required") || strings.Contains(out, "need to be root") ||
		strings.Contains(out, "ermission denied"):
		return apperr.New(apperr.CmdNeedsRoot, "sudo needs a password on this server")
	case strings.Contains(out, "DAVISH_NO_INCLUDE"):
		return apperr.New(apperr.CmdFailed, "sshd does not read /etc/ssh/sshd_config.d here, so nothing was changed")
	case strings.Contains(out, "DAVISH_BACKUP"):
		return apperr.New(apperr.CmdFailed, "the backup of the sshd settings failed, so nothing was changed")
	case strings.Contains(out, "DAVISH_SSHD_T"):
		return apperr.New(apperr.CmdFailed, "sshd rejected the new setting, so nothing was changed")
	case strings.Contains(out, "DAVISH_OVERRIDDEN"):
		return apperr.New(apperr.CmdFailed, "another sshd setting keeps password logins on, so the change was undone")
	case strings.Contains(out, "DAVISH_RELOAD"):
		return apperr.New(apperr.CmdFailed, "sshd could not be reloaded, so the change was undone")
	}
	if res.Exit == 127 || strings.Contains(out, "not found") {
		return apperr.New(apperr.CmdMissing, "sshd or systemctl is not on this server")
	}
	return apperr.New(apperr.CmdFailed, lastLine(out))
}
