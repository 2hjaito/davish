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

// A page of rows, kept small so a peek never weighs on the app (ADR 0024).
const (
	sqlitePage     = 100
	sqliteCellText = 200
	sqliteMaxCols  = 64
	// sqliteSep and sqliteEnd end a cell and a row; text in a cell has
	// them replaced, so the output splits cleanly.
	sqliteSep = "\x1f"
	sqliteEnd = "\x1e"
)

// sqliteSeparators makes sqlite3 end cells and rows with sqliteSep and
// sqliteEnd, set on stdin so the command line stays printable.
const sqliteSeparators = ".separator \"\\037\" \"\\036\"\n"

// SQLiteColumn is one column of a table.
type SQLiteColumn struct {
	Name string `json:"name"`
	Type string `json:"type"`
	PK   bool   `json:"pk"`
}

// SQLiteCell is one value. Text longer than sqliteCellText is cut; a blob
// carries only its size.
type SQLiteCell struct {
	// Kind is null, num, text or blob.
	Kind string `json:"k"`
	Text string `json:"v,omitempty"`
	Cut  bool   `json:"cut,omitempty"`
	Size int64  `json:"size,omitempty"`
}

// SQLiteRows is one page of a table.
type SQLiteRows struct {
	Columns []SQLiteColumn `json:"columns"`
	// AllColumns counts the table's columns, beyond the sqliteMaxCols shown.
	AllColumns int            `json:"allColumns"`
	Rows       [][]SQLiteCell `json:"rows"`
	Offset     int            `json:"offset"`
	More       bool           `json:"more"`
}

// noSeps keeps the separators out of a text value.
func noSeps(expr string) string {
	return "replace(replace(" + expr + ", char(31), ' '), char(30), ' ')"
}

// sqliteColumnsSQL lists the columns of table, or nothing when no such
// table or view exists.
func sqliteColumnsSQL(table string) string {
	t := quoteText(table)
	return sqliteSeparators + "SELECT " + noSeps("name") + ", " + noSeps("type") + ", pk FROM pragma_table_info(" + t + ")" +
		" WHERE EXISTS (SELECT 1 FROM sqlite_master WHERE type IN ('table', 'view') AND name = " + t + ");\n"
}

// sqliteRowsSQL reads one page of table, each cell tagged with its kind:
// n (null), b<size>, t<text>, T<cut text>, v<number>.
func sqliteRowsSQL(table string, cols []SQLiteColumn, offset int) string {
	var b strings.Builder
	b.WriteString(sqliteSeparators + "SELECT ")
	for i, c := range cols {
		if i > 0 {
			b.WriteString(", ")
		}
		q := quoteIdent(c.Name)
		b.WriteString("CASE typeof(" + q + ") WHEN 'null' THEN 'n' WHEN 'blob' THEN 'b' || length(" + q + ")" +
			" WHEN 'text' THEN (CASE WHEN length(" + q + ") > " + strconv.Itoa(sqliteCellText) + " THEN 'T' ELSE 't' END) || " +
			noSeps("substr("+q+", 1, "+strconv.Itoa(sqliteCellText)+")") + " ELSE 'v' || " + q + " END")
	}
	b.WriteString(" FROM " + quoteIdent(table) + " LIMIT " + strconv.Itoa(sqlitePage+1) + " OFFSET " + strconv.Itoa(offset) + ";\n")
	return b.String()
}

// splitSQLite splits sqlite3 output into rows of cells.
func splitSQLite(out string) [][]string {
	var rows [][]string
	for _, r := range strings.Split(out, sqliteEnd) {
		if r == "" || r == "\n" {
			continue
		}
		rows = append(rows, strings.Split(r, sqliteSep))
	}
	return rows
}

func parseSQLiteCell(s string) SQLiteCell {
	if s == "" {
		return SQLiteCell{Kind: "null"}
	}
	switch v := s[1:]; s[0] {
	case 'b':
		n, _ := strconv.ParseInt(v, 10, 64)
		return SQLiteCell{Kind: "blob", Size: n}
	case 't':
		return SQLiteCell{Kind: "text", Text: v}
	case 'T':
		return SQLiteCell{Kind: "text", Text: v, Cut: true}
	case 'v':
		return SQLiteCell{Kind: "num", Text: v}
	default:
		return SQLiteCell{Kind: "null"}
	}
}

// RowsSQLite reads one page of up to sqlitePage rows of a table or view,
// read-only, from offset.
func (s *Service) RowsSQLite(ctx context.Context, serverID, file, table string, offset int) (SQLiteRows, error) {
	if err := checkSQLitePath(file); err != nil {
		return SQLiteRows{}, err
	}
	if table == "" || len(table) > 1024 || strings.ContainsAny(table, "\x00") || offset < 0 || offset > 1<<40 {
		return SQLiteRows{}, apperr.New(apperr.InvalidInput, "no such table, or a bad page")
	}
	out, _, err := s.runSQLite(ctx, serverID, file, sqliteColumnsSQL(table))
	if err != nil {
		return SQLiteRows{}, err
	}
	page := SQLiteRows{Offset: offset, Columns: []SQLiteColumn{}, Rows: [][]SQLiteCell{}}
	for _, c := range splitSQLite(out) {
		if len(c) != 3 {
			continue
		}
		page.Columns = append(page.Columns, SQLiteColumn{Name: c[0], Type: c[1], PK: c[2] != "0"})
	}
	if len(page.Columns) == 0 {
		return SQLiteRows{}, apperr.New(apperr.NotFound, "no such table")
	}
	page.AllColumns = len(page.Columns)
	if len(page.Columns) > sqliteMaxCols {
		page.Columns = page.Columns[:sqliteMaxCols]
	}
	out, _, err = s.runSQLite(ctx, serverID, file, sqliteRowsSQL(table, page.Columns, offset))
	if err != nil {
		return SQLiteRows{}, err
	}
	for _, r := range splitSQLite(out) {
		if len(r) != len(page.Columns) {
			continue
		}
		if len(page.Rows) == sqlitePage {
			page.More = true
			break
		}
		row := make([]SQLiteCell, len(r))
		for i, c := range r {
			row[i] = parseSQLiteCell(c)
		}
		page.Rows = append(page.Rows, row)
	}
	return page, nil
}
