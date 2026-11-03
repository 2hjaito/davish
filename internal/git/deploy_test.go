package git

import (
	"regexp"
	"strings"
	"testing"
)

func TestDeployLine(t *testing.T) {
	got, err := deployLine("/var/www/app", "abc123", []Step{
		{Label: "Get new code", Command: "git pull --ff-only"},
		{Command: ""},
		{Label: "Build", Command: "# build"},
		{Command: "npm ci && npm run build"},
	})
	want := `__davish_mark() { printf '\033]7761;davish-deploy;%s\007' "$1"; }; printf '\033[H\033[2J'; ` +
		`cd -- /var/www/app && __davish_mark 'abc123;begin;2'` +
		` && echo '▶ 1/2  Get new code  (git pull --ff-only)' && __davish_mark 'abc123;step;1' && { git pull --ff-only; }` +
		` && echo '▶ 2/2  npm ci && npm run build' && __davish_mark 'abc123;step;2' && { npm ci && npm run build; }` +
		` && __davish_mark "abc123;end;0;$(git -C /var/www/app rev-parse --short HEAD 2>/dev/null)"` +
		` && echo '✓ Deploy finished'` +
		` || { __davish_mark "abc123;end;$?"; echo '✗ Deploy stopped: the step above failed'; }; unset -f __davish_mark`
	if err != nil || got.Line != want {
		t.Errorf("line =\n%s\nwant\n%s", got.Line, want)
	}
	if strings.Join(got.Steps, "|") != "Get new code|npm ci && npm run build" || got.Sudo {
		t.Errorf("steps = %q, sudo %v", got.Steps, got.Sudo)
	}
	for name, steps := range map[string][]Step{
		"empty recipe":   {{Command: "# only a comment"}},
		"two-line step":  {{Command: "a\nb"}},
		"two-line label": {{Label: "a\nb", Command: "x"}},
	} {
		if _, err := deployLine("/a", "n", steps); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
	if _, err := deployLine("rel", "n", []Step{{Command: "x"}}); err == nil {
		t.Error("relative folder accepted")
	}
}

// A step may hold quotes or $(): its heading is quoted, and the folder in
// the commit lookup is quoted even inside the double-quoted mark.
func TestDeployLineQuoting(t *testing.T) {
	got, err := deployLine("/srv/my app", "n", []Step{{Command: `echo "it's $(date)"`}})
	if err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{
		`cd -- '/srv/my app'`,
		`echo '▶ 1/1  echo "it'\''s $(date)"'`,
		`$(git -C '/srv/my app' rev-parse`,
	} {
		if !strings.Contains(got.Line, part) {
			t.Errorf("line lacks %s:\n%s", part, got.Line)
		}
	}
}

func TestDeployLineNonce(t *testing.T) {
	a, err := DeployLine("/a", []Step{{Command: "x"}})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := DeployLine("/a", []Step{{Command: "x"}})
	if !regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(a.Nonce) || a.Nonce == b.Nonce {
		t.Errorf("nonces %q, %q", a.Nonce, b.Nonce)
	}
}

func TestDeployLineSudo(t *testing.T) {
	for cmd, want := range map[string]bool{
		"sudo systemctl restart app":  true,
		"npm ci && sudo make install": true,
		"(sudo true)":                 true,
		"echo pseudo thing":           false,
		"./sudoers-check":             false,
	} {
		d, err := deployLine("/a", "n", []Step{{Command: cmd}})
		if err != nil || d.Sudo != want {
			t.Errorf("%q: sudo %v, want %v (%v)", cmd, d.Sudo, want, err)
		}
	}
}
