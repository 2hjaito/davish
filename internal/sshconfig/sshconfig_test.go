package sshconfig

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"davish/internal/store"
)

func writeConfig(t *testing.T, files map[string]string) string {
	t.Helper()
	home := t.TempDir()
	for name, body := range files {
		p := filepath.Join(home, ".ssh", name)
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return home
}

func TestLoad(t *testing.T) {
	home := writeConfig(t, map[string]string{
		"config": `# comment
Include conf.d/*
Host web web-alias
  HostName 10.0.0.5
  User deploy
  IdentityFile ~/.ssh/web_ed25519
Host db
  HostName=db.internal
  Port 2222
  ProxyJump bastion
Host *.example.com !skip.example.com
  User nobody
Host *
  User fallback
  Port 22
`,
		"conf.d/bastion": "Host bastion\n  HostName bastion.dangth.dev\n  IdentityFile %d/.ssh/b\n",
	})
	hosts, err := Load(home, "me")
	if err != nil {
		t.Fatal(err)
	}
	want := []Host{
		{Alias: "bastion", HostName: "bastion.dangth.dev", User: "fallback", Port: 22, IdentityFile: home + "/.ssh/b"},
		{Alias: "web", HostName: "10.0.0.5", User: "deploy", Port: 22, IdentityFile: "~/.ssh/web_ed25519"},
		{Alias: "web-alias", HostName: "10.0.0.5", User: "deploy", Port: 22, IdentityFile: "~/.ssh/web_ed25519"},
		{Alias: "db", HostName: "db.internal", User: "fallback", Port: 2222, ProxyJump: "bastion"},
	}
	if len(hosts) != len(want) {
		t.Fatalf("got %d hosts: %+v", len(hosts), hosts)
	}
	for i := range want {
		if hosts[i] != want[i] {
			t.Errorf("host %d = %+v, want %+v", i, hosts[i], want[i])
		}
	}
}

func TestLoadMissing(t *testing.T) {
	hosts, err := Load(t.TempDir(), "me")
	if err != nil || hosts != nil {
		t.Fatalf("got %v, %v", hosts, err)
	}
}

func TestParseJump(t *testing.T) {
	hops, err := ParseJump("a, root@b:2200,ssh://c")
	if err != nil {
		t.Fatal(err)
	}
	want := []Hop{
		{Alias: "a", Host: "a"},
		{Alias: "root@b:2200", User: "root", Host: "b", Port: 2200},
		{Alias: "c", Host: "c"},
	}
	for i := range want {
		if hops[i] != want[i] {
			t.Errorf("hop %d = %+v, want %+v", i, hops[i], want[i])
		}
	}
	if _, err := ParseJump("a,,b"); err == nil {
		t.Error("empty hop accepted")
	}
}

type memSaver struct{ list []store.Server }

func (m *memSaver) Servers(context.Context) ([]store.Server, error) { return m.list, nil }

func (m *memSaver) Save(_ context.Context, s store.Server) (store.Server, error) {
	s.ID = s.Name + "-id"
	m.list = append(m.list, s)
	return s, nil
}

func TestImport(t *testing.T) {
	hosts := []Host{
		{Alias: "bastion", HostName: "b.dangth.dev", User: "me", Port: 22},
		{Alias: "db", HostName: "db.internal", User: "me", Port: 22, ProxyJump: "bastion,ops@mid:2200", IdentityFile: "~/.ssh/k"},
		{Alias: "old", HostName: "old.dangth.dev", User: "me", Port: 22},
	}
	s := &memSaver{list: []store.Server{{ID: "x", Name: "Old", Host: "old.dangth.dev", User: "me", Port: 22}}}
	added, err := Import(context.Background(), s, hosts, []string{"db", "old"}, "me")
	if err != nil {
		t.Fatal(err)
	}
	if len(added) != 3 {
		t.Fatalf("added %+v", added)
	}
	b, mid, db := added[0], added[1], added[2]
	if b.Name != "bastion" || b.JumpID != "" {
		t.Errorf("bastion = %+v", b)
	}
	if mid.Host != "mid" || mid.User != "ops" || mid.Port != 2200 || mid.JumpID != b.ID {
		t.Errorf("mid = %+v", mid)
	}
	if db.JumpID != mid.ID || db.Auth != store.AuthKey || db.KeyPath != "~/.ssh/k" {
		t.Errorf("db = %+v", db)
	}
	c := Candidates(hosts, s.list)
	for _, x := range c {
		if !x.Exists {
			t.Errorf("%s not marked existing", x.Alias)
		}
	}
}

func TestImportLoop(t *testing.T) {
	hosts := []Host{
		{Alias: "a", HostName: "a", User: "me", Port: 22, ProxyJump: "b"},
		{Alias: "b", HostName: "b", User: "me", Port: 22, ProxyJump: "a"},
	}
	if _, err := Import(context.Background(), &memSaver{}, hosts, []string{"a"}, "me"); err == nil {
		t.Fatal("loop accepted")
	}
}
