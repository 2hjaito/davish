//go:build integration

package git_test

import (
	"context"
	"testing"

	"davish/internal/audit"
	"davish/internal/git"
	"davish/internal/testsshd"
)

type discard struct{}

func (discard) Emit(string, any) {}

func TestFindRepos(t *testing.T) {
	m, id := testsshd.Connect(t)
	s := git.New(audit.NewRunner(m, audit.New(discard{})))
	testsshd.Run(t, m, id, `set -e; rm -rf ~/gitfind '/tmp/git find'
mkdir -p ~/gitfind/site ~/gitfind/app/node_modules/dep ~/gitfind/.hidden/repo '/tmp/git find/extra' ~/gitfind/a/b/c/d/too-deep
for d in ~/gitfind/site ~/gitfind/app/node_modules/dep ~/gitfind/.hidden/repo '/tmp/git find/extra' ~/gitfind/a/b/c/d/too-deep; do git init -q "$d"; done`)
	t.Cleanup(func() { testsshd.Run(t, m, id, `rm -rf ~/gitfind '/tmp/git find'`) })

	repos, err := s.Find(context.Background(), id, []string{"/tmp/git find"})
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, r := range repos {
		found[r.Path] = true
	}
	for _, want := range []string{"/home/tester/gitfind/site", "/tmp/git find/extra"} {
		if !found[want] {
			t.Errorf("missing %s in %v", want, repos)
		}
	}
	for _, skip := range []string{"/home/tester/gitfind/app/node_modules/dep", "/home/tester/gitfind/.hidden/repo", "/home/tester/gitfind/a/b/c/d/too-deep"} {
		if found[skip] {
			t.Errorf("%s should be skipped", skip)
		}
	}
}
