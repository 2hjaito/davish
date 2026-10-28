package security

import (
	"strings"
	"testing"
)

const sample = `1000
--journal--
2026-10-28T10:00:00+0000 vps sshd[1]: Invalid user admin from 203.0.113.9 port 4000
2026-10-28T10:00:01+0000 vps sshd[1]: Failed password for invalid user admin from 203.0.113.9 port 4000 ssh2
2026-10-28T10:00:02+0000 vps sshd[2]: Failed password for root from 203.0.113.9 port 4001 ssh2
2026-10-28T10:00:03+0000 vps sshd[3]: Failed password for root from 198.51.100.7 port 5000 ssh2
2026-10-28T10:00:04+0000 vps sshd[4]: Invalid user oracle from 198.51.100.8 port 5001
2026-10-28T10:00:05+0000 vps sshd[5]: Failed password for root from 203.0.113.9 port 4002 ssh2
2026-10-28T11:00:00+0000 vps sshd[6]: Accepted publickey for dangth from 192.0.2.1 port 6000 ssh2: ED25519 SHA256:x
2026-10-28T12:00:00+0000 vps sshd[7]: Accepted password for dangth from 192.0.2.2 port 6001 ssh2
--authlog--
--last--
dangth   pts/0        192.0.2.1        Wed Oct 28 11:00   still logged in
--lastb--
lastb: cannot open /var/log/btmp: Permission denied
--ufw--
INSTALLED
ERROR: You need to be root to run this script
--f2b--
INSTALLED
Status for the jail: sshd
|- Filter
|  |- Currently failed:	1
` + "`- Actions\n   |- Currently banned:\t1\n   `- Banned IP list:\t203.0.113.9\n" + `--sshd--
PasswordAuthentication yes
`

func TestParse(t *testing.T) {
	r := parse([]byte(sample))
	if r.Root || r.Source != "journal" {
		t.Errorf("root/source = %v %q", r.Root, r.Source)
	}
	if len(r.Accepted) != 2 || r.Accepted[0].Method != "password" || r.Accepted[0].IP != "192.0.2.2" {
		t.Errorf("accepted = %+v", r.Accepted)
	}
	if r.FailedTotal != 5 || len(r.Failed) != 3 {
		t.Fatalf("failed = %d %+v", r.FailedTotal, r.Failed)
	}
	top := r.Failed[0]
	if top.IP != "203.0.113.9" || top.Count != 3 || strings.Join(top.Users, ",") != "root,admin" {
		t.Errorf("top = %+v", top)
	}
	if top.Last != "2026-10-28T10:00:05+0000" {
		t.Errorf("last = %q", top.Last)
	}
	if !r.LastbDenied || len(r.Last) != 1 {
		t.Errorf("last/lastb = %v %v", r.Last, r.LastbDenied)
	}
	if !r.UFW.Installed || !r.UFW.NeedsRoot {
		t.Errorf("ufw = %+v", r.UFW)
	}
	if !r.Fail2ban.Active || len(r.Fail2ban.Banned) != 1 || r.Fail2ban.Banned[0] != "203.0.113.9" {
		t.Errorf("fail2ban = %+v", r.Fail2ban)
	}
	if r.PasswordAuth != "yes" {
		t.Errorf("password auth = %q", r.PasswordAuth)
	}
}

func TestForgedUserCannotInjectAnAddress(t *testing.T) {
	line := "2026-10-28T10:00:00+0000 vps sshd[1]: Invalid user a from ;reboot port 1 x from 203.0.113.9 port 4000\n"
	r := parse([]byte("1000\n--journal--\n" + line))
	for _, a := range r.Failed {
		if a.IP != "203.0.113.9" && a.IP != "" {
			t.Errorf("kept address %q", a.IP)
		}
	}
}

func TestParseAuthLogFallback(t *testing.T) {
	r := parse([]byte("0\n--journal--\n--authlog--\nOct 28 10:00:00 vps sshd[1]: Failed password for root from 203.0.113.9 port 1 ssh2\n"))
	if !r.Root || r.Source != "auth.log" || r.Failed[0].Last != "Oct 28 10:00:00" {
		t.Errorf("report = %+v", r)
	}
	if parse([]byte("0\n")).Source != "" {
		t.Error("source set without logs")
	}
}
