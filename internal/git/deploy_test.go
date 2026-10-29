package git

import "testing"

func TestDeployLine(t *testing.T) {
	got, err := DeployLine("/var/www/app", []string{"git pull --ff-only", "", "# build", "npm ci && npm run build"})
	want := "cd -- /var/www/app && echo '▶ 1/2  git pull --ff-only' && { git pull --ff-only; } && echo '▶ 2/2  npm ci && npm run build' && { npm ci && npm run build; } && echo '✓ Deploy finished' || echo '✗ Deploy stopped: the step above failed'"
	if err != nil || got != want {
		t.Errorf("line =\n%s\nwant\n%s", got, want)
	}
	if _, err := DeployLine("/a", []string{"# only a comment"}); err == nil {
		t.Error("empty recipe accepted")
	}
	if _, err := DeployLine("rel", []string{"x"}); err == nil {
		t.Error("relative folder accepted")
	}
}
