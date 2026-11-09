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
	"time"

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

// HourCount is the failed logins in one hour, server time.
type HourCount struct {
	Hour  string `json:"hour"` // "2006-01-02T15"
	Count int    `json:"count"`
}

// UserCount is how often one user name was tried.
type UserCount struct {
	User  string `json:"user"`
	Count int    `json:"count"`
}

// Session is one open login session, as `who` lists it.
type Session struct {
	User  string `json:"user"`
	TTY   string `json:"tty"`
	Since string `json:"since"`
	From  string `json:"from"`
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
	// InvalidTotal is the failed logins for a user that does not exist.
	InvalidTotal int `json:"invalidTotal"`
	// FailedIPs is how many addresses failed, before Failed is cut.
	FailedIPs     int `json:"failedIPs"`
	AcceptedTotal int `json:"acceptedTotal"`
	// Targets is the user names tried most, from every address.
	Targets []UserCount `json:"targets"`
	// Hourly is the failed logins per hour, gaps filled, up to now.
	Hourly   []HourCount `json:"hourly"`
	Sessions []Session   `json:"sessions"`
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
	maxTargets  = 10
	maxHours    = 7 * 24
)

// reportLine gathers everything in one command, in sections. The journal
// is read for the last 7 days; auth.log only when the journal had nothing.
const reportLine = `id -u` +
	`; echo --now--; date +%Y-%m-%dT%H:%M:%S%z` +
	`; echo --journal--; journalctl -u ssh -u sshd --since '-7 days' --no-pager -o short-iso 2>/dev/null | grep -E 'Accepted |Failed password|Invalid user' | tail -n 5000` +
	`; echo --authlog--; [ -r /var/log/auth.log ] && grep -hE 'sshd.*(Accepted |Failed password|Invalid user)' /var/log/auth.log | tail -n 5000` +
	`; echo --last--; last -n 20 -i -w 2>/dev/null | grep -v '^$' | grep -v '^wtmp begins'` +
	`; echo --who--; who 2>/dev/null` +
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
	a := parseAuth(logs)
	r.Accepted, r.Failed, r.FailedTotal = a.accepted, a.failed, a.total
	r.InvalidTotal, r.FailedIPs, r.AcceptedTotal, r.Targets = a.invalid, a.ips, a.acceptedTotal, a.targets
	now, ok := parseTime(first(sec["now"]))
	if !ok {
		now = time.Now()
	}
	r.Hourly = hourly(a.failedAt, now)
	r.Sessions = parseWho(sec["who"])

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

// auth is what the SSH log lines add up to.
type auth struct {
	accepted      []Login
	acceptedTotal int
	failed        []Attempts
	total         int
	invalid       int
	ips           int
	targets       []UserCount
	failedAt      []string // the stamp of every failed login, oldest first
}

func parseAuth(lines []string) auth {
	if len(lines) > maxLogLines {
		lines = lines[len(lines)-maxLogLines:]
	}
	var accepted []Login
	byIP := map[string]*Attempts{}
	users := map[string]map[string]int{}
	var res auth
	tries := map[string]int{}
	// A try with a password logs "Invalid user" and then "Failed password"
	// from the same port; count it once.
	tried := map[string]bool{}
	for _, l := range lines {
		if m := failedRe.FindStringSubmatch(l); m != nil {
			tried[m[2]+" "+m[3]] = true
		}
	}
	fail := func(user, ip, at string, invalid bool) {
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
		tries[user]++
		res.total++
		if invalid {
			res.invalid++
		}
		res.failedAt = append(res.failedAt, at)
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
			fail(m[1], m[2], at, strings.Contains(l, "for invalid user "))
		case invalidRe.MatchString(l):
			if m := invalidRe.FindStringSubmatch(l); !tried[m[2]+" "+m[3]] {
				fail(m[1], m[2], at, true)
			}
		}
	}
	slices.Reverse(accepted)
	res.acceptedTotal = len(accepted)
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
	res.accepted, res.failed, res.ips = accepted, failed, len(byIP)
	res.targets = make([]UserCount, 0, len(tries))
	for u, n := range tries {
		res.targets = append(res.targets, UserCount{User: u, Count: n})
	}
	slices.SortFunc(res.targets, func(x, y UserCount) int {
		return cmp.Or(cmp.Compare(y.Count, x.Count), strings.Compare(x.User, y.User))
	})
	if len(res.targets) > maxTargets {
		res.targets = res.targets[:maxTargets]
	}
	return res
}

// parseTime reads an ISO stamp, from the journal, rsyslog or `date`.
func parseTime(s string) (time.Time, bool) {
	for _, layout := range []string{"2006-01-02T15:04:05-0700", time.RFC3339Nano} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// parseSyslogTime reads "Oct 28 10:00:00", which has no year or zone: it
// is taken in the server's zone, in the latest year that does not put it
// in the future.
func parseSyslogTime(s string, now time.Time) (time.Time, bool) {
	t, err := time.ParseInLocation("Jan 2 15:04:05", s, now.Location())
	if err != nil {
		return time.Time{}, false
	}
	t = t.AddDate(now.Year(), 0, 0)
	if t.After(now.Add(24 * time.Hour)) {
		t = t.AddDate(-1, 0, 0)
	}
	return t, true
}

// hourly counts stamps per hour of the server's clock, from the first
// hour seen to now, at most maxHours of them.
func hourly(stamps []string, now time.Time) []HourCount {
	loc := now.Location()
	hour := func(t time.Time) time.Time {
		t = t.In(loc)
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, loc)
	}
	counts := map[time.Time]int{}
	var first, last time.Time
	for _, s := range stamps {
		t, ok := parseTime(s)
		if !ok {
			t, ok = parseSyslogTime(s, now)
		}
		if !ok {
			continue
		}
		h := hour(t)
		counts[h]++
		if first.IsZero() || h.Before(first) {
			first = h
		}
		if h.After(last) {
			last = h
		}
	}
	if len(counts) == 0 {
		return nil
	}
	end := last
	if n := hour(now); n.After(end) {
		end = n
	}
	if limit := end.Add(-(maxHours - 1) * time.Hour); first.Before(limit) {
		first = limit
	}
	var out []HourCount
	for h := first; !h.After(end); h = h.Add(time.Hour) {
		out = append(out, HourCount{Hour: h.Format("2006-01-02T15"), Count: counts[h]})
	}
	return out
}

// parseWho reads `who`: user, tty, the login time in one or more fields,
// and the remote host in parentheses when there is one.
func parseWho(lines []string) []Session {
	var out []Session
	for _, l := range lines {
		f := strings.Fields(l)
		if len(f) < 3 {
			continue
		}
		s := Session{User: f[0], TTY: f[1]}
		rest := f[2:]
		if n := len(rest); strings.HasPrefix(rest[n-1], "(") && strings.HasSuffix(rest[n-1], ")") {
			s.From = strings.Trim(rest[n-1], "()")
			rest = rest[:n-1]
		}
		s.Since = strings.Join(rest, " ")
		out = append(out, s)
	}
	return out
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
