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
	if len(commits[0].Parents) != 1 || commits[0].Parents[0] != commits[1].Hash ||
		len(commits[0].Refs) != 1 || commits[0].Refs[0] != (git.Ref{Name: "main", Kind: git.RefHead}) ||
		len(commits[1].Refs) != 1 || commits[1].Refs[0] != (git.Ref{Name: "origin/main", Kind: git.RefRemote}) {
		t.Fatalf("graph %+v", commits)
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

func TestFetchPullSwitch(t *testing.T) {
	m, id := testsshd.Connect(t)
	s := git.New(audit.NewRunner(m, audit.New(discard{})))
	ctx := context.Background()
	testsshd.Run(t, m, id, `set -e; rm -rf ~/gitops; mkdir -p ~/gitops; cd ~/gitops
g() { git -c user.name=Tester -c user.email=t@example.com "$@"; }
git init -q --bare -b main remote.git
g clone -q remote.git work 2>/dev/null; cd work
echo '{}' > package.json; echo '{}' > package-lock.json
g add .; g commit -qm one; g push -q -u origin main 2>/dev/null
cd ..; g clone -q remote.git other 2>/dev/null; cd other
echo two > two.txt; g add .; g commit -qm two; g push -q origin main 2>/dev/null
g switch -qc feature; g push -q origin feature 2>/dev/null`)
	t.Cleanup(func() { testsshd.Run(t, m, id, `rm -rf ~/gitops`) })
	dir := "/home/tester/gitops/work"

	run := func(op, branch string) git.Result {
		t.Helper()
		plan, err := s.Plan(ctx, id, dir, op, branch)
		if err != nil {
			t.Fatalf("plan %s: %v", op, err)
		}
		res, err := s.Run(ctx, id, dir, op, branch, plan.Command)
		if err != nil {
			t.Fatalf("run %s: %v", op, err)
		}
		return res
	}

	if r := run(git.OpFetch, ""); r.Error != "" {
		t.Fatalf("fetch: %+v", r)
	}
	st, err := s.Status(ctx, id, dir)
	if err != nil || st.Behind != 1 || st.Install != "npm ci" {
		t.Fatalf("after fetch %+v, %v", st, err)
	}

	// A dirty tree blocks the pull, even with a plan confirmed while clean.
	plan, err := s.Plan(ctx, id, dir, git.OpPull, "")
	if err != nil || plan.Dirty != 0 {
		t.Fatalf("plan %+v, %v", plan, err)
	}
	testsshd.Run(t, m, id, `echo '{"x":1}' > ~/gitops/work/package.json`)
	if _, err := s.Run(ctx, id, dir, git.OpPull, "", plan.Command); !isCode(err, apperr.GitDirty) {
		t.Fatalf("dirty pull: %v", err)
	}
	if p, _ := s.Plan(ctx, id, dir, git.OpPull, ""); p.Dirty != 1 {
		t.Fatalf("dirty count %d", p.Dirty)
	}
	testsshd.Run(t, m, id, `cd ~/gitops/work && git checkout -q -- package.json`)

	if r := run(git.OpPull, ""); r.Error != "" {
		t.Fatalf("pull: %+v", r)
	}
	if st, _ := s.Status(ctx, id, dir); st.Behind != 0 || st.Last == nil || st.Last.Subject != "two" {
		t.Fatalf("after pull %+v", st)
	}

	branches, err := s.Branches(ctx, id, dir)
	if err != nil || len(branches) != 2 || !branches[0].Current || branches[1].Name != "feature" || !branches[1].Remote {
		t.Fatalf("branches %+v, %v", branches, err)
	}
	if r := run(git.OpSwitch, "feature"); r.Error != "" {
		t.Fatalf("switch: %+v", r)
	}
	if st, _ := s.Status(ctx, id, dir); st.Branch != "feature" || st.Upstream != "origin/feature" {
		t.Fatalf("after switch %+v", st)
	}

	if _, err := s.Run(ctx, id, dir, git.OpSwitch, "main", "git switch main; reboot"); !isCode(err, apperr.FSStale) {
		t.Fatalf("unconfirmed command: %v", err)
	}

	// A remote that refuses the key reports GIT_AUTH_FAILED.
	testsshd.Run(t, m, id, `cd ~/gitops/work && git remote set-url origin nobody@127.0.0.1:/nowhere.git`)
	if r := run(git.OpFetch, ""); r.Error != apperr.GitAuthFailed {
		t.Fatalf("auth failure: %+v", r)
	}
}
