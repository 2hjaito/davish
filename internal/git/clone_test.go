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

func TestPlanCloneQuiet(t *testing.T) {
	p, err := planClone(CloneSpec{URL: "https://github.com/o/a.git", Dir: "/srv/a"}, true)
	if err != nil || p.Command != "env GIT_TERMINAL_PROMPT=0 timeout 900 git clone -- https://github.com/o/a.git /srv/a" {
		t.Errorf("quiet = %+v, %v", p, err)
	}
	p, _ = planClone(CloneSpec{URL: "git@github.com:o/a.git", Dir: "/srv/a"}, true)
	if !strings.HasSuffix(p.Command, "; } && env GIT_TERMINAL_PROMPT=0 timeout 900 git clone -- git@github.com:o/a.git /srv/a") {
		t.Errorf("quiet ssh = %s", p.Command)
	}
}

func TestCloneError(t *testing.T) {
	for out, want := range map[string]string{
		"fatal: destination path '/srv/a' already exists and is not an empty directory.":                             "FS_EXISTS",
		"fatal: could not create leading directories of '/x/a': Permission denied":                                   "FS_PERMISSION_DENIED",
		"git@github.com: Permission denied (publickey).\nfatal: Could not read from remote":                          "GIT_AUTH_FAILED",
		"fatal: could not read Username for 'https://github.com': terminal prompts disabled":                         "GIT_AUTH_FAILED",
		"warning: Could not find remote branch dev to clone.\nfatal: Remote branch dev not found in upstream origin": "NOT_FOUND",
	} {
		if got := cloneError(128, out); string(got) != want {
			t.Errorf("%q = %s, want %s", out, got, want)
		}
	}
}
