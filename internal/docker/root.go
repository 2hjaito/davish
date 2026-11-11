package docker

import (
	"context"
	"errors"
	"strings"

	"davish/internal/apperr"
	"davish/internal/audit"
	"davish/internal/shellx"
)

// Root runs a command as root for a user who is not (sudo.Keeper).
type Root interface {
	Run(ctx context.Context, serverID, app, line string, stdin []byte) (audit.Result, error)
}

// UseRoot lets a user outside the docker group run docker through r
// (ADR 0018): what the socket refuses runs again, the same command, as
// root. Lines typed into a terminal (run, pull, exec) stay as they are.
func (s *Service) UseRoot(r Root) { s.root = r }

// exec runs cmd, and again as root when the docker socket refuses the
// user. When there is no way to root, the refusal is returned as it is,
// so checkExit gives CMD_NEEDS_ROOT and the UI offers a terminal; sudo's
// own errors, such as SUDO_PASSWORD_REQUIRED, reach the UI to ask.
func (s *Service) exec(ctx context.Context, serverID string, cmd shellx.Command) (audit.Result, error) {
	res, err := s.run.Run(ctx, serverID, app, cmd)
	if err != nil || s.root == nil || !socketDenied(res) {
		return res, err
	}
	rres, rerr := s.root.Run(ctx, serverID, app, cmd.String(), nil)
	if codeOf(rerr) == apperr.CmdNeedsRoot {
		return res, nil
	}
	return rres, rerr
}

// socketDenied tells a user outside the docker group: the socket is root's.
func socketDenied(res audit.Result) bool {
	msg := string(res.Stderr)
	return res.Exit != 0 && strings.Contains(msg, "permission denied") && strings.Contains(msg, "docker")
}

func codeOf(err error) apperr.Code {
	var e *apperr.Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}
