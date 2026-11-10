package svc

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"davish/internal/apperr"
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
	// AskSudo is set when sudo could read them once given the password:
	// the UI offers to ask for it.
	AskSudo bool `json:"askSudo"`
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
	if !j.Restricted {
		return j, checkExit(res)
	}
	if s.root == nil {
		return j, nil
	}
	// Read as root when sudo lets the user without asking, or the
	// session has the password; never asks by itself.
	res, err = s.root.Run(ctx, serverID, app, cmd.String(), nil)
	switch code := codeOf(err); {
	case err == nil && res.Exit == 0:
		return parseJournal(res.Stdout, res.Stderr, j.Sudo), nil
	case code == apperr.SudoPasswordRequired:
		j.AskSudo = true
	case err != nil && code == "":
		// The connection failed, not sudo.
		return j, err
	}
	return j, nil
}

func codeOf(err error) apperr.Code {
	var e *apperr.Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
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
