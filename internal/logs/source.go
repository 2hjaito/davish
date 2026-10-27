// Package logs tails journald units, files and container output on a
// server and streams the lines to the frontend.
package logs

import (
	"path"
	"strconv"

	"davish/internal/apperr"
	"davish/internal/docker"
	"davish/internal/shellx"
	"davish/internal/svc"
)

// Kinds of log source.
const (
	KindJournal = "journal"
	KindFile    = "file"
	KindDocker  = "docker"
)

// Backlog is how many existing lines a stream starts with.
const Backlog = 200

// Source is what to follow. Target is a unit (empty for the whole
// journal), an absolute file path or a container name.
type Source struct {
	Kind   string `json:"kind"`
	Target string `json:"target"`
}

// Command returns the command that follows src. It is the one the stream
// runs and the one shown in the audit log.
func Command(src Source) (shellx.Command, error) {
	n := strconv.Itoa(Backlog)
	switch src.Kind {
	case KindJournal:
		if src.Target == "" {
			return shellx.Cmd("journalctl", "-f", "-n", n, "-o", "short-iso", "--no-pager"), nil
		}
		if err := svc.CheckUnit(src.Target); err != nil {
			return shellx.Command{}, err
		}
		return shellx.Cmd("journalctl", "-u", src.Target, "-f", "-n", n, "-o", "short-iso", "--no-pager"), nil
	case KindFile:
		if !path.IsAbs(src.Target) || path.Clean(src.Target) != src.Target {
			return shellx.Command{}, apperr.New(apperr.InvalidInput, "log path must be absolute and clean")
		}
		// -F follows the name, so rotated logs keep coming.
		return shellx.Cmd("tail", "-F", "-n", n, "--", src.Target), nil
	case KindDocker:
		if err := docker.CheckName(src.Target); err != nil {
			return shellx.Command{}, err
		}
		return shellx.Cmd("docker", "logs", "-f", "--tail", n, src.Target), nil
	}
	return shellx.Command{}, apperr.New(apperr.InvalidInput, "unknown log source")
}
