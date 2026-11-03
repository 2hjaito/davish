package git

import (
	"regexp"
	"strings"
	"testing"
)

func TestDeployLine(t *testing.T) {
	got, err := deployLine("/var/www/app", "abc123", []string{"git pull --ff-only", "", "# build", "npm ci && npm run build"})
	want := `__davish_mark() { printf '\033]7761;davish-deploy;%s\007' "$1"; }; ` +
		`cd -- /var/www/app && __davish_mark 'abc123;begin;2'` +
		` && echo '▶ 1/2  git pull --ff-only' && __davish_mark 'abc123;step;1' && { git pull --ff-only; }` +
		` && echo '▶ 2/2  npm ci && npm run build' && __davish_mark 'abc123;step;2' && { npm ci && npm run build; }` +
		` && __davish_mark "abc123;end;0;$(git -C /var/www/app rev-parse --short HEAD 2>/dev/null)"` +
		` && echo '✓ Deploy finished'` +
		` || { __davish_mark "abc123;end;$?"; echo '✗ Deploy stopped: the step above failed'; }; unset -f __davish_mark`
	if err != nil || got.Line != want {
		t.Errorf("line =\n%s\nwant\n%s", got.Line, want)
	}
	if strings.Join(got.Steps, "|") != "git pull --ff-only|npm ci && npm run build" {
		t.Errorf("steps = %q", got.Steps)
	}
	if _, err := deployLine("/a", "n", []string{"# only a comment"}); err == nil {
		t.Error("empty recipe accepted")
	}
	if _, err := deployLine("rel", "n", []string{"x"}); err == nil {
		t.Error("relative folder accepted")
	}
	if _, err := deployLine("/a", "n", []string{"a\nb"}); err == nil {
		t.Error("two-line step accepted")
	}
}

// A step may hold quotes or $(): its heading is quoted, and the folder in
// the commit lookup is quoted even inside the double-quoted mark.
func TestDeployLineQuoting(t *testing.T) {
	got, err := deployLine("/srv/my app", "n", []string{`echo "it's $(date)"`})
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
	a, err := DeployLine("/a", []string{"x"})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := DeployLine("/a", []string{"x"})
	if !regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(a.Nonce) || a.Nonce == b.Nonce {
		t.Errorf("nonces %q, %q", a.Nonce, b.Nonce)
	}
}
