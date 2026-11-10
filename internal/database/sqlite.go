package database

import (
	"context"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"davish/internal/apperr"
	"davish/internal/audit"
	"davish/internal/shellx"
)

// SQLite is a SQLite file, read with sqlite3 on the server (ADR 0017).
const SQLite = "sqlite"

// sqliteDumpExt ends the backups of SQLite files in ~/db-backups.
const sqliteDumpExt = ".sqlite.gz"

// SQLiteTable is one table and its rows.
type SQLiteTable struct {
	Name string `json:"name"`
	Rows int64  `json:"rows"`
}

// SQLiteInfo is what one SQLite file holds.
type SQLiteInfo struct {
	Path string `json:"path"`
	// Size is the database's pages in bytes.
	Size    int64         `json:"size"`
	Journal string        `json:"journal"`
	Tables  []SQLiteTable `json:"tables"`
	// Root is set when only root can read the file, through sudo -n.
	Root bool `json:"root"`
	// Backup is the base name its backups go under in ~/db-backups.
	Backup string `json:"backup"`
}

var sqlitePath = regexp.MustCompile(`^/[^\x00\n\r]*$`)

func checkSQLitePath(p string) error {
	if len(p) > 4096 || !sqlitePath.MatchString(p) || path.Clean(p) != p {
		return apperr.New(apperr.InvalidInput, "a SQLite file is an absolute path")
	}
	return nil
}

// sqliteLine opens file read-only, with SQL on stdin.
func sqliteLine(prefix []string, file string) string {
	return withPrefix(prefix, "timeout", "30", "sqlite3", "-readonly", "-batch", "-noheader", "-separator", "\t", file)
}

// runSQLite runs sql on file as the user, then through sudo -n when the
// user may not read it.
func (s *Service) runSQLite(ctx context.Context, serverID, file, sql string) (string, bool, error) {
	var last audit.Result
	for _, p := range [][]string{{}, {"sudo", "-n"}} {
		res, err := s.run.RunInput(ctx, serverID, app, sqliteLine(p, file), []byte(sql))
		if err != nil {
			return "", false, err
		}
		if res.Exit == 0 {
			return string(res.Stdout), len(p) > 0, nil
		}
		last = res
		out := string(res.Stderr)
		switch {
		case res.Exit == 127 && len(p) == 0:
			return "", false, apperr.New(apperr.CmdMissing, "sqlite3 is not installed")
		case strings.Contains(out, "not a database"):
			return "", false, apperr.New(apperr.InvalidInput, "this file is not a SQLite database")
		case !strings.Contains(out, "unable to open") && !denied(out):
			return "", false, sqlError(res)
		}
	}
	if strings.Contains(string(last.Stderr), "sudo:") {
		return "", false, apperr.New(apperr.CmdNeedsRoot, "only root may read this file")
	}
	return "", false, apperr.New(apperr.NotFound, "no such SQLite file, or it cannot be opened")
}

// quoteIdent quotes a SQLite table name.
func quoteIdent(n string) string {
	return `"` + strings.ReplaceAll(n, `"`, `""`) + `"`
}

func quoteText(v string) string {
	return "'" + strings.ReplaceAll(v, "'", "''") + "'"
}

// InspectSQLite reads the tables of a SQLite file and their rows.
func (s *Service) InspectSQLite(ctx context.Context, serverID, file string) (SQLiteInfo, error) {
	if err := checkSQLitePath(file); err != nil {
		return SQLiteInfo{}, err
	}
	out, root, err := s.runSQLite(ctx, serverID, file, "PRAGMA journal_mode;\nPRAGMA page_count;\nPRAGMA page_size;\n"+
		"SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name;\n")
	if err != nil {
		return SQLiteInfo{}, err
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	info := SQLiteInfo{Path: file, Root: root, Tables: []SQLiteTable{}, Backup: sqliteBackupName(file)}
	if len(lines) < 3 {
		return SQLiteInfo{}, apperr.New(apperr.CmdFailed, "sqlite3 answered nothing")
	}
	info.Journal = lines[0]
	pages, _ := strconv.ParseInt(lines[1], 10, 64)
	size, _ := strconv.ParseInt(lines[2], 10, 64)
	info.Size = pages * size
	var count strings.Builder
	for _, n := range lines[3:] {
		if n == "" {
			continue
		}
		if count.Len() > 0 {
			count.WriteString(" UNION ALL ")
		}
		count.WriteString("SELECT " + quoteText(n) + ", count(*) FROM " + quoteIdent(n))
	}
	if count.Len() == 0 {
		return info, nil
	}
	out, _, err = s.runSQLite(ctx, serverID, file, count.String()+";\n")
	if err != nil {
		return SQLiteInfo{}, err
	}
	for l := range strings.Lines(out) {
		name, rows, ok := strings.Cut(strings.TrimRight(l, "\r\n"), "\t")
		if !ok {
			continue
		}
		n, _ := strconv.ParseInt(rows, 10, 64)
		info.Tables = append(info.Tables, SQLiteTable{Name: name, Rows: n})
	}
	return info, nil
}

var unsafeName = regexp.MustCompile(`[^A-Za-z0-9_.-]+`)

// sqliteBackupName is the name a file's backups go under: its base name,
// with anything a shell or Dumps would trip on replaced.
func sqliteBackupName(file string) string {
	n := strings.Trim(unsafeName.ReplaceAllString(path.Base(file), "_"), ".")
	if n == "" {
		n = "sqlite"
	}
	if len(n) > 60 {
		n = n[:60]
	}
	return n
}

// sqliteBackupLine copies file with SQLite's online backup, safe while an
// app writes to it, then compresses the copy and prints its path. A copy
// made through sudo is handed to the user.
func sqliteBackupLine(prefix []string, file, stamp string) string {
	name := sqliteBackupName(file) + "-" + stamp + ".sqlite"
	f := `"$HOME/db-backups/"` + shellx.Quote(name)
	line := `mkdir -p "$HOME/db-backups" && ` + withPrefix(prefix, "sqlite3", file) +
		` ".backup '$HOME/db-backups/` + name + `'"`
	if len(prefix) > 0 {
		line += " && " + withPrefix(prefix, "chown") + ` "$(id -u):$(id -g)" ` + f
	}
	return line + " && gzip -- " + f + " && echo " + f + ".gz"
}

// BackupSQLite copies file to ~/db-backups and returns the backup's path.
func (s *Service) BackupSQLite(ctx context.Context, serverID, file string) (string, error) {
	info, err := s.InspectSQLite(ctx, serverID, file)
	if err != nil {
		return "", err
	}
	var prefix []string
	if info.Root {
		prefix = []string{"sudo", "-n"}
	}
	stamp := time.Now().UTC().Format("20060102T150405Z")
	res, err := s.run.RunInput(ctx, serverID, app, sqliteBackupLine(prefix, file, stamp), nil)
	if err != nil {
		return "", err
	}
	if res.Exit != 0 {
		return "", sqlError(res)
	}
	return strings.TrimSpace(string(res.Stdout)), nil
}

// SQLiteRestoreLine puts a backup back in place of file, keeping the file
// as it was beside it. Davish only types it into a terminal: the app that
// uses the file has to be stopped first, which Davish cannot know.
func SQLiteRestoreLine(file, backup string) (string, error) {
	if err := checkSQLitePath(file); err != nil {
		return "", err
	}
	if !strings.HasSuffix(backup, sqliteDumpExt) || checkSQLitePath(backup) != nil {
		return "", apperr.New(apperr.InvalidInput, "a SQLite backup is a .sqlite.gz file")
	}
	return shellx.Cmd("cp", "-p", "--", file, file+".before-restore").String() + " && " +
		shellx.Cmd("gzip", "-dc", "--", backup).String() + " > " + shellx.Quote(file), nil
}
