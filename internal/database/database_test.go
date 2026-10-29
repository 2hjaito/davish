package database

import (
	"context"
	"strings"
	"testing"

	"davish/internal/audit"
)

type fakeRun struct {
	lines  []string
	inputs []string
	reply  func(line string) audit.Result
}

func (f *fakeRun) RunInput(_ context.Context, _, _, line string, stdin []byte) (audit.Result, error) {
	f.lines = append(f.lines, line)
	f.inputs = append(f.inputs, string(stdin))
	return f.reply(line), nil
}

func TestListFallsBackToSudo(t *testing.T) {
	f := &fakeRun{reply: func(line string) audit.Result {
		switch {
		case strings.HasPrefix(line, "mysql"):
			return audit.Result{Exit: 1, Stderr: []byte("ERROR 1698 (28000): Access denied for user 'dev'@'localhost'")}
		case strings.HasPrefix(line, "sudo -n mysql"):
			return audit.Result{Stdout: []byte("shop\t1048576\nblog\t0\n")}
		case strings.HasPrefix(line, "runuser"):
			return audit.Result{Exit: 1, Stderr: []byte("runuser: may not be used by non-root users")}
		default:
			return audit.Result{Exit: 1, Stderr: []byte("sudo: a password is required")}
		}
	}}
	s := New(f)
	es, err := s.List(context.Background(), "srv")
	if err != nil {
		t.Fatal(err)
	}
	if !es[0].Installed || len(es[0].DBs) != 2 || es[0].DBs[1].Name != "shop" || es[0].DBs[1].Size != 1048576 {
		t.Errorf("mysql = %+v", es[0])
	}
	if !es[1].NeedsRoot {
		t.Errorf("postgres = %+v", es[1])
	}
	// The way in that worked is remembered.
	f.lines, f.inputs = nil, nil
	_, _ = s.Create(context.Background(), "srv", MySQL, "app", "app")
	if len(f.lines) != 1 || !strings.HasPrefix(f.lines[0], "sudo -n mysql") {
		t.Errorf("create ran %v", f.lines)
	}
	if strings.Contains(f.lines[0], "IDENTIFIED") || !strings.Contains(f.inputs[0], "IDENTIFIED BY '") {
		t.Error("the password must go on stdin, not the command line")
	}
}

func TestMissing(t *testing.T) {
	s := New(&fakeRun{reply: func(string) audit.Result { return audit.Result{Exit: 127} }})
	es, _ := s.List(context.Background(), "srv")
	if es[0].Installed || es[1].Installed {
		t.Errorf("engines = %+v", es)
	}
}

func TestSQL(t *testing.T) {
	sql, err := CreateSQL(Postgres, "shop", "shop_app")
	if err != nil || sql != "CREATE USER \"shop_app\" WITH PASSWORD '••••••••';\nCREATE DATABASE \"shop\" OWNER \"shop_app\";\n" {
		t.Errorf("create = %q, %v", sql, err)
	}
	if _, err := CreateSQL(MySQL, "Shop", "x"); err == nil {
		t.Error("uppercase name accepted")
	}
	if d, _ := DropSQL(MySQL, "my-db"); d != "DROP DATABASE `my-db`;" {
		t.Errorf("drop = %s", d)
	}
	if _, err := DropSQL(MySQL, "a`; DROP"); err == nil {
		t.Error("quote accepted")
	}
	pw, _ := password()
	if len(pw) != 24 {
		t.Errorf("password %q", pw)
	}
}
