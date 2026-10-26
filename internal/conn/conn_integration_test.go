//go:build integration

package conn_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"

	"davish/internal/apperr"
	"davish/internal/conn"
	"davish/internal/events"
	"davish/internal/store"
	"davish/internal/testsshd"
)

// recorder collects emitted events.
type recorder struct {
	mu     sync.Mutex
	events []recorded
}

type recorded struct {
	name    string
	payload any
}

func (r *recorder) Emit(name string, payload any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, recorded{name, payload})
}

func (r *recorder) prompts() []conn.HostKeyPrompt {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []conn.HostKeyPrompt
	for _, e := range r.events {
		if p, ok := e.payload.(conn.HostKeyPrompt); ok {
			out = append(out, p)
		}
	}
	return out
}

func (r *recorder) sawState(st conn.State) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, e := range r.events {
		if s, ok := e.payload.(conn.Status); ok && e.name == events.ConnStatus && s.State == st {
			return true
		}
	}
	return false
}

type env struct {
	m        *conn.Manager
	rec      *recorder
	store    *store.Store
	secrets  *store.Secrets
	hostKeys string
	id       string
}

type memKeyring map[string]string

func (k memKeyring) Set(s, u, v string) error { k[s+u] = v; return nil }
func (k memKeyring) Get(s, u string) (string, error) {
	if v, ok := k[s+u]; ok {
		return v, nil
	}
	return "", store.ErrNoSecret
}
func (k memKeyring) Delete(s, u string) error { delete(k, s+u); return nil }

func setup(t *testing.T, srv store.Server, opts conn.Options) *env {
	t.Helper()
	dir := t.TempDir()
	secrets := store.NewSecrets(memKeyring{})
	st := store.New(dir, secrets)
	saved, err := st.Save(context.Background(), srv)
	if err != nil {
		t.Fatal(err)
	}
	rec := &recorder{}
	hk := filepath.Join(dir, "known_hosts")
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	m := conn.NewManager(st, secrets, conn.NewHostKeys(hk), rec, log, opts)
	t.Cleanup(m.Close)
	return &env{m: m, rec: rec, store: st, secrets: secrets, hostKeys: hk, id: saved.ID}
}

func testServer() store.Server {
	return store.Server{
		Name: "test sshd", Host: testsshd.Host, Port: testsshd.Port, User: testsshd.User,
		Auth: store.AuthKey, KeyPath: testsshd.KeyPath(),
	}
}

func pin(t *testing.T, path string, key ssh.PublicKey) {
	t.Helper()
	line := knownhosts.Line([]string{knownhosts.Normalize("127.0.0.1:2222")}, key)
	if err := os.WriteFile(path, []byte(line+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func code(err error) apperr.Code {
	var ae *apperr.Error
	if errors.As(err, &ae) {
		return ae.Code
	}
	return ""
}

func run(t *testing.T, m *conn.Manager, id, cmd string) string {
	t.Helper()
	client, err := m.Client(id)
	if err != nil {
		t.Fatal(err)
	}
	sess, err := client.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sess.Close() }()
	out, err := sess.Output(cmd)
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(out))
}

func TestUnknownHostAskedThenAccepted(t *testing.T) {
	e := setup(t, testServer(), conn.Options{})
	ctx := context.Background()

	if err := e.m.Connect(ctx, e.id); code(err) != apperr.ConnHostKeyUnknown {
		t.Fatalf("first connect: err = %v; want CONN_HOSTKEY_UNKNOWN", err)
	}
	prompts := e.rec.prompts()
	want := conn.Fingerprint(testsshd.HostKey(t))
	if len(prompts) != 1 || prompts[0].Fingerprint != want {
		t.Fatalf("prompts = %+v; want one with %s", prompts, want)
	}

	if err := e.m.AcceptHostKey(ctx, e.id, "SHA256:forged"); err == nil {
		t.Fatal("accepted a fingerprint that was never shown")
	}
	// The forged attempt consumed the pending key; ask again like the UI would.
	_ = e.m.Connect(ctx, e.id)
	if err := e.m.AcceptHostKey(ctx, e.id, want); err != nil {
		t.Fatalf("accept: %v", err)
	}
	if got := run(t, e.m, e.id, "whoami"); got != testsshd.User {
		t.Errorf("whoami = %q", got)
	}

	// The key is now pinned: a fresh manager connects without asking.
	e2 := setup(t, testServer(), conn.Options{})
	data, _ := os.ReadFile(e.hostKeys)
	_ = os.WriteFile(e2.hostKeys, data, 0o600) //nolint:gosec // path is under t.TempDir()
	if err := e2.m.Connect(ctx, e2.id); err != nil {
		t.Fatalf("connect with pinned key: %v", err)
	}
}

func TestHostKeyChangedIsRefused(t *testing.T) {
	e := setup(t, testServer(), conn.Options{})
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	other, _ := ssh.NewPublicKey(pub)
	pin(t, e.hostKeys, other)

	err := e.m.Connect(context.Background(), e.id)
	if code(err) != apperr.ConnHostKeyChanged {
		t.Fatalf("err = %v; want CONN_HOSTKEY_CHANGED", err)
	}
	if len(e.rec.prompts()) != 0 {
		t.Error("a changed key must not offer the accept prompt")
	}
	if e.m.State(e.id) != conn.StateFailed {
		t.Errorf("state = %s; want failed", e.m.State(e.id))
	}
}

func TestWrongKeyFailsAuth(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	block, _ := ssh.MarshalPrivateKey(priv, "")
	keyPath := filepath.Join(t.TempDir(), "wrong_ed25519")
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(block), 0o600); err != nil {
		t.Fatal(err)
	}
	srv := testServer()
	srv.KeyPath = keyPath
	e := setup(t, srv, conn.Options{})
	pin(t, e.hostKeys, testsshd.HostKey(t))

	err := e.m.Connect(context.Background(), e.id)
	if code(err) != apperr.ConnAuthFailed {
		t.Fatalf("err = %v; want CONN_AUTH_FAILED", err)
	}
	if strings.Contains(apperr.Format(err).(*apperr.Error).Message, keyPath) {
		t.Error("error shown to the user contains the key path")
	}
}

func TestPasswordWithoutSecret(t *testing.T) {
	srv := testServer()
	srv.Auth, srv.KeyPath = store.AuthPassword, ""
	e := setup(t, srv, conn.Options{})
	pin(t, e.hostKeys, testsshd.HostKey(t))

	if err := e.m.Connect(context.Background(), e.id); code(err) != apperr.ConnSecretMissing {
		t.Fatalf("err = %v; want CONN_SECRET_MISSING", err)
	}
}

func TestReconnectsAfterServerRestart(t *testing.T) {
	e := setup(t, testServer(), conn.Options{
		KeepaliveInterval: 300 * time.Millisecond,
		KeepaliveMisses:   2,
		BackoffMin:        200 * time.Millisecond,
		BackoffMax:        time.Second,
	})
	pin(t, e.hostKeys, testsshd.HostKey(t))
	if err := e.m.Connect(context.Background(), e.id); err != nil {
		t.Fatal(err)
	}

	testsshd.Restart(t)

	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		if e.rec.sawState(conn.StateReconnecting) && e.m.State(e.id) == conn.StateConnected {
			if got := run(t, e.m, e.id, "echo back"); got != "back" {
				t.Fatalf("after reconnect: %q", got)
			}
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("did not reconnect; state = %s", e.m.State(e.id))
}

func TestDisconnect(t *testing.T) {
	e := setup(t, testServer(), conn.Options{})
	pin(t, e.hostKeys, testsshd.HostKey(t))
	if err := e.m.Connect(context.Background(), e.id); err != nil {
		t.Fatal(err)
	}
	e.m.Disconnect(e.id)
	if _, err := e.m.Client(e.id); code(err) != apperr.ConnNotConnected {
		t.Errorf("Client after disconnect: err = %v", err)
	}
	if e.m.State(e.id) != conn.StateDisconnected {
		t.Errorf("state = %s", e.m.State(e.id))
	}
}
