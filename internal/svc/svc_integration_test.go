//go:build integration

package svc_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"davish/internal/apperr"
	"davish/internal/audit"
	"davish/internal/svc"
	"davish/internal/testsshd"
)

type discard struct{}

func (discard) Emit(string, any) {}

func setup(t *testing.T) (*svc.Service, *audit.Log, string) {
	t.Helper()
	m, id := testsshd.Connect(t)
	log := audit.New(discard{})
	return svc.New(audit.NewRunner(m, log)), log, id
}

func code(err error) apperr.Code {
	var ae *apperr.Error
	if errors.As(err, &ae) {
		return ae.Code
	}
	return ""
}

func find(t *testing.T, s *svc.Service, id, name string) svc.Unit {
	t.Helper()
	units, err := s.List(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	for _, u := range units {
		if u.Name == name {
			return u
		}
	}
	t.Fatalf("%s not listed", name)
	return svc.Unit{}
}

func TestList(t *testing.T) {
	s, log, id := setup(t)
	if u := find(t, s, id, "davish-ok.service"); u.Active != "active" || u.Sub != "running" {
		t.Errorf("davish-ok = %+v", u)
	}
	if u := find(t, s, id, "davish-crash.service"); u.Active != "failed" {
		t.Errorf("davish-crash = %+v", u)
	}
	got := log.Entries()
	if len(got) == 0 || !strings.HasPrefix(got[0].Command, "systemctl list-units") {
		t.Errorf("audit = %+v", got)
	}
}

func TestActRestartsAllowedUnit(t *testing.T) {
	s, _, id := setup(t)
	ctx := context.Background()
	t.Cleanup(func() { _ = s.Act(ctx, id, "davish-ok.service", svc.ActionStart) })

	before, err := s.Status(ctx, id, "davish-ok.service")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Act(ctx, id, "davish-ok.service", svc.ActionRestart); err != nil {
		t.Fatal(err)
	}
	after, err := s.Status(ctx, id, "davish-ok.service")
	if err != nil {
		t.Fatal(err)
	}
	if after.Active != "active" || after.MainPID == 0 || after.MainPID == before.MainPID {
		t.Errorf("after restart = %+v (before PID %d)", after, before.MainPID)
	}

	if err := s.Act(ctx, id, "davish-ok.service", svc.ActionStop); err != nil {
		t.Fatal(err)
	}
	if u := find(t, s, id, "davish-ok.service"); u.Active != "inactive" {
		t.Errorf("after stop = %+v", u)
	}
}

func TestActNeedsRoot(t *testing.T) {
	s, _, id := setup(t)
	err := s.Act(context.Background(), id, "nginx.service", svc.ActionStop)
	if code(err) != apperr.CmdNeedsRoot {
		t.Fatalf("stop nginx = %v, want %s", err, apperr.CmdNeedsRoot)
	}
	if u := find(t, s, id, "nginx.service"); u.Active != "active" {
		t.Errorf("nginx should still run: %+v", u)
	}
}

func TestActFailures(t *testing.T) {
	s, _, id := setup(t)
	ctx := context.Background()
	// A simple service counts as started once its process runs; that it
	// died right after only shows in its state.
	if err := s.Act(ctx, id, "davish-crash.service", svc.ActionStart); err != nil {
		t.Errorf("start crash = %v", err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for find(t, s, id, "davish-crash.service").Active != "failed" {
		if time.Now().After(deadline) {
			t.Fatal("davish-crash did not fail after start")
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err := s.Act(ctx, id, "davish-nope.service", svc.ActionStart); code(err) != apperr.NotFound {
		t.Errorf("start missing = %v, want %s", err, apperr.NotFound)
	}
}

func TestStatusAndJournal(t *testing.T) {
	s, _, id := setup(t)
	ctx := context.Background()
	st, err := s.Status(ctx, id, "davish-crash.service")
	if err != nil {
		t.Fatal(err)
	}
	if st.Active != "failed" || st.Result != "exit-code" || st.Enabled != "enabled" ||
		st.File != "/etc/systemd/system/davish-crash.service" || st.ActiveFor != 0 {
		t.Errorf("status = %+v", st)
	}

	j, err := s.Journal(ctx, id, "davish-crash.service")
	if err != nil {
		t.Fatal(err)
	}
	if j.Restricted || !strings.Contains(strings.Join(j.Lines, "\n"), "cannot open /etc/nginx/nginx.conf") {
		t.Errorf("journal = %+v", j)
	}
	if j.Sudo != "sudo journalctl -u davish-crash.service -n 200 --no-pager -o short-iso" {
		t.Errorf("sudo = %q", j.Sudo)
	}
}
