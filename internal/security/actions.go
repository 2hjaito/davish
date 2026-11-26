package security

import (
	"context"
	"net/netip"
	"strings"

	"davish/internal/apperr"
	"davish/internal/audit"
	"davish/internal/shellx"
)

// What the Security app can do to one address (ADR 0016).
const (
	Block   = "block"   // ufw: deny every packet from it
	Unblock = "unblock" // ufw: remove that deny
	Ban     = "ban"     // fail2ban: ban it in the sshd jail
	Unban   = "unban"   // fail2ban: lift that ban
)

// Plan is the command of one action, in the two forms the user can pick.
type Plan struct {
	// Command is what Davish runs once confirmed: as root, or through
	// sudo -n, which fails rather than ask for a password.
	Command string `json:"command"`
	// Terminal is what the terminal gets instead: plain sudo, so the
	// user types the password there.
	Terminal string `json:"terminal"`
}

// address accepts an IP or a CIDR range and gives it in canonical form.
func address(s string) (string, error) {
	s = strings.TrimSpace(s)
	if a, err := netip.ParseAddr(s); err == nil && !a.Is4In6() && a.Zone() == "" {
		return a.String(), nil
	}
	if p, err := netip.ParsePrefix(s); err == nil {
		return p.Masked().String(), nil
	}
	return "", apperr.New(apperr.InvalidInput, "not an IP address or range")
}

// Narrowest range a block or ban may name: wider ones cut off whole
// networks, and 0.0.0.0/0 cuts off every login, SSH included.
const (
	minBits4 = 8
	minBits6 = 32
)

// checkBlockable refuses an address or range that would lock out more than
// the one host it is meant for: the unspecified and loopback addresses and
// ranges wider than minBits4 / minBits6.
func checkBlockable(addr string) error {
	var p netip.Prefix
	if pp, err := netip.ParsePrefix(addr); err == nil {
		p = pp
	} else if a, err := netip.ParseAddr(addr); err == nil {
		p = netip.PrefixFrom(a, a.BitLen())
	} else {
		return apperr.New(apperr.InvalidInput, "not an IP address or range")
	}
	a := p.Addr()
	floor := minBits4
	if a.Is6() {
		floor = minBits6
	}
	if a.IsUnspecified() || a.IsLoopback() || p.Bits() < floor {
		return apperr.New(apperr.SecWouldLockOut, "this address or range would lock out more than it should")
	}
	return nil
}

// covers tells whether the address or range target holds client.
func covers(target, client string) bool {
	c, err := netip.ParseAddr(client)
	if err != nil {
		return false
	}
	if p, err := netip.ParsePrefix(target); err == nil {
		return p.Contains(c)
	}
	a, err := netip.ParseAddr(target)
	return err == nil && a == c
}

// clientLine prints the address sshd saw this connection come from.
const clientLine = `printf '%s' "${SSH_CLIENT%% *}"`

// script is the action's shell line, without sudo. A ufw insert at 1
// fails while there is no rule yet, so the deny is then appended.
func script(action, ip string) (string, error) {
	q := shellx.Quote(ip)
	switch action {
	case Block:
		return "ufw delete deny from " + q + "; ufw insert 1 deny from " + q + " || ufw deny from " + q, nil
	case Unblock:
		return "ufw delete deny from " + q, nil
	case Ban:
		return "fail2ban-client set sshd banip " + q, nil
	case Unban:
		return "fail2ban-client set sshd unbanip " + q, nil
	}
	return "", apperr.New(apperr.InvalidInput, "unknown action")
}

// sudoForm runs line as root through prefix: one command directly, several
// under sh -c so every part of it gets root.
func sudoForm(prefix, line string) string {
	if strings.ContainsAny(line, ";|&") {
		return prefix + "sh -c " + shellx.Quote(line)
	}
	return prefix + line
}

// PlanAction gives the command of action on ip. root is whether the
// login is already root, which needs no sudo at all.
func PlanAction(action, ip string, root bool) (Plan, error) {
	addr, err := address(ip)
	if err != nil {
		return Plan{}, err
	}
	if action == Block || action == Ban {
		if err := checkBlockable(addr); err != nil {
			return Plan{}, err
		}
	}
	line, err := script(action, addr)
	if err != nil {
		return Plan{}, err
	}
	p := Plan{Command: sudoForm("sudo -n ", line), Terminal: sudoForm("sudo ", line)}
	if root {
		p.Command = line
	}
	return p, nil
}

// Apply runs command, which must be what PlanAction gave for action on
// ip. sudo asking for a password is CMD_NEEDS_ROOT: the terminal form
// is then the way.
func (s *Service) Apply(ctx context.Context, serverID, action, ip, command string) error {
	asRoot, err := PlanAction(action, ip, true)
	if err != nil {
		return err
	}
	withSudo, _ := PlanAction(action, ip, false)
	if command != asRoot.Command && command != withSudo.Command {
		return apperr.New(apperr.FSStale, "the command changed since it was confirmed")
	}
	if action == Block || action == Ban {
		// Never cut off the address Davish itself comes from: a typo in a
		// login or a tick on "block all" could otherwise lock it out.
		if cr, err := s.run.RunLine(ctx, serverID, app, clientLine); err == nil && cr.Exit == 0 {
			if addr, aerr := address(ip); aerr == nil && covers(addr, strings.TrimSpace(string(cr.Stdout))) {
				return apperr.New(apperr.SecWouldLockOut, "this is the address Davish connects from")
			}
		}
	}
	var res audit.Result
	if command == withSudo.Command && s.root != nil {
		// Through sudo.Keeper: without a password when sudo allows, else
		// with the session's (SUDO_PASSWORD_REQUIRED asks for it). The
		// line is the same, only how it reaches root differs.
		res, err = s.root.Run(ctx, serverID, app, asRoot.Command, nil)
	} else {
		res, err = s.run.RunLine(ctx, serverID, app, command)
	}
	if err != nil {
		return err
	}
	if res.Exit == 0 {
		return nil
	}
	out := strings.TrimSpace(string(res.Stderr) + "\n" + string(res.Stdout))
	if strings.Contains(out, "password is required") || strings.Contains(out, "need to be root") ||
		strings.Contains(out, "ermission denied") {
		return apperr.New(apperr.CmdNeedsRoot, "sudo needs a password on this server")
	}
	if res.Exit == 127 || strings.Contains(out, "not found") {
		return apperr.New(apperr.CmdMissing, "ufw or fail2ban-client is not installed")
	}
	return apperr.New(apperr.CmdFailed, lastLine(out))
}

// lastLine is the end of what the command printed, where its error is.
func lastLine(out string) string {
	lines := strings.Split(out, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if l := strings.TrimSpace(lines[i]); l != "" {
			return l
		}
	}
	return "the command failed"
}
