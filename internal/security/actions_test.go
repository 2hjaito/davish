package security

import (
	"context"
	"errors"
	"testing"

	"davish/internal/apperr"
	"davish/internal/audit"
)

func TestPlanAction(t *testing.T) {
	cases := []struct {
		action, ip string
		root       bool
		command    string
		terminal   string
	}{
		{Block, "203.0.113.9", false,
			`sudo -n sh -c 'ufw delete deny from 203.0.113.9; ufw insert 1 deny from 203.0.113.9 || ufw deny from 203.0.113.9'`,
			`sudo sh -c 'ufw delete deny from 203.0.113.9; ufw insert 1 deny from 203.0.113.9 || ufw deny from 203.0.113.9'`},
		{Block, "203.0.113.9", true,
			`ufw delete deny from 203.0.113.9; ufw insert 1 deny from 203.0.113.9 || ufw deny from 203.0.113.9`,
			`sudo sh -c 'ufw delete deny from 203.0.113.9; ufw insert 1 deny from 203.0.113.9 || ufw deny from 203.0.113.9'`},
		{Unblock, " 2001:DB8::1 ", false, `sudo -n ufw delete deny from 2001:db8::1`, `sudo ufw delete deny from 2001:db8::1`},
		{Unblock, "198.51.100.7/24", false, `sudo -n ufw delete deny from 198.51.100.0/24`, `sudo ufw delete deny from 198.51.100.0/24`},
		{Ban, "198.51.100.7", false, `sudo -n fail2ban-client set sshd banip 198.51.100.7`, `sudo fail2ban-client set sshd banip 198.51.100.7`},
		{Unban, "198.51.100.7", true, `fail2ban-client set sshd unbanip 198.51.100.7`, `sudo fail2ban-client set sshd unbanip 198.51.100.7`},
	}
	for _, c := range cases {
		p, err := PlanAction(c.action, c.ip, c.root)
		if err != nil {
			t.Fatalf("%s %s: %v", c.action, c.ip, err)
		}
		if p.Command != c.command || p.Terminal != c.terminal {
			t.Errorf("%s %s root=%v:\n got %q\n     %q", c.action, c.ip, c.root, p.Command, p.Terminal)
		}
	}
}

func TestPlanActionRejects(t *testing.T) {
	for _, ip := range []string{"", "1.2.3.4; reboot", "$(id)", "example.com", "fe80::1%eth0", "::ffff:1.2.3.4"} {
		if _, err := PlanAction(Block, ip, false); !isCode(err, apperr.InvalidInput) {
			t.Errorf("%q: err = %v", ip, err)
		}
	}
	if _, err := PlanAction("reboot", "1.2.3.4", false); !isCode(err, apperr.InvalidInput) {
		t.Errorf("unknown action: err = %v", err)
	}
}

type fakeRun struct {
	res  audit.Result
	line string
}

func (f *fakeRun) RunLine(_ context.Context, _, _, line string) (audit.Result, error) {
	f.line = line
	return f.res, nil
}

func TestApply(t *testing.T) {
	p, _ := PlanAction(Ban, "203.0.113.9", false)
	run := &fakeRun{res: audit.Result{Stdout: []byte("1\n")}}
	s := New(run)
	if err := s.Apply(context.Background(), "s", Ban, "203.0.113.9", p.Command); err != nil || run.line != p.Command {
		t.Fatalf("err = %v, ran %q", err, run.line)
	}

	// Only a command PlanAction gave runs, never the terminal form or another.
	run.line = ""
	for _, cmd := range []string{p.Terminal, "reboot", p.Command + "; reboot"} {
		if err := s.Apply(context.Background(), "s", Ban, "203.0.113.9", cmd); !isCode(err, apperr.FSStale) || run.line != "" {
			t.Errorf("%q: err = %v, ran %q", cmd, err, run.line)
		}
	}

	cases := []struct {
		res  audit.Result
		code apperr.Code
	}{
		{audit.Result{Exit: 1, Stderr: []byte("sudo: a password is required\n")}, apperr.CmdNeedsRoot},
		{audit.Result{Exit: 127, Stderr: []byte("sh: 1: fail2ban-client: not found\n")}, apperr.CmdMissing},
		{audit.Result{Exit: 255, Stdout: []byte("ERROR   Failed to access socket path\n")}, apperr.CmdFailed},
	}
	for _, c := range cases {
		run.res = c.res
		if err := s.Apply(context.Background(), "s", Ban, "203.0.113.9", p.Command); !isCode(err, c.code) {
			t.Errorf("%s: err = %v", c.res.Stderr, err)
		}
	}
}

func isCode(err error, code apperr.Code) bool {
	var e *apperr.Error
	return errors.As(err, &e) && e.Code == code
}
