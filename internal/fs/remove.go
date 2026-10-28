package fs

import (
	"context"
	"strings"

	"davish/internal/apperr"
	"davish/internal/shellx"
)

// RemovePlan is what the confirmation dialog shows before a delete.
type RemovePlan struct {
	Entry
	// NonEmpty marks a directory with content; the UI then asks a second
	// time and makes the user type the name.
	NonEmpty bool `json:"nonEmpty"`
	// Command is exactly what Remove runs on the server.
	Command string `json:"command"`
	// Sudo is the same command with sudo, for a terminal when the server
	// refuses for lack of rights (docs/SECURITY.md).
	Sudo string `json:"sudo"`
}

// PlanRemove describes how p would be deleted. Remove only runs the
// command the user saw, so the plan must be shown first.
func (s *Service) PlanRemove(_ context.Context, serverID, p string) (RemovePlan, error) {
	if err := CheckPath(p); err != nil {
		return RemovePlan{}, err
	}
	if p == "/" {
		return RemovePlan{}, apperr.New(apperr.InvalidInput, "cannot remove the root directory")
	}
	sc, err := s.conns.SFTP(serverID)
	if err != nil {
		return RemovePlan{}, err
	}
	e, err := stat(sc, p)
	if err != nil {
		return RemovePlan{}, err
	}
	plan := RemovePlan{Entry: e}
	if e.Kind == KindDir {
		// A link is removed itself, never what it points to.
		children, err := sc.ReadDir(p)
		if err != nil {
			return RemovePlan{}, mapErr("remove", err)
		}
		plan.NonEmpty = len(children) > 0
	}
	cmd := removeCommand(e.Kind, plan.NonEmpty, p)
	plan.Command = cmd.String()
	plan.Sudo = shellx.Cmd("sudo", cmd.Args()...).String()
	return plan, nil
}

func removeCommand(kind string, nonEmpty bool, p string) shellx.Command {
	switch {
	case kind != KindDir:
		return shellx.Cmd("rm", "--", p)
	case nonEmpty:
		return shellx.Cmd("rm", "-r", "--", p)
	}
	return shellx.Cmd("rmdir", "--", p)
}

// Remove deletes p by running command, which must be the one PlanRemove
// returned and the user confirmed. If p changed since (a file became a
// directory, an empty directory filled up), it fails with FS_STALE so the
// user is asked again.
func (s *Service) Remove(ctx context.Context, serverID, p, command string) error {
	plan, err := s.PlanRemove(ctx, serverID, p)
	if err != nil {
		return err
	}
	if plan.Command != command {
		return apperr.New(apperr.FSStale, "path changed since the delete was confirmed")
	}
	s.log.Info("remove", "server", serverID, "app", "files", "op", "remove")
	return s.shell(ctx, serverID, command)
}

// shell runs command through the audit runner and maps a failure to an
// error code from its stderr.
func (s *Service) shell(ctx context.Context, serverID, command string) error {
	res, err := s.run.RunLine(ctx, serverID, "files", command)
	if err != nil {
		return err
	}
	if res.Exit == 0 {
		return nil
	}
	if res.Exit == 127 { // the shell could not find the program
		return apperr.New(apperr.CmdMissing, "program not installed on the server")
	}
	msg := string(res.Stderr)
	switch {
	case strings.Contains(msg, "Permission denied"), strings.Contains(msg, "Operation not permitted"):
		return apperr.New(apperr.FSPermissionDenied, "permission denied")
	case strings.Contains(msg, "No such file"):
		return apperr.New(apperr.NotFound, "no such file or directory")
	}
	return apperr.New(apperr.Internal, "command failed")
}
