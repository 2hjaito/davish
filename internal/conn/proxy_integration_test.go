//go:build integration

package conn_test

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh/knownhosts"

	"davish/internal/apperr"
	"davish/internal/conn"
	"davish/internal/store"
	"davish/internal/testsshd"
)

// A ProxyCommand is the whole transport: nc bridging to the test server
// stands in for websocat bridging to a relay, and the host key check and
// the login run on top of its stdin and stdout, exactly as with ssh.
func TestConnectThroughProxyCommand(t *testing.T) {
	dir := t.TempDir()
	secrets := store.NewSecrets(proxyKeyring{})
	st := store.New(dir, secrets)
	// Typed in the dialog, so confirmed (ADR 0031).
	srv, err := st.SaveWith(context.Background(), store.Server{
		Name: "proxied", Host: testsshd.Host, Port: testsshd.Port, User: testsshd.User,
		Auth: store.AuthKey, KeyPath: testsshd.KeyPath(), ProxyCommand: "nc %h %p",
	}, store.SaveOptions{ProxyTyped: true})
	if err != nil {
		t.Fatal(err)
	}
	addr := testsshd.Host + ":" + strconv.Itoa(testsshd.Port)
	line := knownhosts.Line([]string{knownhosts.Normalize(addr)}, testsshd.HostKey(t))
	if err := os.WriteFile(filepath.Join(dir, "known_hosts"), []byte(line+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	m := conn.NewManager(st, secrets, conn.NewHostKeys(filepath.Join(dir, "known_hosts")),
		proxyEmitter{}, testsshd.Quiet(), conn.Options{})
	t.Cleanup(m.Close)

	if err := m.Connect(context.Background(), srv.ID); err != nil {
		t.Fatalf("connect through the proxy command: %v\ndetail: %s", err, apperr.Detail(err))
	}
	if out := testsshd.Run(t, m, srv.ID, "echo through-the-tunnel"); out != "through-the-tunnel" {
		t.Fatalf("output %q", out)
	}
}

type proxyKeyring map[string]string

func (k proxyKeyring) Set(s, u, v string) error { k[s+u] = v; return nil }
func (k proxyKeyring) Get(s, u string) (string, error) {
	if v, ok := k[s+u]; ok {
		return v, nil
	}
	return "", store.ErrNoSecret
}
func (k proxyKeyring) Delete(s, u string) error { delete(k, s+u); return nil }

type proxyEmitter struct{}

func (proxyEmitter) Emit(string, any) {}

// A proxy command that came from an import is not run until the user has seen
// it: Connect refuses with CONN_PROXY_UNCONFIRMED and starts nothing; once the
// user confirms that exact command, the same Connect logs in through it.
func TestProxyCommandAsksBeforeItRuns(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "proxy-ran")
	secrets := store.NewSecrets(proxyKeyring{})
	st := store.New(dir, secrets)
	cmd := "touch " + marker + "; nc %h %p"
	// Saved the way an import does: not typed in the dialog.
	srv, err := st.Save(context.Background(), store.Server{
		Name: "imported", Host: testsshd.Host, Port: testsshd.Port, User: testsshd.User,
		Auth: store.AuthKey, KeyPath: testsshd.KeyPath(), ProxyCommand: cmd,
	})
	if err != nil {
		t.Fatal(err)
	}
	addr := testsshd.Host + ":" + strconv.Itoa(testsshd.Port)
	line := knownhosts.Line([]string{knownhosts.Normalize(addr)}, testsshd.HostKey(t))
	if err := os.WriteFile(filepath.Join(dir, "known_hosts"), []byte(line+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	m := conn.NewManager(st, secrets, conn.NewHostKeys(filepath.Join(dir, "known_hosts")),
		proxyEmitter{}, testsshd.Quiet(), conn.Options{})
	t.Cleanup(m.Close)

	if err := m.Connect(context.Background(), srv.ID); code(err) != apperr.ConnProxyUnconfirmed {
		t.Fatalf("err = %v; want CONN_PROXY_UNCONFIRMED", err)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("the proxy command ran before it was confirmed")
	}

	plan, err := conn.PlanProxy(srv)
	if err != nil || !strings.Contains(plan.Command, testsshd.Host) || strings.Contains(plan.Command, "%") {
		t.Fatalf("plan = %+v, %v", plan, err)
	}
	if err := st.ConfirmProxy(context.Background(), srv.ID, plan.Hash); err != nil {
		t.Fatal(err)
	}
	if err := m.Connect(context.Background(), srv.ID); err != nil {
		t.Fatalf("connect after confirming: %v\ndetail: %s", err, apperr.Detail(err))
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatal("the confirmed command did not run")
	}
}

// A proxy that dies must end the connection at once, with its own words in the
// error for the person and none of them in the log (a proxy may print a token
// or a login URL). Before, Davish held the write end of the proxy's stdout, so
// the handshake waited for its whole timeout and Connect hung.
func TestProxyDeathEndsConnectAtOnce(t *testing.T) {
	dir := t.TempDir()
	secrets := store.NewSecrets(proxyKeyring{})
	st := store.New(dir, secrets)
	srv, err := st.SaveWith(context.Background(), store.Server{
		Name: "dies", Host: testsshd.Host, Port: testsshd.Port, User: testsshd.User,
		Auth: store.AuthKey, KeyPath: testsshd.KeyPath(),
		ProxyCommand: "echo relay-refused-token=SECRET123 >&2; exit 3",
	}, store.SaveOptions{ProxyTyped: true})
	if err != nil {
		t.Fatal(err)
	}
	var logs bytes.Buffer
	m := conn.NewManager(st, secrets, conn.NewHostKeys(filepath.Join(dir, "known_hosts")),
		proxyEmitter{}, slog.New(slog.NewTextHandler(&logs, nil)), conn.Options{DialTimeout: 8 * time.Second})
	t.Cleanup(m.Close)

	start := time.Now()
	err = m.Connect(context.Background(), srv.ID)
	took := time.Since(start)
	if err == nil {
		t.Fatal("connected through a proxy that exits")
	}
	if took > 3*time.Second {
		t.Fatalf("Connect took %v after the proxy died; it waited for the handshake timeout", took)
	}
	if detail := apperr.Detail(err); !strings.Contains(detail, "SECRET123") || !strings.Contains(detail, "exit") && !strings.Contains(detail, "stderr") {
		t.Errorf("the error does not say what the proxy said: %q", detail)
	}
	if strings.Contains(logs.String(), "SECRET123") {
		t.Errorf("what the proxy printed reached the log:\n%s", logs.String())
	}
}
