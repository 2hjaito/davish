package cron

import (
	"context"
	"strings"
	"time"

	"davish/internal/apperr"
	"davish/internal/shellx"
)

// Change ops on the user's crontab.
const (
	OpAdd     = "add"
	OpEdit    = "edit"
	OpDelete  = "delete"
	OpEnable  = "enable"
	OpDisable = "disable"
)

// Change is one edit of the user's crontab. ID names the job for every op
// but add; Schedule and Command are the job's new content for add and edit.
type Change struct {
	Op       string `json:"op"`
	ID       string `json:"id"`
	Schedule string `json:"schedule"`
	Command  string `json:"command"`
}

// Plan is what the confirmation shows before the crontab is replaced.
type Plan struct {
	// Command is exactly what runs; the new crontab goes in on stdin.
	Command string `json:"command"`
	// Before and After are the line removed and the line written; one is
	// empty for add and delete.
	Before string `json:"before"`
	After  string `json:"after"`
	// Backup is where the current crontab is saved first, if there is one.
	Backup string `json:"backup"`
	// Version is the crontab the plan was made from; Apply refuses if the
	// crontab changed since.
	Version string `json:"version"`
}

// backupStamp is the UTC time in a backup's name.
const backupStamp = "20060102T150405Z"

// Plan works out change against the current crontab without changing it.
func (s *Service) Plan(ctx context.Context, serverID string, c Change) (Plan, error) {
	p, _, err := s.plan(ctx, serverID, c, s.now())
	return p, err
}

// Apply makes change exactly as confirmed: the plan is worked out again
// with the confirmed backup name, and must still give the same command
// from the same crontab. The previous crontab is saved first; if that
// fails nothing is written.
func (s *Service) Apply(ctx context.Context, serverID string, c Change, confirmed Plan) error {
	// The backup is named when the plan is shown, not when it runs: a
	// second ticking over in between must not change the command.
	stamp := s.now()
	if i := strings.LastIndex(confirmed.Backup, ".crontab.bak-"); i >= 0 {
		t, err := time.Parse(backupStamp, confirmed.Backup[i+len(".crontab.bak-"):])
		if err != nil {
			return apperr.New(apperr.InvalidInput, "not a backup name")
		}
		stamp = t
	}
	p, text, err := s.plan(ctx, serverID, c, stamp)
	if err != nil {
		return err
	}
	if p.Version != confirmed.Version || p.Command != confirmed.Command {
		return apperr.New(apperr.FSStale, "the crontab changed since the change was confirmed")
	}
	res, err := s.run.RunInput(ctx, serverID, app, p.Command, []byte(text))
	if err != nil {
		return err
	}
	if res.Exit != 0 {
		if strings.Contains(string(res.Stderr), "Permission denied") {
			return apperr.New(apperr.FSPermissionDenied, "cannot write the backup")
		}
		return apperr.New(apperr.CmdFailed, "crontab refused the new table")
	}
	return nil
}

// plan returns the plan and the new crontab text.
func (s *Service) plan(ctx context.Context, serverID string, c Change, stamp time.Time) (Plan, string, error) {
	cur, exists, err := s.userCrontab(ctx, serverID)
	if err != nil {
		return Plan{}, "", err
	}
	lines := []string{}
	if cur != "" {
		lines = strings.Split(strings.TrimSuffix(cur, "\n"), "\n")
	}
	p := Plan{Version: version(cur)}

	var newLine string
	if c.Op == OpAdd || c.Op == OpEdit {
		if newLine, err = jobLine(c.Schedule, c.Command); err != nil {
			return Plan{}, "", err
		}
	}
	if c.Op == OpAdd {
		lines = append(lines, newLine)
		p.After = newLine
	} else {
		i, err := findJob(cur, c.ID)
		if err != nil {
			return Plan{}, "", err
		}
		p.Before = lines[i]
		switch c.Op {
		case OpEdit:
			lines[i] = newLine
		case OpDelete:
			lines = append(lines[:i], lines[i+1:]...)
		case OpEnable:
			lines[i] = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(lines[i]), "#"))
		case OpDisable:
			lines[i] = "#" + strings.TrimSpace(lines[i])
		default:
			return Plan{}, "", apperr.New(apperr.InvalidInput, "unknown change")
		}
		if c.Op != OpDelete {
			p.After = lines[i]
		}
	}
	text := strings.Join(lines, "\n") + "\n"
	if len(lines) == 0 {
		text = ""
	}

	install := shellx.Cmd("crontab", "-").String()
	p.Command = install
	if exists {
		home, err := s.home(serverID)
		if err != nil {
			return Plan{}, "", err
		}
		p.Backup = home + "/.crontab.bak-" + stamp.UTC().Format(backupStamp)
		p.Command = shellx.Cmd("crontab", "-l").String() + " > " + shellx.Quote(p.Backup) + " && " + install
	}
	return p, text, nil
}

// jobLine validates a job and renders its line. A line break in the
// command would smuggle a second job in.
func jobLine(schedule, command string) (string, error) {
	sched, err := ParseSchedule(schedule)
	if err != nil {
		return "", apperr.New(apperr.InvalidInput, "invalid schedule")
	}
	command = strings.TrimSpace(command)
	if command == "" || strings.ContainsAny(command, "\r\n\x00") {
		return "", apperr.New(apperr.InvalidInput, "invalid command")
	}
	return strings.Join(strings.Fields(sched.Raw), " ") + " " + command, nil
}

// findJob returns the index of the user's job id names in cur, which must
// still have the same content at the same line.
func findJob(cur, id string) (int, error) {
	for _, e := range ParseCrontab(cur, false) {
		if e.Kind == KindJob && lineID(SourceUser, e.Line, e.Text) == id {
			return e.Line - 1, nil
		}
	}
	return 0, apperr.New(apperr.FSStale, "the job changed since it was shown")
}

func (s *Service) home(serverID string) (string, error) {
	sc, err := s.conns.SFTP(serverID)
	if err != nil {
		return "", err
	}
	home, err := sc.Getwd()
	if err != nil {
		return "", apperr.Wrap(apperr.Internal, "cannot find the home folder", err)
	}
	return home, nil
}
