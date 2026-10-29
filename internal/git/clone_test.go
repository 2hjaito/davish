package git

import (
	"strings"
	"testing"
)

func TestParseURL(t *testing.T) {
	for u, want := range map[string]string{
		"git@github.com:owner/app.git":           "github.com app ssh",
		"ssh://git@gitlab.com:2222/g/sub/x.git":  "gitlab.com x ssh",
		"https://github.com/owner/app":           "github.com app https",
		"https://deploy@git.example.com/a/b.git": "git.example.com b https",
	} {
		host, name, ssh, err := parseURL(u)
		kind := "https"
		if ssh {
			kind = "ssh"
		}
		if err != nil || host+" "+name+" "+kind != want {
			t.Errorf("%s = %s %s %s, %v", u, host, name, kind, err)
		}
	}
	for _, u := range []string{"", "file:///etc", "http://x.com/a", "-oProxyCommand=x", "github.com:a"} {
		if _, _, _, err := parseURL(u); err == nil {
			t.Errorf("%q accepted", u)
		}
	}
}

func TestPlanClone(t *testing.T) {
	p, err := PlanClone(CloneSpec{URL: "git@github.com:owner/app.git", Dir: "/var/www/app", Branch: "main"})
	if err != nil {
		t.Fatal(err)
	}
	if !p.Pinned || !strings.HasPrefix(p.Command, "mkdir -p ~/.ssh && { ssh-keygen -F github.com > /dev/null 2>&1 || printf ") ||
		!strings.HasSuffix(p.Command, "; } && git clone --branch main -- git@github.com:owner/app.git /var/www/app") {
		t.Errorf("command = %s", p.Command)
	}
	p, _ = PlanClone(CloneSpec{URL: "https://github.com/owner/app", Dir: "/srv/app x"})
	if p.Command != "git clone -- https://github.com/owner/app '/srv/app x'" || p.Pinned {
		t.Errorf("https = %+v", p)
	}
	if p, _ := PlanClone(CloneSpec{URL: "https://github.com/o/site.git"}); p.Name != "site" || p.Command != "" {
		t.Errorf("name only = %+v", p)
	}
	if _, err := PlanClone(CloneSpec{URL: "https://github.com/o/a", Dir: "/a", Branch: "-x"}); err == nil {
		t.Error("bad branch accepted")
	}
}

func TestInitCommand(t *testing.T) {
	c, err := InitCommand("/srv/a", "", "git@github.com:o/a.git")
	want := "mkdir -p -- /srv/a && cd -- /srv/a && git init && { cd -- /srv/a && git remote set-url origin git@github.com:o/a.git 2>/dev/null || cd -- /srv/a && git remote add origin git@github.com:o/a.git; }"
	if err != nil || c != want {
		t.Errorf("init = %s, %v", c, err)
	}
}
