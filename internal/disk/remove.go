package disk

import (
	"context"
	"path"
	"slices"
	"strings"

	"davish/internal/apperr"
	"davish/internal/fs"
	"davish/internal/shellx"
)

// Remover deletes files the way Files does: plan, show, then run the
// confirmed command.
type Remover interface {
	PlanRemove(ctx context.Context, serverID, p string) (fs.RemovePlan, error)
	Remove(ctx context.Context, serverID, p, command string) error
}

// systemTrees are never removed or emptied from the Disk app, nor anything
// in them: the system, package databases and database data. Their space
// is freed through the cleanups or the app that owns them.
var systemTrees = []string{
	"/bin", "/boot", "/dev", "/etc", "/lib", "/lib32", "/lib64", "/libx32", "/proc",
	"/run", "/sbin", "/snap", "/sys", "/usr",
	"/var/lib/apt", "/var/lib/containerd", "/var/lib/docker", "/var/lib/dpkg",
	"/var/lib/mysql", "/var/lib/pgsql", "/var/lib/postgresql", "/var/lib/rpm", "/var/lib/snapd",
}

// emptiable are the parts of systemTrees whose files may be emptied: a
// container's log keeps being written after truncate.
var emptiable = []string{"/var/lib/docker/containers"}

// standardDirs are folders that must stay even though what is in them may
// go.
var standardDirs = []string{
	"/", "/home", "/media", "/mnt", "/opt", "/root", "/srv", "/tmp",
	"/var", "/var/backups", "/var/cache", "/var/lib", "/var/log", "/var/mail", "/var/spool",
	"/var/tmp", "/var/www",
}

func under(p string, trees []string) bool {
	return slices.ContainsFunc(trees, func(t string) bool { return p == t || strings.HasPrefix(p, t+"/") })
}

// Protected tells whether the Disk app refuses to remove p: a system
// tree, a standard folder or a home folder itself.
func Protected(p string) bool {
	p = path.Clean(p)
	return under(p, systemTrees) || slices.Contains(standardDirs, p) || path.Dir(p) == "/home"
}

// protectedFile tells whether the Disk app refuses to empty file p.
func protectedFile(p string) bool {
	p = path.Clean(p)
	return under(p, systemTrees) && !under(p, emptiable)
}

// PlanRemove describes how p would be deleted, refusing protected paths.
func (s *Service) PlanRemove(ctx context.Context, serverID, p string) (fs.RemovePlan, error) {
	if err := s.checkRemove(p); err != nil {
		return fs.RemovePlan{}, err
	}
	return s.files.PlanRemove(ctx, serverID, p)
}

// Remove deletes p with the command the user confirmed.
func (s *Service) Remove(ctx context.Context, serverID, p, command string) error {
	if err := s.checkRemove(p); err != nil {
		return err
	}
	return s.files.Remove(ctx, serverID, p, command)
}

func (s *Service) checkRemove(p string) error {
	if err := fs.CheckPath(p); err != nil {
		return err
	}
	if Protected(p) {
		return apperr.New(apperr.InvalidInput, "the Disk app does not remove system folders")
	}
	return nil
}

// TruncatePlan is what the confirmation shows before emptying a file.
type TruncatePlan struct {
	Command string `json:"command"`
	// Sudo is the same with sudo, for a terminal when the file is root's.
	Sudo string `json:"sudo"`
}

// PlanTruncate describes how file p would be emptied, keeping the file so
// a program writing it carries on.
func PlanTruncate(p string) (TruncatePlan, error) {
	if err := fs.CheckPath(p); err != nil {
		return TruncatePlan{}, err
	}
	if p == "/" || protectedFile(p) {
		return TruncatePlan{}, apperr.New(apperr.InvalidInput, "the Disk app does not empty system files")
	}
	cmd := shellx.Cmd("truncate", "-s", "0", "--", p)
	return TruncatePlan{Command: cmd.String(), Sudo: shellx.Cmd("sudo", cmd.Args()...).String()}, nil
}

// Truncate empties p with the command the user confirmed.
func (s *Service) Truncate(ctx context.Context, serverID, p, command string) error {
	plan, err := PlanTruncate(p)
	if err != nil {
		return err
	}
	if command != plan.Command {
		return apperr.New(apperr.InvalidInput, "command does not match what was confirmed")
	}
	res, err := s.run.RunLine(ctx, serverID, app, plan.Command)
	if err != nil {
		return err
	}
	msg := string(res.Stderr)
	switch {
	case res.Exit == 0:
		return nil
	case strings.Contains(msg, "Permission denied"), strings.Contains(msg, "Operation not permitted"):
		return apperr.New(apperr.FSPermissionDenied, "permission denied")
	case strings.Contains(msg, "No such file"):
		return apperr.New(apperr.NotFound, "no such file")
	case strings.Contains(msg, "Is a directory"):
		return apperr.New(apperr.InvalidInput, "not a file")
	}
	return apperr.New(apperr.CmdFailed, strings.TrimSpace(msg))
}
