//go:build integration

package audit_test

import (
	"context"
	"strings"
	"testing"

	"davish/internal/audit"
	"davish/internal/shellx"
	"davish/internal/testsshd"
)

type discard struct{}

func (discard) Emit(string, any) {}

func TestRunnerRecordsEveryCommand(t *testing.T) {
	m, id := testsshd.Connect(t)
	log := audit.New(discard{})
	r := audit.NewRunner(m, log)
	ctx := context.Background()

	res, err := r.Run(ctx, id, "test", shellx.Cmd("echo", "it's here"))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(res.Stdout)); got != "it's here" || res.Exit != 0 {
		t.Errorf("echo = %q exit %d", got, res.Exit)
	}

	res, err = r.Run(ctx, id, "test", shellx.Cmd("ls", "/no/such/dir"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Exit == 0 || !strings.Contains(string(res.Stderr), "No such file") {
		t.Errorf("ls = exit %d, stderr %q", res.Exit, res.Stderr)
	}

	got := log.Entries()
	if len(got) != 2 {
		t.Fatalf("entries = %d, want 2", len(got))
	}
	if got[0].Command != `echo 'it'\''s here'` || got[0].Exit != 0 || got[0].ServerID != id || got[0].App != "test" {
		t.Errorf("first entry = %+v", got[0])
	}
	if got[1].Command != "ls /no/such/dir" || got[1].Exit == 0 {
		t.Errorf("second entry = %+v", got[1])
	}
}
