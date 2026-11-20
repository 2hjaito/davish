package git

import "testing"

func TestParseIdentity(t *testing.T) {
	out := `@@user
root
@@home
/root
@@host
stoic-merkle
@@gitname
Dang Tran
@@gitemail
dang@example.com
@@helper
@@keys
256 SHA256:abc root@stoic-merkle (ED25519)
256 SHA256:abc root@stoic-merkle (ED25519)
3072 SHA256:def no comment (RSA)
not a key line
@@host:github.com
Hi theptaydo! You've successfully authenticated, but GitHub does not provide shell access.
@@host:gitlab.com
git@gitlab.com: Permission denied (publickey).
`
	sec := sections(out)
	id := parseIdentity(sec)
	if id.User != "root" || id.Home != "/root" || id.Host != "stoic-merkle" || id.GitName != "Dang Tran" ||
		id.GitEmail != "dang@example.com" || id.Helper != "" {
		t.Errorf("identity = %+v", id)
	}
	if len(id.Keys) != 2 || id.Keys[0] != (SSHKey{Type: "ED25519", Bits: 256, Fingerprint: "SHA256:abc", Comment: "root@stoic-merkle"}) ||
		id.Keys[1].Comment != "" || id.Keys[1].Type != "RSA" {
		t.Errorf("keys = %+v", id.Keys)
	}
	want := []HostLogin{
		{Host: "github.com", State: LoginOK, Account: "theptaydo"},
		{Host: "gitlab.com", State: LoginDenied},
	}
	hosts := []HostLogin{parseLogin("github.com", sec["host:github.com"]), parseLogin("gitlab.com", sec["host:gitlab.com"])}
	if len(id.Hosts) != 0 || hosts[0] != want[0] || hosts[1] != want[1] {
		t.Errorf("hosts = %+v, %+v", id.Hosts, hosts)
	}
}

func TestParseLogin(t *testing.T) {
	for out, want := range map[string]HostLogin{
		"Hi theptaydo/profile-v2! You've successfully authenticated, but GitHub does not provide shell access.": {
			Host: "h", State: LoginOK, Account: "theptaydo/profile-v2", Deploy: true,
		},
		"Welcome to GitLab, @dang!":                            {Host: "h", State: LoginOK, Account: "dang"},
		"ssh: connect to host h port 22: Connection timed out": {Host: "h", State: LoginUnreachable},
		"": {Host: "h", State: LoginUnreachable},
	} {
		if got := parseLogin("h", out); got != want {
			t.Errorf("%q = %+v, want %+v", out, got, want)
		}
	}
}
