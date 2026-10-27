package fs

import (
	"context"
	"errors"
	"path"

	"golang.org/x/crypto/ssh"

	"davish/internal/apperr"
	"davish/internal/shellx"
)

// writableCommand asks the server whether Save can replace p: the file and
// its folder must be writable (the new version is written next to it and
// renamed over it), and the file must be ours, or we must be root, or the
// file's owner would change. test(1) asks the kernel, so ACLs and root
// count as they should.
func writableCommand(p string) string {
	return shellx.Cmd("test", "-w", p).String() +
		" && " + shellx.Cmd("test", "-w", path.Dir(p)).String() +
		" && { " + shellx.Cmd("test", "-O", p).String() + ` || test "$(id -u)" = 0; }`
}

// Access tells the Editor whether it may save a file, and what to type
// in a terminal to edit it as root otherwise.
type Access struct {
	Writable bool `json:"writable"`
	// SudoEdit is the command to prefill in a terminal; the user presses
	// Enter and types the sudo password there (docs/SECURITY.md).
	SudoEdit string `json:"sudoEdit"`
}

// Access reports whether Save could replace the file at p.
func (s *Service) Access(ctx context.Context, serverID, p string) (Access, error) {
	if err := CheckPath(p); err != nil {
		return Access{}, err
	}
	ok, err := s.writable(ctx, serverID, p)
	if err != nil {
		return Access{}, err
	}
	return Access{Writable: ok, SudoEdit: shellx.Cmd("sudoedit", "--", p).String()}, nil
}

func (s *Service) writable(ctx context.Context, serverID, p string) (bool, error) {
	client, err := s.conns.Client(serverID)
	if err != nil {
		return false, err
	}
	sess, err := client.NewSession()
	if err != nil {
		return false, apperr.Wrap(apperr.ConnNotConnected, "cannot open a session", err)
	}
	defer sess.Close()                                      //nolint:errcheck // closed either way
	stop := context.AfterFunc(ctx, func() { sess.Close() }) //nolint:errcheck,gosec // unblocks Run
	defer stop()

	err = sess.Run(writableCommand(p))
	var exit *ssh.ExitError
	switch {
	case ctx.Err() != nil:
		return false, ctx.Err()
	case err == nil:
		return true, nil
	case errors.As(err, &exit):
		return false, nil
	}
	return false, apperr.Wrap(apperr.ConnNotConnected, "command did not finish", err)
}
