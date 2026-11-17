//go:build integration

package ansibleinv_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh/knownhosts"

	"davish/internal/ansibleinv"
	"davish/internal/conn"
	"davish/internal/store"
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

// The M48 acceptance: an inventory with two hosts on the test servers,
// a Vault host and a winrm host; the two import, connect, and a second
// import adds nothing.
func TestImportAndConnect(t *testing.T) {
	dir := t.TempDir()
	inv := fmt.Sprintf(`[web]
a ansible_host=%[1]s ansible_port=%[2]d
b ansible_host=%[1]s ansible_port=%[3]d

[secret]
v ansible_host="$ANSIBLE_VAULT;1.1;AES256"

[win]
w ansible_connection=winrm

[test:children]
web
secret

[test:vars]
ansible_user=%[4]s
ansible_ssh_private_key_file=%[5]s
`, testsshd.Host, testsshd.Port, testsshd.PortB, testsshd.User, testsshd.KeyPath())
	path := filepath.Join(dir, "hosts")
	if err := os.WriteFile(path, []byte(inv), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := ansibleinv.Load(path, "nobody")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Hosts) != 2 || len(got.Skipped) != 2 {
		t.Fatalf("hosts %+v, skipped %+v", got.Hosts, got.Skipped)
	}

	secrets := store.NewSecrets(keyring{})
	st := store.New(dir, secrets)
	added, err := ansibleinv.Import(context.Background(), st, got, []string{"a", "b"}, "nobody")
	if err != nil || len(added) != 2 {
		t.Fatalf("added %v, %v", added, err)
	}

	var lines []string
	for _, p := range []int{testsshd.Port, testsshd.PortB} {
		addr := testsshd.Host + ":" + strconv.Itoa(p)
		lines = append(lines, knownhosts.Line([]string{knownhosts.Normalize(addr)}, testsshd.HostKey(t)))
	}
	hk := filepath.Join(dir, "known_hosts")
	if err := os.WriteFile(hk, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	m := conn.NewManager(st, secrets, conn.NewHostKeys(hk), discard{}, testsshd.Quiet(), conn.Options{})
	t.Cleanup(m.Close)
	for _, s := range added {
		if s.Auth != store.AuthKey {
			t.Errorf("%s auth = %s", s.Name, s.Auth)
		}
		if err := m.Connect(context.Background(), s.ID); err != nil {
			t.Fatalf("connect %s: %v", s.Name, err)
		}
	}

	again, err := ansibleinv.Import(context.Background(), st, got, []string{"a", "b"}, "nobody")
	if err != nil || len(again) != 0 {
		t.Errorf("second import added %v, %v", again, err)
	}
	all, err := st.Servers(context.Background())
	if err != nil || len(all) != 2 {
		t.Errorf("servers = %d, %v", len(all), err)
	}
}
