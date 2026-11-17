package ansibleinv

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"davish/internal/apperr"
	"davish/internal/store"
)

// sample has 3 groups, a child group, a range, a Vault host and a winrm
// host: the M48 acceptance inventory.
const sample = `# Inventory for the acceptance test
bastion ansible_host=10.0.0.1 ansible_user=jump

[web]
web[1:3].example.com ansible_user=deploy

[db]
db1 ansible_host=10.0.1.5 ansible_port=2222 ansible_password=hunter2
db2 ansible_host="!vault |" # vaulted inline

[win]
win1 ansible_connection=winrm ansible_host=10.0.2.9

[prod:children]
web
db

[prod:vars]
ansible_user=ops
ansible_ssh_common_args='-o ProxyJump=bastion -o StrictHostKeyChecking=no'

[all:vars]
ansible_port=22
`

func parse(t *testing.T, src string) Inventory {
	t.Helper()
	inv, err := Parse(strings.NewReader(src), "/inv")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return inv
}

func byName(inv Inventory) map[string]Host {
	m := map[string]Host{}
	for _, h := range inv.Hosts {
		m[h.Name] = h
	}
	return m
}

func TestParseSample(t *testing.T) {
	inv := parse(t, sample)
	hosts := byName(inv)

	var names []string
	for _, h := range inv.Hosts {
		names = append(names, h.Name)
	}
	want := []string{"bastion", "web1.example.com", "web2.example.com", "web3.example.com", "db1"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("hosts = %v, want %v", names, want)
	}

	if h := hosts["bastion"]; h.Address != "10.0.0.1" || h.User != "jump" || h.Port != 22 || h.ProxyJump != "" ||
		!reflect.DeepEqual(h.Groups, []string{"ungrouped"}) {
		t.Errorf("bastion = %+v", h)
	}
	// The host's own ansible_user beats the parent group's.
	w := hosts["web2.example.com"]
	if w.Address != "web2.example.com" || w.User != "deploy" || w.ProxyJump != "bastion" ||
		!reflect.DeepEqual(w.Groups, []string{"web", "prod"}) {
		t.Errorf("web2 = %+v", w)
	}
	if !reflect.DeepEqual(w.Notes, []Issue{{Code: NoteSSHArgs, Detail: "StrictHostKeyChecking"}}) {
		t.Errorf("web2 notes = %+v", w.Notes)
	}
	d := hosts["db1"]
	if d.Address != "10.0.1.5" || d.Port != 2222 || d.User != "ops" {
		t.Errorf("db1 = %+v", d)
	}
	if !hasNote(d, NotePassword) {
		t.Errorf("db1 notes = %+v, want a password note", d.Notes)
	}

	skipped := map[string]Issue{}
	for _, s := range inv.Skipped {
		skipped[s.Name] = s.Issue
	}
	wantSkip := map[string]Issue{
		"db2":  {Code: SkipVault, Detail: "ansible_host"},
		"win1": {Code: SkipConnection, Detail: "winrm"},
	}
	if !reflect.DeepEqual(skipped, wantSkip) {
		t.Errorf("skipped = %+v, want %+v", skipped, wantSkip)
	}
	if want := []string{"ungrouped", "web", "db", "win", "prod"}; !reflect.DeepEqual(inv.Groups, want) {
		t.Errorf("groups = %v, want %v", inv.Groups, want)
	}
}

func hasNote(h Host, code string) bool {
	for _, n := range h.Notes {
		if n.Code == code {
			return true
		}
	}
	return false
}

func TestPasswordNeverKept(t *testing.T) {
	inv := parse(t, "h1 ansible_ssh_pass=s3cret ansible_become_pass='an0ther'\n[g:vars]\nansible_password=gr0up\n[g]\nh2\n")
	for _, h := range inv.Hosts {
		for _, n := range h.Notes {
			if strings.Contains(n.Detail, "s3cret") || strings.Contains(n.Detail, "an0ther") || strings.Contains(n.Detail, "gr0up") {
				t.Fatalf("a password leaked into %+v", h)
			}
		}
		if !hasNote(h, NotePassword) {
			t.Errorf("%s: no password note", h.Name)
		}
	}
	// A malformed line does not quote the word back.
	_, err := Parse(strings.NewReader("h1 ansible_password s3cret\n"), "")
	if err == nil || strings.Contains(err.Error(), "s3cret") {
		t.Errorf("err = %v", err)
	}
}

func TestParseCases(t *testing.T) {
	tests := []struct {
		name, src string
		host      string
		want      Host
		skip      *Issue
	}{
		{name: "port in name", src: "db.example.com:5309\n", host: "db.example.com",
			want: Host{Address: "db.example.com", Port: 5309}},
		{name: "old ssh names", src: "h ansible_ssh_host=1.2.3.4 ansible_ssh_user=u ansible_ssh_port=2200\n", host: "h",
			want: Host{Address: "1.2.3.4", User: "u", Port: 2200}},
		{name: "relative key", src: "h ansible_ssh_private_key_file=keys/id_ed25519\n", host: "h",
			want: Host{Address: "h", Port: 22, KeyFile: filepath.Join("/inv", "keys/id_ed25519")}},
		{name: "home key kept", src: "h ansible_private_key_file=~/.ssh/id_rsa\n", host: "h",
			want: Host{Address: "h", Port: 22, KeyFile: "~/.ssh/id_rsa"}},
		{name: "jump flag", src: `h ansible_ssh_common_args="-J me@gw:2022"` + "\n", host: "h",
			want: Host{Address: "h", Port: 22, ProxyJump: "me@gw:2022"}},
		{name: "comment after vars", src: "h ansible_user=a # owner: team\n", host: "h",
			want: Host{Address: "h", User: "a", Port: 22}},
		{name: "jinja", src: `h ansible_host="{{ lookup('env','IP') }}"` + "\n", host: "h",
			skip: &Issue{Code: SkipJinja, Detail: "ansible_host"}},
		{name: "local", src: "localhost ansible_connection=local\n", host: "localhost",
			skip: &Issue{Code: SkipConnection, Detail: "local"}},
		{name: "docker", src: "c ansible_connection=community.docker.docker\n", host: "c",
			skip: &Issue{Code: SkipConnection, Detail: "community.docker.docker"}},
		{name: "bad port", src: "h ansible_port=ssh\n", host: "h",
			skip: &Issue{Code: SkipPort, Detail: "ssh"}},
		{name: "huge range", src: "n[1:999]\n", host: "n[1:999]",
			skip: &Issue{Code: SkipRange, Detail: "n[1:999]"}},
		{name: "builtin ssh", src: "h ansible_connection=ansible.builtin.ssh\n", host: "h",
			want: Host{Address: "h", Port: 22}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inv := parse(t, tt.src)
			if tt.skip != nil {
				if len(inv.Skipped) != 1 || inv.Skipped[0].Name != tt.host || inv.Skipped[0].Issue != *tt.skip {
					t.Fatalf("skipped = %+v, want %s %+v", inv.Skipped, tt.host, *tt.skip)
				}
				return
			}
			got, ok := byName(inv)[tt.host]
			if !ok {
				t.Fatalf("no host %q in %+v (skipped %+v)", tt.host, inv.Hosts, inv.Skipped)
			}
			got.Name, got.Groups = "", nil
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestExpand(t *testing.T) {
	tests := map[string][]string{
		"web[01:03].x":   {"web01.x", "web02.x", "web03.x"},
		"n[1:7:3]":       {"n1", "n4", "n7"},
		"db-[a:c]":       {"db-a", "db-b", "db-c"},
		"r[1:2]-[a:b]":   {"r1-a", "r1-b", "r2-a", "r2-b"},
		"plain.host":     {"plain.host"},
		"z[:2]":          {"z0", "z1", "z2"},
		"pad[008:010].x": {"pad008.x", "pad009.x", "pad010.x"},
	}
	for in, want := range tests {
		got, err := expand(in)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("expand(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	for _, bad := range []string{"a[1:", "a[3:1]", "a[1:2:0]", "a[x]", "a[1:300]", "a[1:20]-[1:20]"} {
		if _, err := expand(bad); err == nil {
			t.Errorf("expand(%q) = nil error", bad)
		}
	}
}

func TestSplit(t *testing.T) {
	got, err := split(`h a="x y" b='it''s' c=d\ e #rest`)
	want := []string{"h", "a=x y", "b=its", "c=d e"}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("split = %q, %v; want %q", got, err, want)
	}
	if _, err := split(`h a="open`); err == nil {
		t.Error("unclosed quote: nil error")
	}
}

func TestChildrenLoopAndDepth(t *testing.T) {
	// The deeper group wins over its parent, whatever the file order; a
	// loop in children does not hang.
	src := "[child:vars]\nansible_user=deep\n[parent:vars]\nansible_user=shallow\n" +
		"[parent:children]\nchild\n[child]\nh\n"
	h := byName(parse(t, src))["h"]
	if h.User != "deep" {
		t.Errorf("user = %q, want deep", h.User)
	}
	loop := "[a:children]\nb\n[b:children]\na\n[top:children]\na\n[b]\nh\n"
	h = byName(parse(t, loop))["h"]
	if !reflect.DeepEqual(h.Groups, []string{"a", "b", "top"}) {
		t.Errorf("groups = %v", h.Groups)
	}
}

func write(t *testing.T, name, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func code(err error) apperr.Code {
	var ae *apperr.Error
	if errors.As(err, &ae) {
		return ae.Code
	}
	return ""
}

func TestLoadUnsupported(t *testing.T) {
	tests := map[string]struct {
		name, body string
		want       apperr.Code
	}{
		"yml extension":  {"hosts.yml", "[web]\nh\n", apperr.InventoryYAMLUnsupported},
		"yaml all":       {"hosts", "# x\nall:\n  hosts:\n    h:\n", apperr.InventoryYAMLUnsupported},
		"yaml doc start": {"hosts", "---\nweb:\n", apperr.InventoryYAMLUnsupported},
		"json":           {"hosts", `{"all": {}}`, apperr.InventoryYAMLUnsupported},
		"script":         {"inv.py", "#!/usr/bin/env python3\nprint('{}')\n", apperr.InventoryScriptUnsupported},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := Load(write(t, tt.name, tt.body), "me")
			if code(err) != tt.want {
				t.Errorf("err = %v, want %s", err, tt.want)
			}
		})
	}
	if _, err := Load(t.TempDir(), "me"); code(err) != apperr.InvalidInput {
		t.Errorf("folder: err = %v", err)
	}
}

func TestLoadDefaultsUser(t *testing.T) {
	inv, err := Load(write(t, "hosts", "a ansible_user=root\nb:2222\n"), "me")
	if err != nil {
		t.Fatal(err)
	}
	h := byName(inv)
	if h["a"].User != "root" || h["b"].User != "me" || h["b"].Port != 2222 {
		t.Errorf("hosts = %+v", inv.Hosts)
	}
}

func TestCfgInventory(t *testing.T) {
	p := write(t, "ansible.cfg", "[ssh_connection]\ninventory = no\n[defaults]\nremote_user = x\ninventory = ~/inv/hosts, other\n")
	if got := cfgInventory(p); got != "~/inv/hosts" {
		t.Errorf("cfgInventory = %q", got)
	}
}

type memStore struct{ servers []store.Server }

func (m *memStore) Servers(context.Context) ([]store.Server, error) { return m.servers, nil }

func (m *memStore) Save(_ context.Context, s store.Server) (store.Server, error) {
	s.ID = s.Name
	m.servers = append(m.servers, s)
	return s, nil
}

func TestImport(t *testing.T) {
	inv := parse(t, sample)
	for i := range inv.Hosts {
		if inv.Hosts[i].User == "" {
			inv.Hosts[i].User = "me"
		}
	}
	m := &memStore{}
	added, err := Import(context.Background(), m, inv, []string{"web1.example.com", "db1"}, "me")
	if err != nil {
		t.Fatal(err)
	}
	// The jump host comes first, with the inventory's settings for it.
	var got []string
	for _, s := range added {
		got = append(got, s.Name+"="+s.User+"@"+s.Host)
	}
	want := []string{"bastion=jump@10.0.0.1", "web1.example.com=deploy@web1.example.com", "db1=ops@10.0.1.5"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("added = %v, want %v", got, want)
	}
	if added[1].JumpID != "bastion" || added[2].JumpID != "bastion" {
		t.Errorf("jump = %q %q", added[1].JumpID, added[2].JumpID)
	}

	// A second import adds nothing, and List marks them as saved.
	again, err := Import(context.Background(), m, inv, []string{"web1.example.com", "db1"}, "me")
	if err != nil || len(again) != 0 {
		t.Errorf("again = %v, %v", again, err)
	}
	l := List("/inv/hosts", inv, m.servers)
	for _, c := range l.Hosts {
		saved := c.Name == "bastion" || c.Name == "web1.example.com" || c.Name == "db1"
		if c.Exists != saved {
			t.Errorf("%s exists = %v", c.Name, c.Exists)
		}
	}

	if _, err := Import(context.Background(), m, inv, []string{"win1"}, "me"); code(err) != apperr.NotFound {
		t.Errorf("skipped host: err = %v", err)
	}
}
