//go:build integration

package conn_test

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"

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
	srv, err := st.Save(context.Background(), store.Server{
		Name: "proxied", Host: testsshd.Host, Port: testsshd.Port, User: testsshd.User,
		Auth: store.AuthKey, KeyPath: testsshd.KeyPath(), ProxyCommand: "nc %h %p",
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
