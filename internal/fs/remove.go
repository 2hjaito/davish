package fs

import (
	"bytes"
	"context"
	"errors"
	"strings"

	"golang.org/x/crypto/ssh"

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
	plan.Command = removeCommand(e.Kind, plan.NonEmpty, p).String()
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
	client, err := s.conns.Client(serverID)
	if err != nil {
		return err
	}
	// TODO(M7): record the command in audit.
	s.log.Info("remove", "server", serverID, "app", "files", "op", "remove")
	return run(ctx, client, command)
}

// run executes command and maps a failure to an error code from its stderr.
func run(ctx context.Context, client *ssh.Client, command string) error {
	sess, err := client.NewSession()
	if err != nil {
		return apperr.Wrap(apperr.ConnNotConnected, "cannot open a session", err)
	}
	defer sess.Close() //nolint:errcheck // closed either way

	stop := context.AfterFunc(ctx, func() { sess.Close() }) //nolint:errcheck,gosec // unblocks Run
	defer stop()

	var stderr bytes.Buffer
	sess.Stderr = &stderr
	err = sess.Run(command)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err == nil {
		return nil
	}
	var exit *ssh.ExitError
	if !errors.As(err, &exit) {
		return apperr.Wrap(apperr.ConnNotConnected, "command did not finish", err)
	}
	msg := stderr.String()
	switch {
	case strings.Contains(msg, "Permission denied"), strings.Contains(msg, "Operation not permitted"):
		return apperr.Wrap(apperr.FSPermissionDenied, "permission denied", err)
	case strings.Contains(msg, "No such file"):
		return apperr.Wrap(apperr.NotFound, "no such file or directory", err)
	}
	return apperr.Wrap(apperr.Internal, "command failed", err)
}
