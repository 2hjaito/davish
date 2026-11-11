package database

import (
	"context"
	"strings"
	"testing"

	"davish/internal/apperr"
	"davish/internal/audit"
	"davish/internal/sudo"
)

type inputRunner struct {
	res   audit.Result
	lines []string
	ins   [][]byte
}

func (r *inputRunner) RunInput(_ context.Context, _, _, line string, in []byte) (audit.Result, error) {
	r.lines = append(r.lines, line)
	r.ins = append(r.ins, append([]byte(nil), in...))
	return r.res, nil
}

type fakeRoot struct {
	state string
	res   audit.Result
	err   error
	lines []string
	ins   [][]byte
}

func (f *fakeRoot) Run(_ context.Context, _, _, line string, in []byte) (audit.Result, error) {
	f.lines = append(f.lines, line)
	f.ins = append(f.ins, append([]byte(nil), in...))
	return f.res, f.err
}

func (f *fakeRoot) Prepare(_ context.Context, _, _, line string) (string, []byte, error) {
	return "SUDO[" + line + "]", []byte("pw\n"), f.err
}

func (f *fakeRoot) State(context.Context, string) (string, error) { return f.state, nil }

func TestExecReachesTheClientAsRoot(t *testing.T) {
	ctx := context.Background()
	run := &inputRunner{res: audit.Result{Exit: 1, Stderr: []byte("ERROR 1698 (28000): Access denied for user 'u'@'localhost'")}}
	root := &fakeRoot{res: audit.Result{Stdout: []byte("shop\t1024\n")}}
	s := New(run)
	s.UseRoot(root)
	es, err := s.List(ctx, "srv")
	if err != nil {
		t.Fatal(err)
	}
	if es[0].Kind != MySQL || len(es[0].DBs) != 1 || es[0].DBs[0].Name != "shop" {
		t.Fatalf("mysql = %+v", es[0])
	}
	// As the user first, never sudo -n; then as root with the SQL on stdin.
	for _, l := range run.lines {
		if strings.Contains(l, "sudo") {
			t.Errorf("ran sudo itself: %s", l)
		}
	}
	if root.lines[0] != line(nil, MySQL) || string(root.ins[0]) != listSQL[MySQL] {
		t.Errorf("root ran %q with %q", root.lines[0], root.ins[0])
	}
	if !strings.HasPrefix(root.lines[1], "runuser -u postgres -- psql ") {
		t.Errorf("postgres as root: %s", root.lines[1])
	}
}

func TestListOffersThePromptOnlyToSudoers(t *testing.T) {
	for state, ask := range map[string]bool{sudo.Password: true, sudo.Unlikely: false} {
		run := &inputRunner{res: audit.Result{Exit: 1, Stderr: []byte("Access denied")}}
		s := New(run)
		s.UseRoot(&fakeRoot{state: state, err: apperr.New(apperr.SudoPasswordRequired, "x")})
		es, err := s.List(context.Background(), "srv")
		if err != nil {
			t.Fatal(err)
		}
		if !es[0].NeedsRoot || es[0].AskSudo != ask {
			t.Errorf("%s: needsRoot %v ask %v", state, es[0].NeedsRoot, es[0].AskSudo)
		}
	}
}

func TestRootRestoreLine(t *testing.T) {
	got := rootRestoreLine("SUDO", "/home/u/db-backups/shop-1.sql.gz", true)
	want := "gzip -t -- /home/u/db-backups/shop-1.sql.gz && { head -n 1; gzip -dc -- /home/u/db-backups/shop-1.sql.gz; } | SUDO"
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
	if got := rootRestoreLine("SUDO", "/x/it's.sql", false); got != `cat -- '/x/it'\''s.sql' | SUDO` {
		t.Errorf("plain: %s", got)
	}
}

func TestDumpAsRootWritesAsTheUser(t *testing.T) {
	run := &inputRunner{res: audit.Result{Stdout: []byte("/home/u/db-backups/shop-x.sql.gz\n")}}
	s := New(run)
	s.UseRoot(&fakeRoot{})
	s.access["srv/"+MySQL] = viaRoot
	if _, err := s.Dump(context.Background(), "srv", MySQL, "shop"); err != nil {
		t.Fatal(err)
	}
	l := run.lines[0]
	if !strings.Contains(l, "SUDO[mysqldump --single-transaction --routines --triggers shop] > \"$HOME/db-backups/shop-") ||
		string(run.ins[0]) != "pw\n" {
		t.Errorf("dump line %s with %q", l, run.ins[0])
	}
}
