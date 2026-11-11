package database

import (
	"context"
	"slices"

	"davish/internal/audit"
	"davish/internal/shellx"
)

// Root runs commands as root for a user who is not (sudo.Keeper).
type Root interface {
	Run(ctx context.Context, serverID, app, line string, stdin []byte) (audit.Result, error)
	Prepare(ctx context.Context, serverID, app, line string) (string, []byte, error)
	State(ctx context.Context, serverID string) (string, error)
}

// UseRoot lets Davish reach MariaDB and PostgreSQL through r when only
// root may connect (ADR 0018): without a password when sudo allows, else
// with the session's, asked once. It replaces the sudo -n way in.
func (s *Service) UseRoot(r Root) { s.root = r }

// viaRoot marks, in access, a client reached through Root.
var viaRoot = []string{"\x00root"}

func isRoot(p []string) bool { return slices.Equal(p, viaRoot) }

// rootPrefixes are how root reaches each client: PostgreSQL's admin is
// the postgres system user, which root becomes with runuser.
var rootPrefixes = map[string][]string{
	MySQL:    {},
	Postgres: {"runuser", "-u", "postgres", "--"},
}

// tries are the ways in, in order: as the user, then through Root if
// there is one, else sudo -n.
func (s *Service) tries(kind string) [][]string {
	t := prefixes[kind]
	if s.root == nil {
		return t
	}
	return [][]string{t[0], viaRoot}
}

// rootLine prepares args, run as root, for a line the user's own shell
// runs: a dump writing to the user's home, a restore reading a file. in
// is what to write on its stdin first (the password line, or nil); the
// caller wipes it.
func (s *Service) rootLine(ctx context.Context, serverID, kind string, args ...string) (string, []byte, error) {
	all := append(append([]string{}, rootPrefixes[kind]...), args...)
	return s.root.Prepare(ctx, serverID, app, shellx.Cmd(all[0], all[1:]...).String())
}

func wipe(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
