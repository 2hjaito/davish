package sysmon

import (
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"davish/internal/events"
	"davish/internal/platform"
)

// fakeClients answers with an empty client: the paths under test decide by
// OS and never open a session.
type fakeClients struct{ os platform.OS }

func (f fakeClients) Client(string) (*ssh.Client, error) { return &ssh.Client{}, nil }
func (f fakeClients) Platform(string) platform.OS        { return f.os }

type recorder struct {
	mu     sync.Mutex
	status []Status
}

func (r *recorder) Emit(name string, payload any) {
	if name != events.SysmonStatus {
		return
	}
	r.mu.Lock()
	r.status = append(r.status, payload.(Status))
	r.mu.Unlock()
}

func (r *recorder) all() []Status {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Status(nil), r.status...)
}

func newTestService(os platform.OS, rec *recorder) *Service {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return New(fakeClients{os}, rec, log, Options{RetryEvery: time.Millisecond})
}

func waitStatus(t *testing.T, rec *recorder, n int) []Status {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if got := rec.all(); len(got) >= n {
			return got
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("expected %d status events, got %v", n, rec.all())
	return nil
}

func TestUnsupportedServerReportsInsteadOfWaiting(t *testing.T) {
	for _, os := range []platform.OS{platform.Darwin, platform.Windows} {
		t.Run(string(os), func(t *testing.T) {
			rec := &recorder{}
			s := newTestService(os, rec)
			defer s.Close()
			s.Watch("srv")
			got := waitStatus(t, rec, 1)[0]
			want := Status{ServerID: "srv", State: StateUnsupported, OS: os}
			if got != want {
				t.Fatalf("status = %+v, want %+v", got, want)
			}
		})
	}
}

func TestWatchReplaysStatusToALaterWindow(t *testing.T) {
	rec := &recorder{}
	s := newTestService(platform.Darwin, rec)
	defer s.Close()
	s.Watch("srv")
	waitStatus(t, rec, 1)
	before := len(rec.all())
	// A second Monitor window opens after the event was emitted.
	s.Watch("srv")
	if got := rec.all(); len(got) <= before {
		t.Fatalf("second Watch did not replay the status: %v", got)
	}
}

func TestUnsupportedServerHasNoScripts(t *testing.T) {
	for _, os := range []platform.OS{platform.Darwin, platform.Windows} {
		if _, ok := targetFor(os); ok {
			t.Errorf("targetFor(%s) should be unsupported", os)
		}
	}
	for _, os := range []platform.OS{platform.Linux, platform.Unknown, ""} {
		tg, ok := targetFor(os)
		if !ok || tg.collect == "" || tg.procs == "" || tg.info == "" {
			t.Errorf("targetFor(%q) = %+v, %v", os, tg, ok)
		}
	}
}

func TestPosixLaunchKeepsTheScriptOffTheCommandLine(t *testing.T) {
	cmd, stdin := posixLaunch("echo hi")
	if cmd != "sh -s" || stdin != "echo hi" {
		t.Fatalf("launch = %q, %q", cmd, stdin)
	}
}

// Windows scripts end every line with CRLF; the stream must still split.
func TestReaderAcceptsCRLF(t *testing.T) {
	in := strings.NewReader("@@alpha\r\none\r\ntwo\r\n@@beta\r\nthree\r\n@@end\r\n@@alpha\r\nfour\r\n@@end\r\n")
	r := NewReader(in)
	got, err := r.sections()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got["alpha"], "|") != "one|two" || strings.Join(got["beta"], "|") != "three" {
		t.Fatalf("sections = %q", got)
	}
	got, err = r.sections()
	if err != nil || strings.Join(got["alpha"], "|") != "four" {
		t.Fatalf("second snapshot = %q, %v", got, err)
	}
}

func TestSetStatusClearsOnlyWhatIsSet(t *testing.T) {
	rec := &recorder{}
	s := newTestService(platform.Darwin, rec)
	// A watcher without a collector goroutine, so nothing sets a status behind the test.
	s.watchers["srv"] = &watcher{refs: 1}

	s.setStatus("srv", nil) // nothing to clear: no event, no change
	s.setStatus("srv", &Status{ServerID: "srv", State: StateError})
	if got := s.watchers["srv"].status; got == nil || got.State != StateError {
		t.Fatalf("status not kept: %+v", got)
	}
	s.setStatus("srv", nil)
	if got := s.watchers["srv"].status; got != nil {
		t.Fatalf("status not cleared: %+v", got)
	}
	if got := rec.all(); len(got) != 1 {
		t.Fatalf("events = %v, want only the error", got)
	}
}
