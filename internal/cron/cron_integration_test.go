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
	"davish/internal/sudo"
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
	// A second ticking over between showing and confirming must not
	// rename the backup.
	time.Sleep(1100 * time.Millisecond)
	if err := s.Apply(ctx, id, c, p); err != nil {
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
	if err := s.Apply(ctx, id, cron.Change{Op: cron.OpDelete, ID: mine[0].ID}, stale); code(err) != apperr.FSStale {
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

func rootJobs(tab cron.Table) []cron.Job {
	var jobs []cron.Job
	for _, j := range tab.Jobs {
		if j.Source == cron.SourceRoot {
			jobs = append(jobs, j)
		}
	}
	return jobs
}

// sudoer edits root's crontab through sudo once the password is given;
// the job goes in on stdin and the old crontab is backed up in /root
// (ADR 0018).
func TestSudoerEditsRootCrontab(t *testing.T) {
	ctx := context.Background()
	m, id := testsshd.ConnectAs(t, testsshd.Sudoer)
	run := audit.NewRunner(m, audit.New(discard{}))
	k := sudo.New(run)
	s := cron.New(run, m)
	s.UseRoot(k)
	clean := func() {
		if _, err := k.Run(ctx, id, "test", "crontab -u root -r 2>/dev/null; rm -f /root/.crontab.bak-*; true", nil); err != nil {
			t.Error(err)
		}
	}

	tab, err := s.List(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if !tab.RootLocked || tab.Root || len(rootJobs(tab)) != 0 {
		t.Fatalf("before the password: locked %v, root %v, jobs %+v", tab.RootLocked, tab.Root, rootJobs(tab))
	}
	add := cron.Change{Source: cron.SourceRoot, Op: cron.OpAdd, Schedule: "17 4 * * *", Command: "/usr/bin/true davish-root-job"}
	if _, err := s.Plan(ctx, id, add); code(err) != apperr.SudoPasswordRequired {
		t.Fatalf("plan before the password: %v", err)
	}
	if err := k.SetPassword(ctx, id, []byte(testsshd.SudoPassword)); err != nil {
		t.Fatal(err)
	}
	clean()
	t.Cleanup(clean)

	p := apply(t, s, id, add)
	if !strings.HasPrefix(p.Sudo, "sudo sh -c ") || p.Backup != "" {
		t.Errorf("first plan = %+v", p)
	}
	tab, err = s.List(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	jobs := rootJobs(tab)
	if !tab.Root || tab.RootLocked || len(jobs) != 1 || jobs[0].Command != add.Command || !jobs[0].Editable {
		t.Fatalf("after adding: root %v, jobs %+v", tab.Root, jobs)
	}
	for _, j := range tab.Jobs {
		if j.Source == cron.SourceUser && j.Command == add.Command {
			t.Fatal("root's job landed in sudoer's own crontab")
		}
	}

	p = apply(t, s, id, cron.Change{Source: cron.SourceRoot, Op: cron.OpDelete, ID: jobs[0].ID})
	if !strings.HasPrefix(p.Backup, "/root/.crontab.bak-") {
		t.Errorf("backup = %q", p.Backup)
	}
	res, err := k.Run(ctx, id, "test", "cat "+p.Backup, nil)
	if err != nil || !strings.Contains(string(res.Stdout), "davish-root-job") {
		t.Errorf("backup holds %q, %v", res.Stdout, err)
	}
	tab, _ = s.List(ctx, id)
	if len(rootJobs(tab)) != 0 {
		t.Errorf("after deleting: %+v", rootJobs(tab))
	}
}

// tester may not use sudo: root's crontab is neither shown nor editable.
func TestNoSudoKeepsRootCrontabOut(t *testing.T) {
	ctx := context.Background()
	s, m, id := setup(t)
	s.UseRoot(sudo.New(audit.NewRunner(m, audit.New(discard{}))))
	tab, err := s.List(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if tab.Root || tab.RootLocked || len(rootJobs(tab)) != 0 {
		t.Fatalf("root %v, locked %v, jobs %+v", tab.Root, tab.RootLocked, rootJobs(tab))
	}
	add := cron.Change{Source: cron.SourceRoot, Op: cron.OpAdd, Schedule: "* * * * *", Command: "id"}
	if _, err := s.Plan(ctx, id, add); code(err) != apperr.CmdNeedsRoot {
		t.Fatalf("plan for root's crontab: %v", err)
	}
	if err := s.Apply(ctx, id, add, cron.Plan{Command: "crontab -u root -"}); code(err) != apperr.CmdNeedsRoot {
		t.Fatalf("apply for root's crontab: %v", err)
	}
	if _, err := s.Plan(ctx, id, cron.Change{Source: "/etc/crontab", Op: cron.OpAdd, Schedule: "* * * * *", Command: "id"}); code(err) != apperr.InvalidInput {
		t.Fatalf("plan for a system file: %v", err)
	}
}
