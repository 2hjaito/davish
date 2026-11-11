//go:build integration

package database

import (
	"context"
	"strings"
	"testing"

	"davish/internal/audit"
	"davish/internal/shellx"
	"davish/internal/sudo"
	"davish/internal/testsshd"
)

type quiet struct{}

func (quiet) Emit(string, any) {}

// The restore trick on a real sudo: the password reaches sudo, and only
// the dump reaches the client (here cat, writing a root-only file).
func TestRootRestoreLineFeedsOnlyTheDump(t *testing.T) {
	ctx := context.Background()
	m, id := testsshd.ConnectAs(t, testsshd.Sudoer)
	run := audit.NewRunner(m, audit.New(quiet{}))
	k := sudo.New(run)
	if err := k.SetPassword(ctx, id, []byte(testsshd.SudoPassword)); err != nil {
		t.Fatal(err)
	}
	dump := "/tmp/davish-restore-test.sql"
	out := "/root/davish-restore-test.out"
	testsshd.Run(t, m, id, "printf 'CREATE TABLE t (x int);\\nINSERT INTO t VALUES (1);\\n' > "+dump+" && gzip -f "+dump)
	t.Cleanup(func() {
		testsshd.Run(t, m, id, "rm -f "+dump+".gz")
		_, _ = k.Run(ctx, id, "test", "rm -f "+out, nil)
	})

	client, pw, err := k.Prepare(ctx, id, "test", shellx.Cmd("sh", "-c", "cat > "+out).String())
	if err != nil || pw == nil {
		t.Fatalf("prepare: %v, password %v", err, pw != nil)
	}
	line := rootRestoreLine(client, dump+".gz", true)
	if strings.Contains(line, testsshd.SudoPassword) {
		t.Fatal("password on the command line")
	}
	res, err := run.RunInput(ctx, id, "test", line, pw)
	if err != nil || res.Exit != 0 {
		t.Fatalf("restore: %v %s", err, res.Stderr)
	}
	got, err := k.Run(ctx, id, "test", "cat "+out+"; stat -c %U "+out, nil)
	if err != nil {
		t.Fatal(err)
	}
	if want := "CREATE TABLE t (x int);\nINSERT INTO t VALUES (1);\nroot\n"; string(got.Stdout) != want {
		t.Errorf("client got %q, want %q", got.Stdout, want)
	}
}
