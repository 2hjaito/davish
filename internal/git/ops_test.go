package git

import (
	"strings"
	"testing"

	"davish/internal/apperr"
)

func TestCommand(t *testing.T) {
	tests := []struct {
		op, branch, want string
	}{
		{OpFetch, "", "cd -- '/srv/my app' && env GIT_TERMINAL_PROMPT=0 timeout 120 git fetch --prune"},
		{OpPull, "", "cd -- '/srv/my app' && env GIT_TERMINAL_PROMPT=0 timeout 120 git pull --ff-only"},
		{OpSwitch, "feature/login", "cd -- '/srv/my app' && git switch feature/login"},
	}
	for _, tt := range tests {
		got, err := command("/srv/my app", tt.op, tt.branch)
		if err != nil || got != tt.want {
			t.Errorf("%s: got %q, %v; want %q", tt.op, got, err, tt.want)
		}
	}
	for _, b := range []string{"", "-f", "--orphan=x", "a..b", "a b", "x;reboot", "$(id)", "main.lock", "a/", "a//b", "a@{1}", "a\nb"} {
		if _, err := command("/srv/x", OpSwitch, b); err == nil {
			t.Errorf("branch %q accepted", b)
		}
	}
	if _, err := command("/srv/x", "push", ""); err == nil {
		t.Error("push accepted")
	}
}

func TestParseBranches(t *testing.T) {
	out := strings.Join([]string{
		" \trefs/heads/develop",
		"*\trefs/heads/main",
		" \trefs/remotes/origin/HEAD",
		" \trefs/remotes/origin/main",
		" \trefs/remotes/origin/feature/login",
		" \trefs/remotes/upstream/feature/login",
		"",
	}, "\n")
	got := parseBranches(out)
	want := []Branch{
		{Name: "develop"},
		{Name: "main", Current: true},
		{Name: "feature/login", Remote: true},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("branch %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestOpError(t *testing.T) {
	tests := []struct {
		out  string
		exit int
		want apperr.Code
	}{
		{"git@github.com: Permission denied (publickey).\nfatal: Could not read from remote repository.", 128, apperr.GitAuthFailed},
		{"fatal: could not read Username for 'https://github.com': terminal prompts disabled", 128, apperr.GitAuthFailed},
		{"hint: Diverging branches can't be fast-forwarded\nfatal: Not possible to fast-forward, aborting.", 128, apperr.GitNotFastForward},
		{"There is no tracking information for the current branch.", 1, apperr.GitNoUpstream},
		{"error: Your local changes to the following files would be overwritten by checkout:", 1, apperr.GitDirty},
		{"ssh: Could not resolve host: gitlab.example.com", 128, apperr.GitRemoteFailed},
		{"", 124, apperr.GitRemoteFailed},
		{"fatal: something new", 128, apperr.CmdFailed},
	}
	for _, tt := range tests {
		if got := opError(tt.exit, tt.out); got != tt.want {
			t.Errorf("%q: got %s, want %s", tt.out, got, tt.want)
		}
	}
}

func TestRedact(t *testing.T) {
	in := "fatal: unable to access 'https://deploy:ghp_secret123@github.com/acme/shop.git/': 403"
	got := redact(in)
	if strings.Contains(got, "ghp_secret123") || !strings.Contains(got, "https://***@github.com/acme/shop.git") {
		t.Fatalf("got %q", got)
	}
	if s := "git@github.com:acme/shop.git"; redact(s) != s {
		t.Fatalf("scp-like remote changed: %q", redact(s))
	}
}

func TestInstallFor(t *testing.T) {
	tests := []struct {
		files []string
		want  string
	}{
		{nil, ""},
		{[]string{"package-lock.json"}, ""},
		{[]string{"package.json"}, ""},
		{[]string{"package.json", "package-lock.json"}, "npm ci"},
		{[]string{"package.json", "yarn.lock"}, "yarn install --frozen-lockfile"},
		{[]string{"package.json", "pnpm-lock.yaml", "package-lock.json"}, "pnpm install --frozen-lockfile"},
	}
	for _, tt := range tests {
		if got := installFor(tt.files); got != tt.want {
			t.Errorf("%v: got %q, want %q", tt.files, got, tt.want)
		}
	}
}
