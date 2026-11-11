//go:build integration

package logs_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"davish/internal/apperr"
	"davish/internal/audit"
	"davish/internal/events"
	"davish/internal/logs"
	"davish/internal/sudo"
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
	st, err := s.Start(context.Background(), id, logs.Source{Kind: logs.KindDocker, Target: "davish-web"})
	if err != nil {
		t.Fatal(err)
	}
	sid := st.StreamID
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
	st, err := s.Start(context.Background(), id, logs.Source{Kind: logs.KindDocker, Target: "davish-crashloop"})
	if err != nil {
		t.Fatal(err)
	}
	sid := st.StreamID
	t.Cleanup(func() { s.Stop(sid) })
	rec.waitFor(t, "the error line", func() bool {
		return contains(rec.lines[sid], "Error: cannot open /etc/nginx/nginx.conf")
	})
}

func TestFollowsJournalAndFile(t *testing.T) {
	s, rec, _, id := setup(t)
	ctx := context.Background()
	js, err := s.Start(ctx, id, logs.Source{Kind: logs.KindJournal, Target: "davish-crash.service"})
	if err != nil {
		t.Fatal(err)
	}
	j := js.StreamID
	if want := "sudo journalctl -u davish-crash.service -f -n 200 -o short-iso --no-pager"; js.Sudo != want {
		t.Errorf("sudo = %q, want %q", js.Sudo, want)
	}
	t.Cleanup(func() { s.Stop(j) })
	rec.waitFor(t, "journal lines", func() bool { return contains(rec.lines[j], "cannot open /etc/nginx/nginx.conf") })

	fs, err := s.Start(ctx, id, logs.Source{Kind: logs.KindFile, Target: "/etc/nginx/mime.types"})
	if err != nil {
		t.Fatal(err)
	}
	f := fs.StreamID
	t.Cleanup(func() { s.Stop(f) })
	rec.waitFor(t, "file lines", func() bool { return contains(rec.lines[f], "}") })
}

func TestUnreadableFileNeedsRoot(t *testing.T) {
	s, rec, _, id := setup(t)
	st, err := s.Start(context.Background(), id, logs.Source{Kind: logs.KindFile, Target: "/etc/shadow"})
	if err != nil {
		t.Fatal(err)
	}
	sid := st.StreamID
	rec.waitFor(t, "end", func() bool { _, ok := rec.ends[sid]; return ok })
	end := rec.ends[sid]
	if end.Error == nil || end.Error.Code != apperr.CmdNeedsRoot || end.Sudo != "sudo tail -F -n 200 -- /etc/shadow" {
		t.Errorf("end = %+v", end)
	}
}

// sudoer follows a file only root may read through sudo once the password
// is given, and the password never reaches the audit log (ADR 0018).
func TestSudoerFollowsRootFileThroughSudo(t *testing.T) {
	ctx := context.Background()
	m, id := testsshd.ConnectAs(t, testsshd.Sudoer)
	rec := &recorder{lines: map[string][]string{}, ends: map[string]logs.End{}}
	log := audit.New(rec)
	k := sudo.New(audit.NewRunner(m, log))
	s := logs.New(m, log, rec, testsshd.Quiet())
	s.UseRoot(k)
	t.Cleanup(s.Close)

	src := logs.Source{Kind: logs.KindFile, Target: "/root/davish-sudo.log"}
	if _, err := s.StartAsRoot(ctx, id, src); codeOf(err) != apperr.SudoPasswordRequired {
		t.Fatalf("before the password: %v", err)
	}
	if err := k.SetPassword(ctx, id, []byte(testsshd.SudoPassword)); err != nil {
		t.Fatal(err)
	}
	if _, err := k.Run(ctx, id, "test", "printf 'davish-root-line\\n' > /root/davish-sudo.log && chmod 600 /root/davish-sudo.log", nil); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = k.Run(ctx, id, "test", "rm -f /root/davish-sudo.log", nil) })

	plain, err := s.Start(ctx, id, src)
	if err != nil {
		t.Fatal(err)
	}
	rec.waitFor(t, "plain end", func() bool { _, ok := rec.ends[plain.StreamID]; return ok })
	if e := rec.ends[plain.StreamID]; e.Error == nil || e.Error.Code != apperr.CmdNeedsRoot {
		t.Fatalf("plain end = %+v", e)
	}

	st, err := s.StartAsRoot(ctx, id, src)
	if err != nil {
		t.Fatal(err)
	}
	rec.waitFor(t, "the root-only line", func() bool { return contains(rec.lines[st.StreamID], "davish-root-line") })
	s.Stop(st.StreamID)
	rec.waitFor(t, "end", func() bool { _, ok := rec.ends[st.StreamID]; return ok })
	if e := rec.ends[st.StreamID]; e.Error != nil {
		t.Errorf("end after stop = %+v", e)
	}
	for _, e := range log.Entries() {
		if strings.Contains(e.Command, testsshd.SudoPassword) {
			t.Fatalf("password in the audit log: %s", e.Command)
		}
	}
	for _, l := range rec.lines[st.StreamID] {
		if strings.Contains(l, testsshd.SudoPassword) {
			t.Fatalf("password in the stream: %s", l)
		}
	}
}

// refusing plays sudo turning the password down on a stream: it says so
// and exits 1, which alone would read as a failed command.
type refusing struct{ forgot string }

func (r *refusing) Prepare(context.Context, string, string, string) (string, []byte, error) {
	return "echo 'sudo: 1 incorrect password attempt' >&2; exit 1", nil, nil
}

func (r *refusing) Forget(serverID string) { r.forgot = serverID }

func TestRefusedPasswordEndsTheStreamAndIsForgotten(t *testing.T) {
	s, rec, _, id := setup(t)
	r := &refusing{}
	s.UseRoot(r)
	st, err := s.StartAsRoot(context.Background(), id, logs.Source{Kind: logs.KindJournal})
	if err != nil {
		t.Fatal(err)
	}
	rec.waitFor(t, "end", func() bool { _, ok := rec.ends[st.StreamID]; return ok })
	if e := rec.ends[st.StreamID]; e.Error == nil || e.Error.Code != apperr.SudoPasswordWrong {
		t.Fatalf("end = %+v", e)
	}
	if r.forgot != id {
		t.Errorf("password not forgotten: %q", r.forgot)
	}
}

func codeOf(err error) apperr.Code {
	var e *apperr.Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}
