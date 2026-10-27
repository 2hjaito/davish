//go:build integration

package logs_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"davish/internal/apperr"
	"davish/internal/audit"
	"davish/internal/events"
	"davish/internal/logs"
	"davish/internal/testsshd"
)

// recorder collects logs events per stream.
type recorder struct {
	mu    sync.Mutex
	lines map[string][]string
	ends  map[string]logs.End
}

func (r *recorder) Emit(name string, payload any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	switch name {
	case events.LogsLine:
		p := payload.(logs.Lines)
		r.lines[p.StreamID] = append(r.lines[p.StreamID], p.Lines...)
	case events.LogsEnd:
		p := payload.(logs.End)
		r.ends[p.StreamID] = p
	}
}

func (r *recorder) waitFor(t *testing.T, what string, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		r.mu.Lock()
		done := ok()
		r.mu.Unlock()
		if done {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func setup(t *testing.T) (*logs.Service, *recorder, *audit.Log, string) {
	t.Helper()
	m, id := testsshd.Connect(t)
	rec := &recorder{lines: map[string][]string{}, ends: map[string]logs.End{}}
	log := audit.New(rec)
	s := logs.New(m, log, rec, testsshd.Quiet())
	t.Cleanup(s.Close)
	return s, rec, log, id
}

func contains(lines []string, s string) bool {
	for _, l := range lines {
		if strings.Contains(l, s) {
			return true
		}
	}
	return false
}

func TestFollowsContainerUntilStopped(t *testing.T) {
	s, rec, log, id := setup(t)
	sid, err := s.Start(context.Background(), id, logs.Source{Kind: logs.KindDocker, Target: "davish-web"})
	if err != nil {
		t.Fatal(err)
	}
	// The backlog, then new lines as the container writes them.
	rec.waitFor(t, "two batches of output", func() bool { return len(rec.lines[sid]) > 0 })
	first := len(rec.lines[sid])
	rec.waitFor(t, "new lines", func() bool { return len(rec.lines[sid]) > first })
	if !contains(rec.lines[sid], "served") {
		t.Errorf("lines = %q", rec.lines[sid])
	}
	s.Stop(sid)
	rec.waitFor(t, "end", func() bool { _, ok := rec.ends[sid]; return ok })
	if end := rec.ends[sid]; end.Error != nil {
		t.Errorf("stopped stream ended with %+v", end.Error)
	}
	if e := log.Entries(); len(e) != 1 || e[0].Command != "docker logs -f --tail 200 davish-web" {
		t.Errorf("audit = %+v", e)
	}
}

func TestFollowsCrashLoopStderr(t *testing.T) {
	s, rec, _, id := setup(t)
	sid, err := s.Start(context.Background(), id, logs.Source{Kind: logs.KindDocker, Target: "davish-crashloop"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Stop(sid) })
	rec.waitFor(t, "the error line", func() bool {
		return contains(rec.lines[sid], "Error: cannot open /etc/nginx/nginx.conf")
	})
}

func TestFollowsJournalAndFile(t *testing.T) {
	s, rec, _, id := setup(t)
	ctx := context.Background()
	j, err := s.Start(ctx, id, logs.Source{Kind: logs.KindJournal, Target: "davish-crash.service"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Stop(j) })
	rec.waitFor(t, "journal lines", func() bool { return contains(rec.lines[j], "cannot open /etc/nginx/nginx.conf") })

	f, err := s.Start(ctx, id, logs.Source{Kind: logs.KindFile, Target: "/etc/nginx/mime.types"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Stop(f) })
	rec.waitFor(t, "file lines", func() bool { return contains(rec.lines[f], "}") })
}

func TestUnreadableFileNeedsRoot(t *testing.T) {
	s, rec, _, id := setup(t)
	sid, err := s.Start(context.Background(), id, logs.Source{Kind: logs.KindFile, Target: "/etc/shadow"})
	if err != nil {
		t.Fatal(err)
	}
	rec.waitFor(t, "end", func() bool { _, ok := rec.ends[sid]; return ok })
	end := rec.ends[sid]
	if end.Error == nil || end.Error.Code != apperr.CmdNeedsRoot || end.Sudo != "sudo tail -F -n 200 -- /etc/shadow" {
		t.Errorf("end = %+v", end)
	}
}
