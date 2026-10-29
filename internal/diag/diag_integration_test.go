//go:build integration

package diag_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"davish/internal/apperr"
	"davish/internal/audit"
	"davish/internal/diag"
	"davish/internal/testsshd"
)

type discard struct{}

func (discard) Emit(string, any) {}

// A ~/.bashrc that never returns for non-interactive shells: the runner
// must give up at its deadline instead of waiting with the server, and
// Diagnose must name the shell.
func TestHungShellIsReported(t *testing.T) {
	m, id := testsshd.Connect(t)
	testsshd.Run(t, m, id, `cp ~/.bashrc ~/.bashrc.davish-bak; { echo 'sleep 600'; cat ~/.bashrc.davish-bak; } > ~/.bashrc`)
	// Every way in goes through the shell, sftp included; put the file
	// back from outside, through the container.
	t.Cleanup(func() {
		testsshd.Exec(t, "sh", "-c", "mv /home/tester/.bashrc.davish-bak /home/tester/.bashrc; pkill -f '^sleep 600$' || true")
	})
	run := audit.NewRunner(m, audit.New(discard{}))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	start := time.Now()
	_, err := run.RunLine(ctx, id, "test", "echo hi")
	var ae *apperr.Error
	if !errors.As(err, &ae) || ae.Code != apperr.CmdTimeout || time.Since(start) > 5*time.Second {
		t.Fatalf("err = %v after %s; want CMD_TIMEOUT at the deadline", err, time.Since(start))
	}

	// sftp goes through the same shell: it must give up too.
	if _, err := m.SFTP(id); !errors.As(err, &ae) || ae.Code != apperr.CmdTimeout {
		t.Errorf("sftp: %v; want CMD_TIMEOUT", err)
	}

	r := diag.New(run, m).Diagnose(context.Background(), id)
	if r.Verdict != "shell-hangs" {
		t.Fatalf("report = %s", diag.Summary(r))
	}
}
