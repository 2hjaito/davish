package database

import (
	"context"
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
