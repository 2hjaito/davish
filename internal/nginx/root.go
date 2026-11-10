package nginx

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

// UseRoot lets what the server refuses for lack of root run again as root
// through r (ADR 0018). certbot stays in a terminal: it may ask questions.
func (s *Service) UseRoot(r Root) { s.root = r }

// asRoot runs line again as root after the server refused it. ok is false
// when there is no way to root, so the caller keeps its own answer and the
// UI offers a terminal; err is sudo's, such as SUDO_PASSWORD_REQUIRED for
// the UI to ask.
func (s *Service) asRoot(ctx context.Context, serverID, line string) (res audit.Result, ok bool, err error) {
	if s.root == nil {
		return audit.Result{}, false, nil
	}
	res, err = s.root.Run(ctx, serverID, app, line, nil)
	if codeOf(err) == apperr.CmdNeedsRoot {
		return res, false, nil
	}
	return res, err == nil, err
}

func codeOf(err error) apperr.Code {
	var e *apperr.Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}
