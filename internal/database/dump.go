package database

import (
	"context"
	"path"
	"regexp"
	"strings"
	"time"

	"davish/internal/apperr"
	"davish/internal/shellx"
)

// dumpDir is where dumps go, in the user's home.
const dumpDir = `"$HOME/db-backups"`

var dumpTools = map[string][]string{
	MySQL:    {"mysqldump", "--single-transaction", "--routines", "--triggers"},
	Postgres: {"pg_dump", "--no-owner"},
}

// way returns the prefix that reaches kind's client, finding it once.
func (s *Service) way(ctx context.Context, serverID, kind string) ([]string, error) {
	key := serverID + "/" + kind
	s.mu.Lock()
	p, ok := s.access[key]
	s.mu.Unlock()
	if ok {
		return p, nil
	}
	if _, err := s.exec(ctx, serverID, kind, "SELECT 1;"); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.access[key], nil
}

func withPrefix(prefix []string, args ...string) string {
	all := append(append([]string{}, prefix...), args...)
	return shellx.Cmd(all[0], all[1:]...).String()
}

// dumpLine writes db to dumpDir/<db>-<stamp>.sql, compresses it, and
// prints the file's path. The dump goes to a file before gzip, since sh
// has no pipefail and a failed dump must not leave a good-looking file.
func dumpLine(prefix []string, kind, db, stamp string) string {
	f := `"$HOME/db-backups/` + db + "-" + stamp + `.sql"`
	args := append(append([]string{}, dumpTools[kind]...), db)
	return "mkdir -p " + dumpDir + " && " + withPrefix(prefix, args...) + " > " + f +
		" && gzip -- " + f + ` && echo "$HOME/db-backups/` + db + "-" + stamp + `.sql.gz"`
}

// Dump writes db to a compressed file in ~/db-backups and returns its path.
func (s *Service) Dump(ctx context.Context, serverID, kind, db string) (string, error) {
	if !safeName.MatchString(db) {
		return "", apperr.New(apperr.InvalidInput, "not a database name")
	}
	if _, ok := dumpTools[kind]; !ok {
		return "", apperr.New(apperr.InvalidInput, "unknown engine")
	}
	prefix, err := s.way(ctx, serverID, kind)
	if err != nil {
		return "", err
	}
	stamp := time.Now().UTC().Format("20060102T150405Z")
	res, err := s.run.RunInput(ctx, serverID, app, dumpLine(prefix, kind, db, stamp), nil)
	if err != nil {
		return "", err
	}
	if res.Exit != 0 {
		return "", sqlError(res)
	}
	return strings.TrimSpace(string(res.Stdout)), nil
}

// Dumps lists the dumps of db in ~/db-backups, newest first.
func (s *Service) Dumps(ctx context.Context, serverID, db string) ([]string, error) {
	if !safeName.MatchString(db) {
		return nil, apperr.New(apperr.InvalidInput, "not a database name")
	}
	line := `ls -1t "$HOME/db-backups"/` + shellx.Quote(db) + `-*.sql.gz 2>/dev/null; true`
	res, err := s.run.RunInput(ctx, serverID, app, line, nil)
	if err != nil {
		return nil, err
	}
	out := []string{}
	for l := range strings.Lines(string(res.Stdout)) {
		if l = strings.TrimSpace(l); strings.HasPrefix(l, "/") {
			out = append(out, l)
		}
	}
	return out, nil
}

var dumpFile = regexp.MustCompile(`^/[^\x00\n\r]*\.sql(\.gz)?$`)

var restoreTools = map[string][]string{
	MySQL:    {"mysql", "--batch"},
	Postgres: {"psql", "-X", "-q", "-v", "ON_ERROR_STOP=1", "-d"},
}

// RestoreCommand is the line that loads file into db, for the
// confirmation. A .gz file is tested before it is read.
func (s *Service) RestoreCommand(ctx context.Context, serverID, kind, db, file string) (string, error) {
	if !safeName.MatchString(db) {
		return "", apperr.New(apperr.InvalidInput, "not a database name")
	}
	if !dumpFile.MatchString(file) || path.Clean(file) != file {
		return "", apperr.New(apperr.InvalidInput, "a dump is a .sql or .sql.gz file")
	}
	tool, ok := restoreTools[kind]
	if !ok {
		return "", apperr.New(apperr.InvalidInput, "unknown engine")
	}
	prefix, err := s.way(ctx, serverID, kind)
	if err != nil {
		return "", err
	}
	return restoreLine(prefix, tool, db, file), nil
}

func restoreLine(prefix, tool []string, db, file string) string {
	client := withPrefix(prefix, append(append([]string{}, tool...), db)...)
	if strings.HasSuffix(file, ".gz") {
		return shellx.Cmd("gzip", "-t", "--", file).String() + " && " +
			shellx.Cmd("gzip", "-dc", "--", file).String() + " | " + client
	}
	return client + " < " + shellx.Quote(file)
}

// Restore runs command, which must be what RestoreCommand gives now.
func (s *Service) Restore(ctx context.Context, serverID, kind, db, file, command string) error {
	want, err := s.RestoreCommand(ctx, serverID, kind, db, file)
	if err != nil {
		return err
	}
	if want != command {
		return apperr.New(apperr.FSStale, "the command changed since it was confirmed")
	}
	res, err := s.run.RunInput(ctx, serverID, app, command, nil)
	if err != nil {
		return err
	}
	if res.Exit != 0 {
		return sqlError(res)
	}
	return nil
}
