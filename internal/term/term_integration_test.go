//go:build integration

package term_test

import (
	"context"
	"encoding/base64"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/ssh/knownhosts"

	"davish/internal/conn"
	"davish/internal/events"
	"davish/internal/store"
	"davish/internal/term"
	"davish/internal/testsshd"
)

// screen collects term events for one session.
type screen struct {
	mu   sync.Mutex
	out  strings.Builder
	exit *term.Exit
}

func (s *screen) Emit(name string, payload any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch p := payload.(type) {
	case term.Data:
		b, _ := base64.StdEncoding.DecodeString(p.B64)
		s.out.Write(b)
	case term.Exit:
		if name == events.TermExit {
			s.exit = &p
		}
	}
}

func (s *screen) waitFor(t *testing.T, cond func(out string, exit *term.Exit) bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		s.mu.Lock()
		ok := cond(s.out.String(), s.exit)
		s.mu.Unlock()
		if ok {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	t.Fatalf("timed out; output so far:\n%s", s.out.String())
}

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

func connect(t *testing.T) (*conn.Manager, string) {
	t.Helper()
	dir := t.TempDir()
	secrets := store.NewSecrets(keyring{})
	st := store.New(dir, secrets)
	srv, err := st.Save(context.Background(), store.Server{
		Name: "test", Host: testsshd.Host, Port: testsshd.Port, User: testsshd.User,
		Auth: store.AuthKey, KeyPath: testsshd.KeyPath(),
	})
	if err != nil {
		t.Fatal(err)
	}
	hk := filepath.Join(dir, "known_hosts")
	line := knownhosts.Line([]string{knownhosts.Normalize("127.0.0.1:2222")}, testsshd.HostKey(t))
	if err := os.WriteFile(hk, []byte(line+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	m := conn.NewManager(st, secrets, conn.NewHostKeys(hk), discard{}, log, conn.Options{})
	t.Cleanup(m.Close)
	if err := m.Connect(context.Background(), srv.ID); err != nil {
		t.Fatal(err)
	}
	return m, srv.ID
}

func TestShellRunsCommandsAndReportsExit(t *testing.T) {
	m, id := connect(t)
	scr := &screen{}
	svc := term.New(m, scr, slog.New(slog.NewTextHandler(io.Discard, nil)))
	t.Cleanup(svc.CloseAll)

	sid, err := svc.Start(context.Background(), id, 80, 24)
	if err != nil {
		t.Fatal(err)
	}
	// The arithmetic proves a shell ran it, not that the input was echoed back.
	if err := svc.Write(sid, "echo hello-$((40+2))\n"); err != nil {
		t.Fatal(err)
	}
	scr.waitFor(t, func(out string, _ *term.Exit) bool { return strings.Contains(out, "hello-42") })

	if err := svc.Resize(sid, 132, 40); err != nil {
		t.Fatal(err)
	}
	if err := svc.Write(sid, "stty size\n"); err != nil {
		t.Fatal(err)
	}
	scr.waitFor(t, func(out string, _ *term.Exit) bool { return strings.Contains(out, "40 132") })

	if err := svc.Write(sid, "exit 3\n"); err != nil {
		t.Fatal(err)
	}
	scr.waitFor(t, func(_ string, exit *term.Exit) bool { return exit != nil })
	if scr.exit.Code != 3 || scr.exit.ID != sid {
		t.Errorf("exit = %+v; want code 3 for %s", scr.exit, sid)
	}
	if err := svc.Write(sid, "x"); err == nil {
		t.Error("write after exit succeeded")
	}
}

func TestStartNeedsConnection(t *testing.T) {
	m, _ := connect(t)
	svc := term.New(m, discard{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if _, err := svc.Start(context.Background(), "not-connected", 80, 24); err == nil {
		t.Error("started a shell on a server that is not connected")
	}
}
