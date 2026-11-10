//go:build integration

package sudo_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"davish/internal/apperr"
	"davish/internal/audit"
	"davish/internal/sudo"
	"davish/internal/testsshd"
)

type discard struct{}

func (discard) Emit(string, any) {}

func code(err error) apperr.Code {
	var e *apperr.Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}

func TestSudoer(t *testing.T) {
	ctx := context.Background()
	m, id := testsshd.ConnectAs(t, testsshd.Sudoer)
	log := audit.New(discard{})
	k := sudo.New(audit.NewRunner(m, log))

	if st, err := k.State(ctx, id); err != nil || st != sudo.Password {
		t.Fatalf("state = %s, %v", st, err)
	}
	if _, err := k.Run(ctx, id, "test", "systemctl restart davish-ok.service", nil); code(err) != apperr.SudoPasswordRequired {
		t.Fatalf("before the password: %v", err)
	}
	if err := k.SetPassword(ctx, id, []byte("wrong")); code(err) != apperr.SudoPasswordWrong {
		t.Fatalf("wrong password: %v", err)
	}
	if err := k.SetPassword(ctx, id, []byte(testsshd.SudoPassword)); err != nil {
		t.Fatal(err)
	}
	res, err := k.Run(ctx, id, "test", "systemctl restart davish-ok.service", nil)
	if err != nil || res.Exit != 0 {
		t.Fatalf("restart: %v, exit %d, %s", err, res.Exit, res.Stderr)
	}
	// sudo remembers a password for a while; it must still never reach
	// the command's stdin.
	res, err = k.Run(ctx, id, "test", "cat; id -u", []byte("only this\n"))
	if err != nil || string(res.Stdout) != "only this\n0\n" {
		t.Fatalf("stdin = %q, %v", res.Stdout, err)
	}
	for _, e := range log.Entries() {
		if strings.Contains(e.Command, testsshd.SudoPassword) {
			t.Fatalf("password in the audit log: %s", e.Command)
		}
	}
	k.Forget(id)
	if _, err := k.Run(ctx, id, "test", "true", nil); code(err) != apperr.SudoPasswordRequired {
		t.Fatalf("after forget: %v", err)
	}
}

// sudo asks a user who may not use it for a password all the same, and
// only says so once given one: the UI asks, then offers a terminal.
func TestUserWithoutSudo(t *testing.T) {
	ctx := context.Background()
	m, id := testsshd.Connect(t)
	k := sudo.New(audit.NewRunner(m, audit.New(discard{})))
	if _, err := k.Run(ctx, id, "test", "true", nil); code(err) != apperr.SudoPasswordRequired {
		t.Fatalf("run: %v", err)
	}
	if err := k.SetPassword(ctx, id, []byte("anything")); code(err) != apperr.CmdNeedsRoot {
		t.Fatalf("set password: %v", err)
	}
}
