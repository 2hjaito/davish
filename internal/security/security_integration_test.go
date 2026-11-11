//go:build integration

package security_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"davish/internal/apperr"
	"davish/internal/audit"
	"davish/internal/security"
	"davish/internal/sudo"
	"davish/internal/testsshd"
)

type discard struct{}

func (discard) Emit(string, any) {}

func TestReportSeesOwnLogin(t *testing.T) {
	m, id := testsshd.Connect(t)
	s := security.New(audit.NewRunner(m, audit.New(discard{})))
	r, err := s.Report(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if r.Source != "journal" || len(r.Accepted) == 0 || r.Accepted[0].User != testsshd.User {
		t.Fatalf("report = %+v", r)
	}
	if !r.LastbDenied || r.Root {
		t.Errorf("lastb denied = %v, root = %v", r.LastbDenied, r.Root)
	}
}

func code(err error) apperr.Code {
	var e *apperr.Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}

// sudoer reads what needs root, and runs an action, through sudo once the
// password is given (ADR 0018). The test server has no ufw, so the action
// reaching root shows as CMD_MISSING.
func TestSudoerReadsAndActsThroughSudo(t *testing.T) {
	ctx := context.Background()
	m, id := testsshd.ConnectAs(t, testsshd.Sudoer)
	run := audit.NewRunner(m, audit.New(discard{}))
	k := sudo.New(run)
	s := security.New(run)
	s.UseRoot(k)

	r, err := s.Report(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if r.Root || r.AsRoot || !r.LastbDenied || !r.AskSudo || !r.CanSudo || !strings.HasPrefix(r.SudoRead, "sudo sh -c ") {
		t.Fatalf("before the password: root %v asRoot %v lastbDenied %v ask %v canSudo %v", r.Root, r.AsRoot, r.LastbDenied, r.AskSudo, r.CanSudo)
	}
	plan, err := security.PlanAction(security.Block, "203.0.113.9", false)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Apply(ctx, id, security.Block, "203.0.113.9", plan.Command); code(err) != apperr.SudoPasswordRequired {
		t.Fatalf("block before the password: %v", err)
	}
	if err := k.SetPassword(ctx, id, []byte(testsshd.SudoPassword)); err != nil {
		t.Fatal(err)
	}
	if r, err = s.Report(ctx, id); err != nil {
		t.Fatal(err)
	}
	if r.Root || !r.AsRoot || r.LastbDenied || r.AskSudo {
		t.Fatalf("after the password: root %v asRoot %v lastbDenied %v ask %v", r.Root, r.AsRoot, r.LastbDenied, r.AskSudo)
	}
	if err := s.Apply(ctx, id, security.Block, "203.0.113.9", plan.Command); code(err) != apperr.CmdMissing {
		t.Fatalf("block through sudo: %v", err)
	}
}

// tester may not use sudo: nothing is offered and actions stay plain.
func TestNoSudoOffersNothing(t *testing.T) {
	m, id := testsshd.Connect(t)
	run := audit.NewRunner(m, audit.New(discard{}))
	s := security.New(run)
	s.UseRoot(sudo.New(run))
	r, err := s.Report(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if r.AskSudo || r.CanSudo || r.AsRoot {
		t.Fatalf("ask %v canSudo %v asRoot %v", r.AskSudo, r.CanSudo, r.AsRoot)
	}
}
