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
	if _, err := e.m.SFTP(e.id); err != nil {
		t.Fatal(err)
	}

	testsshd.Restart(t)

	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		if e.rec.sawState(conn.StateReconnecting) && e.m.State(e.id) == conn.StateConnected {
			if got := run(t, e.m, e.id, "echo back"); got != "back" {
				t.Fatalf("after reconnect: %q", got)
			}
			// The SFTP client of the dead connection must not be reused.
			sc, err := e.m.SFTP(e.id)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := sc.Getwd(); err != nil {
				t.Fatalf("sftp after reconnect: %v", err)
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

func TestAutomaticUsesDefaultKeyLikeSSH(t *testing.T) {
	// No agent, but ~/.ssh/id_ed25519 exists: the ssh command would log in,
	// so automatic mode must too.
	home := t.TempDir()
	key, err := os.ReadFile(testsshd.KeyPath())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".ssh"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".ssh", "id_ed25519"), key, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("SSH_AUTH_SOCK", "")

	srv := testServer()
	srv.Auth, srv.KeyPath = store.AuthAgent, ""
	e := setup(t, srv, conn.Options{})
	pin(t, e.hostKeys, testsshd.HostKey(t))
	if err := e.m.Connect(context.Background(), e.id); err != nil {
		t.Fatalf("automatic auth with a default key: %v", err)
	}

	t.Setenv("HOME", t.TempDir()) // no keys at all
	e2 := setup(t, srv, conn.Options{})
	pin(t, e2.hostKeys, testsshd.HostKey(t))
	if err := e2.m.Connect(context.Background(), e2.id); code(err) != apperr.ConnAgentUnavailable {
		t.Fatalf("no keys anywhere: err = %v; want CONN_AGENT_UNAVAILABLE", err)
	}
}

func TestProxyJump(t *testing.T) {
	e := setup(t, testServer(), conn.Options{
		KeepaliveInterval: 300 * time.Millisecond,
		KeepaliveMisses:   2,
		BackoffMin:        200 * time.Millisecond,
		BackoffMax:        time.Second,
	})
	ctx := context.Background()
	pin(t, e.hostKeys, testsshd.HostKey(t))
	inner, err := e.store.Save(ctx, store.Server{
		Name: "inner", Host: testsshd.InnerHost, Port: 22, User: testsshd.User,
		Auth: store.AuthKey, KeyPath: testsshd.KeyPath(), JumpID: e.id,
	})
	if err != nil {
		t.Fatal(err)
	}

	// The inner host's key is unknown: the prompt names the hop and is
	// answered for the server being opened.
	if err := e.m.Connect(ctx, inner.ID); code(err) != apperr.ConnHostKeyUnknown {
		t.Fatalf("first connect: err = %v; want CONN_HOSTKEY_UNKNOWN", err)
	}
	prompts := e.rec.prompts()
	if len(prompts) != 1 || prompts[0].ServerID != inner.ID || prompts[0].Host != testsshd.InnerHost+":22" {
		t.Fatalf("prompts = %+v", prompts)
	}
	if err := e.m.AcceptHostKey(ctx, inner.ID, prompts[0].Fingerprint); err != nil {
		t.Fatalf("accept: %v", err)
	}
	innerName := run(t, e.m, inner.ID, "hostname")
	if err := e.m.Connect(ctx, e.id); err != nil {
		t.Fatal(err)
	}
	if jump := run(t, e.m, e.id, "hostname"); jump == innerName {
		t.Fatalf("inner and jump report the same hostname %q", jump)
	}

	// Losing the hop drops the chain; it is rebuilt once the hop is back.
	testsshd.Restart(t, "sshd")
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		if e.rec.sawState(conn.StateReconnecting) && e.m.State(inner.ID) == conn.StateConnected {
			if got := run(t, e.m, inner.ID, "hostname"); got != innerName {
				t.Fatalf("after reconnect: %q", got)
			}
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("did not reconnect; state = %s", e.m.State(inner.ID))
}

func TestProxyJumpLoopIsRefused(t *testing.T) {
	e := setup(t, testServer(), conn.Options{})
	ctx := context.Background()
	b, err := e.store.Save(ctx, store.Server{
		Name: "b", Host: "b.invalid", Port: 22, User: testsshd.User,
		Auth: store.AuthKey, KeyPath: testsshd.KeyPath(), JumpID: e.id,
	})
	if err != nil {
		t.Fatal(err)
	}
	a, _ := e.store.Server(ctx, e.id)
	a.JumpID = b.ID
	if _, err := e.store.Save(ctx, a); err != nil {
		t.Fatal(err)
	}
	if err := e.m.Connect(ctx, e.id); code(err) != apperr.InvalidInput {
		t.Fatalf("err = %v; want INVALID_INPUT", err)
	}
}
