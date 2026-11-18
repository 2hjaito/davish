package database

import (
	"bytes"
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"davish/internal/apperr"
	"davish/internal/audit"
)

func TestSQLiteBackupName(t *testing.T) {
	for in, want := range map[string]string{
		"/srv/app/app.db":               "app.db",
		"/var/lib/x/my db's (1).sqlite": "my_db_s_1_.sqlite",
		"/x/.hidden.db":                 "hidden.db",
		"/x/...":                        "sqlite",
	} {
		if got := sqliteBackupName(in); got != want {
			t.Errorf("sqliteBackupName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSQLiteBackupLine(t *testing.T) {
	got := sqliteBackupLine([]string{"sudo", "-n"}, "/var/lib/vault/db.sqlite3", "20261110T000000Z")
	want := `mkdir -p "$HOME/db-backups" && sudo -n sqlite3 /var/lib/vault/db.sqlite3 ".backup '$HOME/db-backups/db.sqlite3-20261110T000000Z.sqlite'"` +
		` && sudo -n chown "$(id -u):$(id -g)" "$HOME/db-backups/"db.sqlite3-20261110T000000Z.sqlite` +
		` && gzip -- "$HOME/db-backups/"db.sqlite3-20261110T000000Z.sqlite && echo "$HOME/db-backups/"db.sqlite3-20261110T000000Z.sqlite.gz`
	if got != want {
		t.Errorf("backup =\n%s\nwant\n%s", got, want)
	}
}

func TestSQLiteRestoreLine(t *testing.T) {
	got, err := SQLiteRestoreLine("/srv/my app/app.db", "/home/u/db-backups/app.db-x.sqlite.gz")
	want := "cp -p -- '/srv/my app/app.db' '/srv/my app/app.db.before-restore' && gzip -dc -- /home/u/db-backups/app.db-x.sqlite.gz > '/srv/my app/app.db'"
	if err != nil || got != want {
		t.Errorf("restore = %s, %v", got, err)
	}
	for _, bad := range [][2]string{{"rel.db", "/a.sqlite.gz"}, {"/a.db", "/a.sql.gz"}, {"/a/../b.db", "/a.sqlite.gz"}} {
		if _, err := SQLiteRestoreLine(bad[0], bad[1]); err == nil {
			t.Errorf("%v accepted", bad)
		}
	}
}

func TestInspectSQLiteFallsBackToSudo(t *testing.T) {
	f := &fakeRun{reply: func(line string) audit.Result {
		if !strings.HasPrefix(line, "sudo -n ") {
			return audit.Result{Exit: 1, Stderr: []byte(`Error: unable to open database "/v/db.sqlite": unable to open database file`)}
		}
		if strings.Contains(line, "sqlite3") {
			return audit.Result{Stdout: []byte("wal\n3\n4096\nitems\nodd\"name\n")}
		}
		return audit.Result{}
	}}
	s := New(f)
	f.reply = func(prev func(string) audit.Result) func(string) audit.Result {
		n := 0
		return func(line string) audit.Result {
			if strings.HasPrefix(line, "sudo -n ") {
				n++
				if n == 2 {
					return audit.Result{Stdout: []byte("items\t12\nodd\"name\t0\n")}
				}
			}
			return prev(line)
		}
	}(f.reply)
	info, err := s.InspectSQLite(context.Background(), "srv", "/v/db.sqlite")
	if err != nil {
		t.Fatal(err)
	}
	if !info.Root || info.Journal != "wal" || info.Size != 3*4096 || len(info.Tables) != 2 ||
		info.Tables[0] != (SQLiteTable{Name: "items", Rows: 12}) {
		t.Fatalf("info = %+v", info)
	}
	if !strings.Contains(f.inputs[len(f.inputs)-1], `SELECT 'odd"name', count(*) FROM "odd""name"`) {
		t.Errorf("count = %s", f.inputs[len(f.inputs)-1])
	}
}

func TestInspectSQLiteErrors(t *testing.T) {
	for stderr, want := range map[string]apperr.Code{
		"Error: file is not a database": apperr.InvalidInput,
		"sudo: a password is required":  apperr.CmdNeedsRoot,
	} {
		f := &fakeRun{reply: func(line string) audit.Result {
			if strings.HasPrefix(line, "sudo") {
				return audit.Result{Exit: 1, Stderr: []byte(stderr)}
			}
			return audit.Result{Exit: 1, Stderr: []byte(stderr + "\nunable to open database file")}
		}}
		if _, err := New(f).InspectSQLite(context.Background(), "srv", "/a.db"); codeOf(err) != want {
			t.Errorf("%q: %v, want %s", stderr, err, want)
		}
	}
	if _, err := New(nil).InspectSQLite(context.Background(), "srv", "a.db"); codeOf(err) != apperr.InvalidInput {
		t.Errorf("relative path: %v", err)
	}
}

func TestParseSQLiteCell(t *testing.T) {
	for in, want := range map[string]SQLiteCell{
		"n":        {Kind: "null"},
		"":         {Kind: "null"},
		"b2048":    {Kind: "blob", Size: 2048},
		"tsome\tx": {Kind: "text", Text: "some\tx"},
		"Tlong":    {Kind: "text", Text: "long", Cut: true},
		"t":        {Kind: "text"},
		"v-3.5":    {Kind: "num", Text: "-3.5"},
	} {
		if got := parseSQLiteCell(in); got != want {
			t.Errorf("parseSQLiteCell(%q) = %+v, want %+v", in, got, want)
		}
	}
}

func TestSQLiteRowsSQL(t *testing.T) {
	got := sqliteRowsSQL(`we"ird`, []SQLiteColumn{{Name: "a b"}}, 200)
	for _, want := range []string{`typeof("a b")`, `FROM "we""ird" LIMIT 101 OFFSET 200;`, `substr("a b", 1, 200)`} {
		if !strings.Contains(got, want) {
			t.Errorf("rows SQL lacks %s:\n%s", want, got)
		}
	}
	if c := sqliteColumnsSQL(`x'); DROP TABLE y; --`); !strings.Contains(c, `pragma_table_info('x''); DROP TABLE y; --')`) {
		t.Errorf("columns SQL = %s", c)
	}
}

func TestRowsSQLiteRejects(t *testing.T) {
	s := New(nil)
	for _, c := range []struct {
		file, table string
		offset      int
	}{{"a.db", "t", 0}, {"/a.db", "", 0}, {"/a.db", "t", -1}} {
		if _, err := s.RowsSQLite(context.Background(), "srv", c.file, c.table, c.offset); codeOf(err) != apperr.InvalidInput {
			t.Errorf("%+v: %v", c, err)
		}
	}
}

// localSQLite runs the lines on a sqlite3 on this computer, so the SQL
// Davish builds is checked against a real SQLite.
type localSQLite struct{ file string }

func (l localSQLite) RunInput(ctx context.Context, _, _, _ string, stdin []byte) (audit.Result, error) {
	cmd := exec.CommandContext(ctx, "sqlite3", "-readonly", "-batch", "-noheader", "-separator", "\t", l.file) //nolint:gosec // a temp file of this test
	cmd.Stdin = bytes.NewReader(stdin)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	res := audit.Result{Stdout: out.Bytes(), Stderr: errb.Bytes()}
	if err != nil {
		res.Exit = 1
	}
	return res, nil
}

func TestRowsSQLiteOnRealSQLite(t *testing.T) {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("no sqlite3 here")
	}
	file := filepath.Join(t.TempDir(), "t.db")
	setup := `CREATE TABLE "odd ""t""" (id INTEGER PRIMARY KEY, name TEXT, note, data BLOB);
INSERT INTO "odd ""t""" VALUES (1, 'a' || char(9) || 'b' || char(10) || 'c' || char(31) || char(30), NULL, x'0102');
INSERT INTO "odd ""t""" VALUES (2, 'it''s', 3.5, NULL);
INSERT INTO "odd ""t""" VALUES (3, printf('%.300c', 'x'), -7, zeroblob(5000));
WITH RECURSIVE n(i) AS (SELECT 4 UNION ALL SELECT i + 1 FROM n WHERE i < 150) INSERT INTO "odd ""t""" (id) SELECT i FROM n;`
	if out, err := exec.Command("sqlite3", file, setup).CombinedOutput(); err != nil { //nolint:gosec // a temp file of this test
		t.Fatalf("setup: %v %s", err, out)
	}
	s := New(localSQLite{file})
	page, err := s.RowsSQLite(context.Background(), "srv", "/x.db", `odd "t"`, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Columns) != 4 || page.Columns[0] != (SQLiteColumn{Name: "id", Type: "INTEGER", PK: true}) || page.AllColumns != 4 {
		t.Fatalf("columns = %+v", page.Columns)
	}
	if len(page.Rows) != 100 || !page.More {
		t.Fatalf("rows = %d, more = %v", len(page.Rows), page.More)
	}
	r := page.Rows
	if r[0][1] != (SQLiteCell{Kind: "text", Text: "a\tb\nc  "}) || r[0][2].Kind != "null" || r[0][3] != (SQLiteCell{Kind: "blob", Size: 2}) {
		t.Errorf("row 1 = %+v", r[0])
	}
	if r[1][1].Text != "it's" || r[1][2] != (SQLiteCell{Kind: "num", Text: "3.5"}) {
		t.Errorf("row 2 = %+v", r[1])
	}
	if c := r[2][1]; !c.Cut || len(c.Text) != 200 || r[2][3].Size != 5000 {
		t.Errorf("row 3 = %+v", r[2])
	}
	last, err := s.RowsSQLite(context.Background(), "srv", "/x.db", `odd "t"`, 100)
	if err != nil || len(last.Rows) != 50 || last.More || last.Rows[0][0].Text != "101" {
		t.Errorf("page 2 = %d rows, more %v, %v", len(last.Rows), last.More, err)
	}
	if _, err := s.RowsSQLite(context.Background(), "srv", "/x.db", "nope", 0); codeOf(err) != apperr.NotFound {
		t.Errorf("missing table: %v", err)
	}
}
