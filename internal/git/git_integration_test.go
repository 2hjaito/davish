//go:build integration

package git_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"davish/internal/apperr"
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

func TestStatusLogDiff(t *testing.T) {
	m, id := testsshd.Connect(t)
	s := git.New(audit.NewRunner(m, audit.New(discard{})))
	ctx := context.Background()
	testsshd.Run(t, m, id, `set -e; rm -rf ~/gitstat; mkdir -p ~/gitstat; cd ~/gitstat
g() { git -c user.name=Tester -c user.email=t@example.com "$@"; }
git init -q --bare -b main remote.git
g clone -q remote.git work 2>/dev/null; cd work
echo one > 'a file.txt'; g add .; g commit -qm 'first commit'; g push -q origin HEAD:main 2>/dev/null
g branch -q --set-upstream-to=origin/main
echo two >> 'a file.txt'; g commit -qam 'second commit'
echo three >> 'a file.txt'; echo new > untracked.txt`)
	t.Cleanup(func() { testsshd.Run(t, m, id, `rm -rf ~/gitstat`) })
	dir := "/home/tester/gitstat/work"

	st, err := s.Status(ctx, id, dir)
	if err != nil {
		t.Fatal(err)
	}
	if st.Branch != "main" || st.Upstream != "origin/main" || st.Ahead != 1 || st.Behind != 0 {
		t.Fatalf("status %+v", st)
	}
	if st.Last == nil || st.Last.Subject != "second commit" || st.Last.Author != "Tester" {
		t.Fatalf("last %+v", st.Last)
	}
	if len(st.Changes) != 2 || st.Changes[0].Path != "a file.txt" || st.Changes[0].Worktree != "M" || st.Changes[1].Index != "?" {
		t.Fatalf("changes %+v", st.Changes)
	}

	commits, err := s.Log(ctx, id, dir, 0)
	if err != nil || len(commits) != 2 || commits[1].Subject != "first commit" {
		t.Fatalf("log %+v, %v", commits, err)
	}

	d, err := s.Diff(ctx, id, git.DiffRequest{Dir: dir, File: "a file.txt"})
	if err != nil || !strings.Contains(d.Text, "+three") || strings.Contains(d.Text, "+two") {
		t.Fatalf("file diff %q, %v", d.Text, err)
	}
	d, err = s.Diff(ctx, id, git.DiffRequest{Dir: dir, File: "untracked.txt", Untracked: true})
	if err != nil || !strings.Contains(d.Text, "+new") {
		t.Fatalf("untracked diff %q, %v", d.Text, err)
	}
	d, err = s.Diff(ctx, id, git.DiffRequest{Dir: dir, Commit: commits[0].Hash})
	if err != nil || !strings.Contains(d.Text, "+two") {
		t.Fatalf("commit diff %q, %v", d.Text, err)
	}

	if _, err := s.Status(ctx, id, "/home/tester"); !isCode(err, apperr.NotFound) {
		t.Fatalf("status outside a repo: %v", err)
	}
}

func isCode(err error, code apperr.Code) bool {
	var ae *apperr.Error
	return errors.As(err, &ae) && ae.Code == code
}
