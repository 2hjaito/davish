//go:build integration

package testsshd

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh/knownhosts"

	"davish/internal/conn"
	"davish/internal/store"
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

// Quiet is a logger that drops everything.
func Quiet() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// Connect returns a manager connected to the test server, with its host
// key pinned, and the server's ID. It is closed when the test ends.
func Connect(t *testing.T) (*conn.Manager, string) {
	t.Helper()
	dir := t.TempDir()
	secrets := store.NewSecrets(keyring{})
	st := store.New(dir, secrets)
	srv, err := st.Save(context.Background(), store.Server{
		Name: "test", Host: Host, Port: Port, User: User,
		Auth: store.AuthKey, KeyPath: KeyPath(),
	})
	if err != nil {
		t.Fatal(err)
	}
	hk := filepath.Join(dir, "known_hosts")
	line := knownhosts.Line([]string{knownhosts.Normalize("127.0.0.1:2222")}, HostKey(t))
	if err := os.WriteFile(hk, []byte(line+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	m := conn.NewManager(st, secrets, conn.NewHostKeys(hk), discard{}, Quiet(), conn.Options{})
	t.Cleanup(m.Close)
	if err := m.Connect(context.Background(), srv.ID); err != nil {
		t.Fatal(err)
	}
	return m, srv.ID
}

// Run runs cmd on the test server and returns its trimmed output.
func Run(t *testing.T, m *conn.Manager, id, cmd string) string {
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
	out, err := sess.CombinedOutput(cmd)
	if err != nil {
		t.Fatalf("%s: %v\n%s", cmd, err, out)
	}
	return strings.TrimSpace(string(out))
}
