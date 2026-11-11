package sysmon

import (
	"context"
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
// OS and never open a session. While gate is open (non-nil and not closed)
// the OS is still being detected, as right after a connect.
type fakeClients struct {
	os   platform.OS
	gate chan struct{}
}

func (f fakeClients) Client(string) (*ssh.Client, error) { return &ssh.Client{}, nil }

func (f fakeClients) WaitPlatform(ctx context.Context, _ string) platform.OS {
	if f.gate != nil {
		select {
		case <-f.gate:
		case <-ctx.Done():
			return platform.Unknown
		}
	}
	return f.os
}

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
	return newGatedService(fakeClients{os: os}, rec)
}

func newGatedService(c fakeClients, rec *recorder) *Service {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return New(c, rec, log, Options{RetryEvery: time.Millisecond})
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
	for _, os := range []platform.OS{platform.Darwin} {
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

// Right after a connect the OS is still being detected. A collector started
// then would take the Linux scripts for a Windows server, so nothing runs
// and nothing is reported until the answer is in.
func TestNothingStartsWhileTheOSIsBeingDetected(t *testing.T) {
	rec := &recorder{}
	gate := make(chan struct{})
	s := newGatedService(fakeClients{os: platform.Darwin, gate: gate}, rec)
	defer s.Close()
	s.Watch("srv")

	time.Sleep(100 * time.Millisecond)
	if got := rec.all(); len(got) != 0 {
		t.Fatalf("status before the OS was known: %v", got)
	}
	close(gate)
	got := waitStatus(t, rec, 1)[0]
	if got.State != StateUnsupported || got.OS != platform.Darwin {
		t.Fatalf("status = %+v, want unsupported darwin", got)
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
	for _, os := range []platform.OS{platform.Darwin} {
		if _, ok := targetFor(os); ok {
			t.Errorf("targetFor(%s) should be unsupported", os)
		}
	}
	for _, os := range []platform.OS{platform.Linux, platform.Windows, platform.Unknown, ""} {
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
