package disk

import (
	"context"
	"errors"

	"davish/internal/apperr"
	"davish/internal/audit"
)

// Root runs a command as root for a user who is not (sudo.Keeper).
type Root interface {
	Run(ctx context.Context, serverID, app, line string, stdin []byte) (audit.Result, error)
}

// UseRoot lets Disk measure, clean and empty what needs root through r
// (ADR 0018): measuring as root whenever sudo needs no password or the
// session has it, and running the confirmed cleanup or truncate as root.
func (s *Service) UseRoot(r Root) { s.root = r }

// measureAsRoot runs a read-only line as root when that needs no prompt.
// ok is false when sudo needs a password Davish does not have (locked) or
// the user cannot use sudo, so the caller measures as the user instead.
func (s *Service) measureAsRoot(ctx context.Context, serverID, line string) (res audit.Result, ok, locked bool, err error) {
	if s.root == nil {
		return audit.Result{}, false, false, nil
	}
	res, err = s.root.Run(ctx, serverID, app, line, nil)
	switch codeOf(err) {
	case "":
		return res, err == nil, false, err
	case apperr.SudoPasswordRequired, apperr.SudoPasswordWrong:
		return audit.Result{}, false, true, nil
	case apperr.CmdNeedsRoot:
		return audit.Result{}, false, false, nil
	}
	return audit.Result{}, false, false, err
}

func codeOf(err error) apperr.Code {
	var e *apperr.Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}
