package cron

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata" // the server's zone, whatever the user's OS ships

	"github.com/pkg/sftp"

	"davish/internal/apperr"
	"davish/internal/audit"
	"davish/internal/shellx"
)

// app is the name commands are recorded under in the audit log.
const app = "cron"

// SourceUser is the user's own crontab, the one Davish can edit.
const SourceUser = "user"

// Runner runs a command on a server and records it in the audit log.
type Runner interface {
	Run(ctx context.Context, serverID, app string, cmd shellx.Command) (audit.Result, error)
	RunInput(ctx context.Context, serverID, app, line string, stdin []byte) (audit.Result, error)
}

// Conns hands out a server's shared SFTP client, to read system crontabs.
type Conns interface {
	SFTP(serverID string) (*sftp.Client, error)
}

// Job is one scheduled command.
type Job struct {
	// ID names the line and its content, so an edit of a line that
	// changed meanwhile is refused.
	ID       string   `json:"id"`
	Source   string   `json:"source"` // "user", "/etc/crontab", "/etc/cron.d/x"
	Line     int      `json:"line"`
	Schedule Schedule `json:"schedule"`
	User     string   `json:"user,omitempty"`
	Command  string   `json:"command"`
	Enabled  bool     `json:"enabled"`
	// Next runs as unix seconds, up to three; empty for @reboot.
	Next []int64 `json:"next"`
	// Editable is true for the user's crontab; system files need root.
	Editable bool `json:"editable"`
}

// Env is a variable set in a crontab, such as MAILTO.
type Env struct {
	Source string `json:"source"`
	Name   string `json:"name"`
	Value  string `json:"value"`
}

// Table is every job Davish can see on a server.
type Table struct {
	Jobs []Job `json:"jobs"`
	Env  []Env `json:"env"`
	// Zone is the server's time zone, in which Next is computed.
	Zone string `json:"zone"`
	// Unreadable lists system crontabs the user may not read.
	Unreadable []string `json:"unreadable"`
	// Version identifies the user's crontab as read, for Apply.
	Version string `json:"version"`
}

// Service reads and edits crontabs on each server.
type Service struct {
	run   Runner
	conns Conns
	now   func() time.Time
}

// New creates the service.
func New(run Runner, conns Conns) *Service {
	return &Service{run: run, conns: conns, now: time.Now}
}

// List returns the user's jobs, then those of /etc/crontab and
// /etc/cron.d, with their next runs in the server's zone.
func (s *Service) List(ctx context.Context, serverID string) (Table, error) {
	text, _, err := s.userCrontab(ctx, serverID)
	if err != nil {
		return Table{}, err
	}
	sc, err := s.conns.SFTP(serverID)
	if err != nil {
		return Table{}, err
	}
	loc := serverZone(sc)
	t := Table{Jobs: []Job{}, Env: []Env{}, Unreadable: []string{}, Zone: loc.String(), Version: version(text)}
	now := s.now().In(loc)
	t.add(SourceUser, text, false, now)
	for _, p := range systemFiles(sc) {
		data, err := readFile(sc, p)
		if err != nil {
			t.Unreadable = append(t.Unreadable, p)
			continue
		}
		t.add(p, data, true, now)
	}
	return t, nil
}

func (t *Table) add(source, text string, system bool, now time.Time) {
	for _, e := range ParseCrontab(text, system) {
		switch e.Kind {
		case KindEnv:
			t.Env = append(t.Env, Env{Source: source, Name: e.Name, Value: e.Value})
		case KindJob:
			sched, _ := ParseSchedule(e.Schedule)
			j := Job{
				ID: lineID(source, e.Line, e.Text), Source: source, Line: e.Line,
				Schedule: sched, User: e.User, Command: e.Command, Enabled: e.Enabled,
				Next: []int64{}, Editable: source == SourceUser,
			}
			for at := now; len(j.Next) < 3; {
				at = sched.Next(at)
				if at.IsZero() {
					break
				}
				j.Next = append(j.Next, at.Unix())
			}
			t.Jobs = append(t.Jobs, j)
		}
	}
}

func lineID(source string, line int, text string) string {
	h := sha256.Sum256([]byte(text))
	return source + ":" + strconv.Itoa(line) + ":" + hex.EncodeToString(h[:6])
}

func version(text string) string {
	h := sha256.Sum256([]byte(text))
	return hex.EncodeToString(h[:8])
}

// userCrontab returns the user's crontab and whether there is one.
func (s *Service) userCrontab(ctx context.Context, serverID string) (string, bool, error) {
	res, err := s.run.Run(ctx, serverID, app, shellx.Cmd("crontab", "-l"))
	if err != nil {
		return "", false, err
	}
	switch {
	case res.Exit == 0:
		return string(res.Stdout), true, nil
	case res.Exit == 127:
		return "", false, apperr.New(apperr.CmdMissing, "cron is not installed on the server")
	case strings.Contains(string(res.Stderr), "no crontab for"):
		return "", false, nil
	case strings.Contains(string(res.Stderr), "not allowed"):
		// cron.allow / cron.deny keep this user out.
		return "", false, apperr.New(apperr.CmdNeedsRoot, "the user may not use cron")
	}
	return "", false, apperr.New(apperr.CmdFailed, "crontab -l failed")
}

// systemFiles lists /etc/crontab and the files cron reads in /etc/cron.d,
// skipping package leftovers and editor backups as cron does.
func systemFiles(sc *sftp.Client) []string {
	files := []string{"/etc/crontab"}
	entries, err := sc.ReadDir("/etc/cron.d")
	if err != nil {
		return files
	}
	var names []string
	for _, e := range entries {
		n := e.Name()
		if e.IsDir() || strings.HasPrefix(n, ".") || strings.HasSuffix(n, "~") ||
			strings.Contains(n, ".dpkg-") || strings.HasSuffix(n, ".rpmsave") || strings.HasSuffix(n, ".rpmnew") {
			continue
		}
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		files = append(files, path.Join("/etc/cron.d", n))
	}
	return files
}

func readFile(sc *sftp.Client, p string) (string, error) {
	f, err := sc.Open(p)
	if err != nil {
		return "", err
	}
	defer f.Close() //nolint:errcheck // read only
	data, err := io.ReadAll(io.LimitReader(f, 1<<20))
	return string(data), err
}

// serverZone is the zone cron schedules in: /etc/localtime's target, else
// /etc/timezone, else UTC.
func serverZone(sc *sftp.Client) *time.Location {
	if target, err := sc.ReadLink("/etc/localtime"); err == nil {
		if _, name, ok := strings.Cut(target, "zoneinfo/"); ok {
			if loc, err := time.LoadLocation(name); err == nil {
				return loc
			}
		}
	}
	if data, err := readFile(sc, "/etc/timezone"); err == nil {
		if loc, err := time.LoadLocation(strings.TrimSpace(data)); err == nil {
			return loc
		}
	}
	return time.UTC
}
