//go:build integration

package cron_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"davish/internal/apperr"
	"davish/internal/audit"
	"davish/internal/conn"
	"davish/internal/cron"
	"davish/internal/testsshd"
)

type discard struct{}

func (discard) Emit(string, any) {}

func setup(t *testing.T) (*cron.Service, *conn.Manager, string) {
	t.Helper()
	m, id := testsshd.Connect(t)
	s := cron.New(audit.NewRunner(m, audit.New(discard{})), m)
	testsshd.Run(t, m, id, "crontab -r 2>/dev/null; rm -f ~/.crontab.bak-*; true")
	t.Cleanup(func() { testsshd.Run(t, m, id, "crontab -r 2>/dev/null; rm -f ~/.crontab.bak-*; true") })
	return s, m, id
}

func code(err error) apperr.Code {
	var ae *apperr.Error
	if errors.As(err, &ae) {
		return ae.Code
	}
	return ""
}

func userJobs(t *testing.T, s *cron.Service, id string) (cron.Table, []cron.Job) {
	t.Helper()
	tab, err := s.List(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	var mine []cron.Job
	for _, j := range tab.Jobs {
		if j.Source == cron.SourceUser {
			mine = append(mine, j)
		}
	}
	return tab, mine
}

func apply(t *testing.T, s *cron.Service, id string, c cron.Change) cron.Plan {
	t.Helper()
	ctx := context.Background()
	p, err := s.Plan(ctx, id, c)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Apply(ctx, id, c, p.Version); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestListsSystemCrontabs(t *testing.T) {
	s, _, id := setup(t)
	tab, mine := userJobs(t, s, id)
	if len(mine) != 0 {
		t.Errorf("user jobs before adding any: %+v", mine)
	}
	sources := map[string]int{}
	for _, j := range tab.Jobs {
		sources[j.Source]++
		if j.Editable {
			t.Errorf("system job editable: %+v", j)
		}
	}
	if sources["/etc/crontab"] != 4 || sources["/etc/cron.d/e2scrub_all"] != 2 {
		t.Errorf("sources = %v", sources)
	}
	if tab.Zone == "" {
		t.Error("no zone")
	}
}

// The M8 acceptance: add a job and see its next run.
func TestAddEditToggleDelete(t *testing.T) {
	s, m, id := setup(t)
	ctx := context.Background()

	p := apply(t, s, id, cron.Change{Op: cron.OpAdd, Schedule: "*/5 * * * *", Command: "echo davish >> /tmp/davish-cron"})
	if p.Command != "crontab -" || p.Backup != "" || p.After != "*/5 * * * * echo davish >> /tmp/davish-cron" {
		t.Errorf("first add plan = %+v", p)
	}
	_, mine := userJobs(t, s, id)
	if len(mine) != 1 || !mine[0].Enabled || !mine[0].Editable || len(mine[0].Next) != 3 {
		t.Fatalf("after add: %+v", mine)
	}
	next := time.Unix(mine[0].Next[0], 0)
	if next.Minute()%5 != 0 || next.Second() != 0 || time.Until(next) > 5*time.Minute || time.Until(next) < -time.Minute {
		t.Errorf("next run %s is not the next multiple of five minutes", next)
	}

	p = apply(t, s, id, cron.Change{Op: cron.OpDisable, ID: mine[0].ID})
	if !strings.HasPrefix(p.Command, "crontab -l > ") || !strings.HasSuffix(p.Command, " && crontab -") || p.Backup == "" {
		t.Errorf("disable plan = %+v", p)
	}
	if got := testsshd.Run(t, m, id, "cat "+p.Backup); got != "*/5 * * * * echo davish >> /tmp/davish-cron" {
		t.Errorf("backup = %q", got)
	}
	_, mine = userJobs(t, s, id)
	if len(mine) != 1 || mine[0].Enabled {
		t.Fatalf("after disable: %+v", mine)
	}

	apply(t, s, id, cron.Change{Op: cron.OpEnable, ID: mine[0].ID})
	_, mine = userJobs(t, s, id)
	apply(t, s, id, cron.Change{Op: cron.OpEdit, ID: mine[0].ID, Schedule: "@daily", Command: "echo edited"})
	if got := testsshd.Run(t, m, id, "crontab -l"); got != "@daily echo edited" {
		t.Errorf("crontab after edit = %q", got)
	}

	// A plan made before someone else changed the crontab is refused.
	_, mine = userJobs(t, s, id)
	stale, err := s.Plan(ctx, id, cron.Change{Op: cron.OpDelete, ID: mine[0].ID})
	if err != nil {
		t.Fatal(err)
	}
	testsshd.Run(t, m, id, "(crontab -l; echo '0 1 * * * echo other') | crontab -")
	if err := s.Apply(ctx, id, cron.Change{Op: cron.OpDelete, ID: mine[0].ID}, stale.Version); code(err) != apperr.FSStale {
		t.Errorf("stale apply = %v", err)
	}

	apply(t, s, id, cron.Change{Op: cron.OpDelete, ID: mine[0].ID})
	if got := testsshd.Run(t, m, id, "crontab -l"); got != "0 1 * * * echo other" {
		t.Errorf("crontab after delete = %q", got)
	}
}

func TestRefusesBadJobs(t *testing.T) {
	s, _, id := setup(t)
	for _, c := range []cron.Change{
		{Op: cron.OpAdd, Schedule: "every day", Command: "x"},
		{Op: cron.OpAdd, Schedule: "* * * * *", Command: "a\n* * * * * b"},
		{Op: cron.OpDelete, ID: "user:1:nope"},
	} {
		if _, err := s.Plan(context.Background(), id, c); err == nil {
			t.Errorf("Plan(%+v) accepted", c)
		}
	}
}

func TestRuns(t *testing.T) {
	s, m, id := setup(t)
	// cron itself writes these lines; logger writes one the same way.
	testsshd.Run(t, m, id, "logger -t CRON '(tester) CMD (echo davish-run)'")
	deadline := time.Now().Add(5 * time.Second)
	for {
		r, err := s.Runs(context.Background(), id, "echo davish-run")
		if err != nil {
			t.Fatal(err)
		}
		if len(r.Lines) > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("no run lines: %+v", r)
		}
		time.Sleep(200 * time.Millisecond)
	}
}
