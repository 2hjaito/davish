package cron

import (
	"context"
	"strings"

	"davish/internal/shellx"
)

// RunLines is how many of a job's latest log lines Runs returns.
const RunLines = 20

// Runs is what the system log says about a job's latest runs.
type Runs struct {
	Lines []string `json:"lines"`
	// Restricted is set when the user may not read the system journal,
	// so Lines is empty for that reason.
	Restricted bool `json:"restricted"`
	// Sudo reads the same log as root, for a terminal.
	Sudo string `json:"sudo"`
}

// runsCommand reads cron's own lines from the journal: CRON on Debian and
// Ubuntu, CROND on Red Hat systems.
var runsCommand = shellx.Cmd("journalctl", "-t", "CRON", "-t", "CROND", "-n", "5000", "--no-pager", "-o", "short-iso")

// Runs returns the latest log lines that cron wrote when starting command,
// oldest first.
func (s *Service) Runs(ctx context.Context, serverID, command string) (Runs, error) {
	res, err := s.run.Run(ctx, serverID, app, runsCommand)
	if err != nil {
		return Runs{}, err
	}
	r := parseRuns(string(res.Stdout), string(res.Stderr), command)
	r.Sudo = shellx.Cmd("sudo", runsCommand.Args()...).String()
	return r, nil
}

// parseRuns keeps the "CMD (command)" lines of command. Cron logs the
// command as written in the crontab, so a plain substring match finds it.
func parseRuns(stdout, stderr, command string) Runs {
	r := Runs{Lines: []string{}}
	needle := "CMD (" + strings.TrimSpace(command) + ")"
	for line := range strings.Lines(stdout) {
		if strings.Contains(line, needle) {
			r.Lines = append(r.Lines, strings.TrimRight(line, "\r\n"))
		}
	}
	if len(r.Lines) > RunLines {
		r.Lines = r.Lines[len(r.Lines)-RunLines:]
	}
	r.Restricted = strings.Contains(stderr, "not seeing messages from other users") ||
		strings.Contains(stderr, "insufficient permissions")
	return r
}
