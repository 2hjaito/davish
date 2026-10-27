package svc

import (
	"context"
	"strconv"
	"strings"

	"davish/internal/shellx"
)

// JournalLines is how many recent log lines Journal returns.
const JournalLines = 200

// Journal is a unit's latest log lines.
type Journal struct {
	Lines []string `json:"lines"`
	// Restricted is set when the user may not read system logs (not in
	// the adm or systemd-journal group), so Lines is empty for that reason
	// rather than because the unit logged nothing.
	Restricted bool `json:"restricted"`
	// Sudo reads the same lines as root, for the user to run in a terminal.
	Sudo string `json:"sudo"`
}

// Journal returns the last JournalLines log lines of unit, oldest first.
func (s *Service) Journal(ctx context.Context, serverID, unit string) (Journal, error) {
	if err := CheckUnit(unit); err != nil {
		return Journal{}, err
	}
	cmd := journalCommand(unit)
	res, err := s.run.Run(ctx, serverID, app, cmd)
	if err != nil {
		return Journal{}, err
	}
	j := parseJournal(res.Stdout, res.Stderr, shellx.Cmd("sudo", cmd.Args()...).String())
	if j.Restricted {
		return j, nil
	}
	return j, checkExit(res)
}

func journalCommand(unit string) shellx.Command {
	return shellx.Cmd("journalctl", "-u", unit, "-n", strconv.Itoa(JournalLines), "--no-pager", "-o", "short-iso")
}

func parseJournal(stdout, stderr []byte, sudo string) Journal {
	j := Journal{Lines: []string{}, Sudo: sudo}
	for line := range strings.Lines(string(stdout)) {
		line = strings.TrimRight(line, "\r\n")
		// journalctl marks an empty result and reboots with "-- … --".
		if line == "" || line == "-- No entries --" {
			continue
		}
		j.Lines = append(j.Lines, line)
	}
	// A user with a journal of their own only gets a hint (exit 0); one
	// without gets no journal at all (exit 1).
	msg := string(stderr)
	j.Restricted = strings.Contains(msg, "not seeing messages from other users") ||
		strings.Contains(msg, "insufficient permissions")
	return j
}
