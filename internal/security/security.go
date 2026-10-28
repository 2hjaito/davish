// Package security reads who logs in to a server over SSH and who tries:
// accepted logins, failed attempts grouped by address, and the state of
// ufw and fail2ban. It only reads; every change is a sudo command the user
// runs in a terminal.
package security

import (
	"bufio"
	"bytes"
	"cmp"
	"context"
	"net"
	"regexp"
	"slices"
	"strings"

	"davish/internal/audit"
)

// app is the name commands are recorded under in the audit log.
const app = "security"

// Runner runs a command on a server and records it in the audit log.
type Runner interface {
	RunLine(ctx context.Context, serverID, app, line string) (audit.Result, error)
}

// Service reads login security.
type Service struct {
	run Runner
}

// New creates the service.
func New(run Runner) *Service {
	return &Service{run: run}
}

// Login is one accepted SSH login.
type Login struct {
	Time   string `json:"time"`
	User   string `json:"user"`
	IP     string `json:"ip"`
	Method string `json:"method"`
}

// Attempts is the failed logins from one address.
type Attempts struct {
	IP    string   `json:"ip"`
	Count int      `json:"count"`
	Users []string `json:"users"` // the names tried, most tried first, at most 5
	Last  string   `json:"last"`
}

// Firewall is what ufw or fail2ban report, when installed.
type Firewall struct {
	Installed bool `json:"installed"`
	// NeedsRoot is set when the status could not be read without root.
	NeedsRoot bool     `json:"needsRoot"`
	Active    bool     `json:"active"`
	Lines     []string `json:"lines"`
	// Banned is fail2ban's banned addresses in the sshd jail.
	Banned []string `json:"banned,omitempty"`
}

// Report is everything the Security app shows.
type Report struct {
	Root bool `json:"root"`
	// Source is where logins were read: "journal", "auth.log" or "" when
	// neither was readable.
	Source      string     `json:"source"`
	Accepted    []Login    `json:"accepted"`
	Failed      []Attempts `json:"failed"`
	FailedTotal int        `json:"failedTotal"`
	// Last is `last` (sessions) and Lastb `lastb` (bad logins, root only).
	Last        []string `json:"last"`
	Lastb       []string `json:"lastb"`
	LastbDenied bool     `json:"lastbDenied"`
	UFW         Firewall `json:"ufw"`
	Fail2ban    Firewall `json:"fail2ban"`
	// PasswordAuth is sshd's PasswordAuthentication: "yes", "no" or ""
	// when it could not be read.
	PasswordAuth string `json:"passwordAuth"`
	// DisablePassword is the sudo command that turns password logins off.
	DisablePassword string `json:"disablePassword"`
}

const (
	maxLogLines = 5000
	maxAccepted = 30
	maxFailed   = 50
)

// reportLine gathers everything in one command, in sections. The journal
// is read for the last 7 days; auth.log only when the journal had nothing.
const reportLine = `id -u` +
	`; echo --journal--; journalctl -u ssh -u sshd --since '-7 days' --no-pager -o short-iso 2>/dev/null | grep -E 'Accepted |Failed password|Invalid user' | tail -n 5000` +
	`; echo --authlog--; [ -r /var/log/auth.log ] && grep -hE 'sshd.*(Accepted |Failed password|Invalid user)' /var/log/auth.log | tail -n 5000` +
	`; echo --last--; last -n 20 -i -w 2>/dev/null | grep -v '^$' | grep -v '^wtmp begins'` +
	`; echo --lastb--; lastb -n 20 -i -w 2>&1 | grep -v '^$' | grep -v '^btmp begins'` +
	`; echo --ufw--; command -v ufw >/dev/null && { echo INSTALLED; ufw status 2>&1 | head -n 60; }` +
	`; echo --f2b--; command -v fail2ban-client >/dev/null && { echo INSTALLED; fail2ban-client status sshd 2>&1 | head -n 30; }` +
	`; echo --sshd--; { sshd -T 2>/dev/null || cat /etc/ssh/sshd_config.d/*.conf /etc/ssh/sshd_config 2>/dev/null; } | grep -iE '^[[:space:]]*passwordauthentication[[:space:]]' | head -n 1` +
	`; true`

// Report reads the server's login security.
func (s *Service) Report(ctx context.Context, serverID string) (Report, error) {
	res, err := s.run.RunLine(ctx, serverID, app, reportLine)
	if err != nil {
		return Report{}, err
	}
	r := parse(res.Stdout)
	r.DisablePassword = DisablePasswordCommand
	return r, nil
}

func sections(out []byte) map[string][]string {
	m := map[string][]string{}
	cur := "uid"
	sc := bufio.NewScanner(bytes.NewReader(out))
	sc.Buffer(make([]byte, 64<<10), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		if t := strings.TrimSpace(line); len(t) > 4 && strings.HasPrefix(t, "--") && strings.HasSuffix(t, "--") && !strings.Contains(t, " ") {
			cur = strings.Trim(t, "-")
			continue
		}
		if strings.TrimSpace(line) != "" {
			m[cur] = append(m[cur], line)
		}
	}
	return m
}

func parse(out []byte) Report {
	sec := sections(out)
	r := Report{Root: first(sec["uid"]) == "0"}

	logs := sec["journal"]
	r.Source = "journal"
	if len(logs) == 0 {
		logs, r.Source = sec["authlog"], "auth.log"
	}
	if len(logs) == 0 {
		r.Source = ""
	}
	r.Accepted, r.Failed, r.FailedTotal = parseAuth(logs)

	r.Last = sec["last"]
	for _, l := range sec["lastb"] {
		if strings.Contains(l, "Permission denied") {
			r.LastbDenied = true
			r.Lastb = nil
			break
		}
		r.Lastb = append(r.Lastb, l)
	}
	r.UFW = parseUFW(sec["ufw"])
	r.Fail2ban = parseF2B(sec["f2b"])
	if f := strings.Fields(first(sec["sshd"])); len(f) >= 2 {
		r.PasswordAuth = strings.ToLower(f[1])
	}
	return r
}

func first(l []string) string {
	if len(l) == 0 {
		return ""
	}
	return strings.TrimSpace(l[0])
}

var (
	acceptedRe = regexp.MustCompile(`Accepted (\S+) for (\S+) from (\S+) port`)
	failedRe   = regexp.MustCompile(`Failed password for (?:invalid user )?(\S+) from (\S+) port (\d+)`)
	invalidRe  = regexp.MustCompile(`Invalid user (\S*) from (\S+) port (\d+)`)
)

// stamp is the time a log line starts with: one ISO field from the
// journal, or "Oct 28 10:00:00" from syslog.
func stamp(line string) string {
	f := strings.Fields(line)
	if len(f) == 0 {
		return ""
	}
	if strings.Contains(f[0], "T") || len(f) < 3 {
		return f[0]
	}
	return strings.Join(f[:3], " ")
}

func parseAuth(lines []string) ([]Login, []Attempts, int) {
	if len(lines) > maxLogLines {
		lines = lines[len(lines)-maxLogLines:]
	}
	var accepted []Login
	byIP := map[string]*Attempts{}
	users := map[string]map[string]int{}
	total := 0
	// A try with a password logs "Invalid user" and then "Failed password"
	// from the same port; count it once.
	tried := map[string]bool{}
	for _, l := range lines {
		if m := failedRe.FindStringSubmatch(l); m != nil {
			tried[m[2]+" "+m[3]] = true
		}
	}
	fail := func(user, ip, at string) {
		// The user name is the client's to choose and may shift the fields;
		// only a real address is kept, since it ends up in a ufw command.
		if net.ParseIP(ip) == nil {
			return
		}
		a := byIP[ip]
		if a == nil {
			a = &Attempts{IP: ip}
			byIP[ip] = a
			users[ip] = map[string]int{}
		}
		a.Count++
		a.Last = at
		users[ip][user]++
		total++
	}
	for _, l := range lines {
		at := stamp(l)
		switch {
		case acceptedRe.MatchString(l):
			m := acceptedRe.FindStringSubmatch(l)
			if net.ParseIP(m[3]) == nil {
				continue
			}
			accepted = append(accepted, Login{Time: at, Method: m[1], User: m[2], IP: m[3]})
		case failedRe.MatchString(l):
			m := failedRe.FindStringSubmatch(l)
			fail(m[1], m[2], at)
		case invalidRe.MatchString(l):
			if m := invalidRe.FindStringSubmatch(l); !tried[m[2]+" "+m[3]] {
				fail(m[1], m[2], at)
			}
		}
	}
	slices.Reverse(accepted)
	if len(accepted) > maxAccepted {
		accepted = accepted[:maxAccepted]
	}
	failed := make([]Attempts, 0, len(byIP))
	for ip, a := range byIP {
		names := make([]string, 0, len(users[ip]))
		for u := range users[ip] {
			names = append(names, u)
		}
		slices.SortFunc(names, func(x, y string) int {
			return cmp.Or(cmp.Compare(users[ip][y], users[ip][x]), strings.Compare(x, y))
		})
		a.Users = names[:min(5, len(names))]
		failed = append(failed, *a)
	}
	slices.SortFunc(failed, func(x, y Attempts) int {
		return cmp.Or(cmp.Compare(y.Count, x.Count), strings.Compare(x.IP, y.IP))
	})
	if len(failed) > maxFailed {
		failed = failed[:maxFailed]
	}
	return accepted, failed, total
}

func parseUFW(lines []string) Firewall {
	var f Firewall
	if first(lines) != "INSTALLED" {
		return f
	}
	f.Installed = true
	for _, l := range lines[1:] {
		switch {
		case strings.Contains(l, "need to be root"):
			f.NeedsRoot = true
		case strings.HasPrefix(l, "Status:"):
			f.Active = strings.Contains(l, "active") && !strings.Contains(l, "inactive")
		default:
			f.Lines = append(f.Lines, l)
		}
	}
	return f
}

func parseF2B(lines []string) Firewall {
	var f Firewall
	if first(lines) != "INSTALLED" {
		return f
	}
	f.Installed = true
	for _, l := range lines[1:] {
		t := strings.TrimLeft(l, " |`-")
		switch {
		case strings.Contains(l, "ermission denied") || strings.Contains(l, "need to be root"):
			f.NeedsRoot = true
		case strings.HasPrefix(t, "Banned IP list:"):
			f.Banned = strings.Fields(strings.TrimPrefix(t, "Banned IP list:"))
			f.Active = true
		case strings.HasPrefix(t, "Status for the jail"):
			f.Active = true
		default:
			f.Lines = append(f.Lines, strings.TrimSpace(l))
		}
	}
	return f
}

// DisablePasswordCommand is what turns off password logins: a drop-in
// that sshd checks with -t before reloading, removed again if the check
// fails. The user runs it with sudo in a terminal.
const DisablePasswordCommand = `sudo sh -c 'f=/etc/ssh/sshd_config.d/60-davish-no-password.conf;` +
	` printf "PasswordAuthentication no\nKbdInteractiveAuthentication no\n" > "$f"` +
	` && sshd -t && { systemctl reload ssh 2>/dev/null || systemctl reload sshd; }` +
	` || { rm -f "$f"; echo "sshd -t failed: change reverted"; }'`
