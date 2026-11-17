//go:build integration

package fleet_test

import (
	"context"
	"encoding/base64"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/ssh/knownhosts"

	"davish/internal/audit"
	"davish/internal/conn"
	"davish/internal/events"
	"davish/internal/fleet"
	"davish/internal/store"
	"davish/internal/term"
	"davish/internal/testsshd"
)

type keyring map[string]string

func (k keyring) Set(s, u, v string) error { k[s+u] = v; return nil }
func (k keyring) Get(s, u string) (string, error) {
	if v, ok := k[s+u]; ok {
		return v, nil
	}
	return "", store.ErrNoSecret
}
func (k keyring) Delete(s, u string) error { delete(k, s+u); return nil }

type discard struct{}

func (discard) Emit(string, any) {}

// screen keeps each session's output and every job event.
type screen struct {
	mu   sync.Mutex
	out  map[string]*strings.Builder
	jobs chan fleet.Job
}

func (s *screen) Emit(name string, payload any) {
	switch name {
	case events.TermData:
		d := payload.(term.Data)
		b, _ := base64.StdEncoding.DecodeString(d.B64)
		s.mu.Lock()
		if s.out[d.ID] == nil {
			s.out[d.ID] = &strings.Builder{}
		}
		s.out[d.ID].Write(b)
		s.mu.Unlock()
	case events.FleetJob:
		s.jobs <- payload.(fleet.Job)
	}
}

func (s *screen) text(id string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.out[id] == nil {
		return ""
	}
	return s.out[id].String()
}

type setup struct {
	terms *term.Service
	fleet *fleet.Service
	log   *audit.Log
	scr   *screen
	ids   []string // tester@A, tester@B, sudoer@A
}

func newSetup(t *testing.T) *setup {
	t.Helper()
	dir := t.TempDir()
	secrets := store.NewSecrets(keyring{})
	st := store.New(dir, secrets)
	var ids, lines []string
	for _, s := range []struct {
		user string
		port int
	}{{testsshd.User, testsshd.Port}, {testsshd.User, testsshd.PortB}, {testsshd.Sudoer, testsshd.Port}} {
		srv, err := st.Save(context.Background(), store.Server{
			Name: s.user + "-" + strconv.Itoa(s.port), Host: testsshd.Host, Port: s.port, User: s.user,
			Auth: store.AuthKey, KeyPath: testsshd.KeyPath(),
		})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, srv.ID)
		addr := testsshd.Host + ":" + strconv.Itoa(s.port)
		lines = append(lines, knownhosts.Line([]string{knownhosts.Normalize(addr)}, testsshd.HostKey(t)))
	}
	hk := filepath.Join(dir, "known_hosts")
	if err := os.WriteFile(hk, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	quiet := slog.New(slog.NewTextHandler(io.Discard, nil))
	m := conn.NewManager(st, secrets, conn.NewHostKeys(hk), discard{}, quiet, conn.Options{})
	t.Cleanup(m.Close)
	for _, id := range ids {
		if err := m.Connect(context.Background(), id); err != nil {
			t.Fatal(err)
		}
	}
	scr := &screen{out: map[string]*strings.Builder{}, jobs: make(chan fleet.Job, 100)}
	terms := term.New(m, scr, quiet)
	t.Cleanup(terms.CloseAll)
	log := audit.New(discard{})
	return &setup{terms: terms, fleet: fleet.New(terms, log, scr, quiet), log: log, scr: scr, ids: ids}
}

// wait returns each server's final job; onRunning sees each running one.
func (s *setup) wait(t *testing.T, n int, onRunning func(fleet.Job)) map[string]fleet.Job {
	t.Helper()
	out := map[string]fleet.Job{}
	timeout := time.After(30 * time.Second)
	for len(out) < n {
		select {
		case j := <-s.scr.jobs:
			switch j.State {
			case fleet.Running:
				if onRunning != nil {
					onRunning(j)
				}
			case fleet.Queued:
			default:
				out[j.ServerID] = j
			}
		case <-timeout:
			t.Fatalf("timed out with %+v", out)
		}
	}
	return out
}

func each(ids []string, cmd string) []fleet.Item {
	var it []fleet.Item
	for _, id := range ids {
		it = append(it, fleet.Item{ServerID: id, Command: cmd})
	}
	return it
}

// The M49 acceptance, in parts: a plain command on three servers, a sudo
// prompt answered in the tab, and sudo -n refused without a password.
func TestRunOnThreeServers(t *testing.T) {
	s := newSetup(t)
	if _, _, err := s.fleet.Start(context.Background(), each(s.ids, "uptime"), 100, 30); err != nil {
		t.Fatal(err)
	}
	got := s.wait(t, 3, nil)
	for _, id := range s.ids {
		j := got[id]
		if j.State != fleet.Done || j.Exit != 0 || !strings.Contains(s.scr.text(j.TermID), "load average") {
			t.Errorf("%s: %+v, output %q", id, j, s.scr.text(j.TermID))
		}
	}
	if n := len(s.log.Entries()); n != 3 {
		t.Errorf("audit has %d entries, want one per server", n)
	}
}

func TestSudoPromptInTab(t *testing.T) {
	s := newSetup(t)
	sudoer := s.ids[2]
	// -k forgets any cached sudo, so it must ask.
	if _, _, err := s.fleet.Start(context.Background(), each([]string{sudoer}, "sudo -k; sudo true && echo granted"), 100, 30); err != nil {
		t.Fatal(err)
	}
	got := s.wait(t, 1, func(j fleet.Job) {
		// The user sees the prompt in the tab and types the password there.
		deadline := time.Now().Add(10 * time.Second)
		for !strings.Contains(s.scr.text(j.TermID), "password for") {
			if time.Now().After(deadline) {
				t.Errorf("no sudo prompt: %q", s.scr.text(j.TermID))
				return
			}
			time.Sleep(50 * time.Millisecond)
		}
		if err := s.terms.Write(j.TermID, testsshd.SudoPassword+"\r"); err != nil {
			t.Error(err)
		}
	})
	j := got[sudoer]
	if j.State != fleet.Done || !strings.Contains(s.scr.text(j.TermID), "granted") {
		t.Errorf("job %+v, output %q", j, s.scr.text(j.TermID))
	}
	// The password typed into the tab is not echoed, nor in the audit.
	if strings.Contains(s.scr.text(j.TermID), testsshd.SudoPassword) {
		t.Error("the password was echoed")
	}
	for _, e := range s.log.Entries() {
		if strings.Contains(e.Command, testsshd.SudoPassword) {
			t.Error("the password reached the audit log")
		}
	}

	// sudo -n never asks: it fails and says a password is required.
	if _, _, err := s.fleet.Start(context.Background(), each([]string{sudoer}, "sudo -k; sudo -n true"), 100, 30); err != nil {
		t.Fatal(err)
	}
	got = s.wait(t, 1, nil)
	if j := got[sudoer]; j.State != fleet.Failed || !strings.Contains(s.scr.text(j.TermID), "a password is required") {
		t.Errorf("sudo -n: %+v, output %q", j, s.scr.text(j.TermID))
	}
}

func TestStopRunning(t *testing.T) {
	s := newSetup(t)
	id, _, err := s.fleet.Start(context.Background(), each(s.ids[:2], "sleep 30"), 100, 30)
	if err != nil {
		t.Fatal(err)
	}
	running := 0
	got := s.wait(t, 2, func(fleet.Job) {
		if running++; running == 2 {
			s.fleet.Stop(id)
		}
	})
	for _, j := range got {
		if j.State != fleet.Stopped {
			t.Errorf("%+v", j)
		}
	}
}
