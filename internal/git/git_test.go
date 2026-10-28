package git

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"davish/internal/apperr"
	"davish/internal/audit"
)

func TestFindLineQuotesFolders(t *testing.T) {
	line := findLine([]string{"/srv/my site", "/x/it's", "/x/$(reboot)", "/x/`id`"})
	for _, want := range []string{
		`find "$HOME" /var/www /srv /opt '/srv/my site' '/x/it'\''s' '/x/$(reboot)' '/x/` + "`id`" + `' -mindepth 1`,
		`-maxdepth 4`,
		`| head -n 500; true`,
	} {
		if !strings.Contains(line, want) {
			t.Errorf("line %q lacks %q", line, want)
		}
	}
}

func TestCheckExtra(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
		ok   bool
	}{
		{"none", nil, []string{}, true},
		{"cleaned", []string{"/home/deploy/apps/", "/srv//x/../y"}, []string{"/home/deploy/apps", "/srv/y"}, true},
		{"relative", []string{"apps"}, nil, false},
		{"option", []string{"-delete"}, nil, false},
		{"newline", []string{"/srv/a\n/b"}, nil, false},
		{"too many", make([]string, maxExtra+1), nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := checkExtra(tt.in)
			if !tt.ok {
				var ae *apperr.Error
				if !errors.As(err, &ae) || ae.Code != apperr.InvalidInput {
					t.Fatalf("err = %v, want INVALID_INPUT", err)
				}
				return
			}
			if err != nil || strings.Join(got, ",") != strings.Join(tt.want, ",") {
				t.Fatalf("got %q, %v; want %q", got, err, tt.want)
			}
		})
	}
}

func TestParseFind(t *testing.T) {
	out := strings.Join([]string{
		"/var/www/shop/.git",
		"/home/deploy/api/.git",
		"/srv/home/deploy/api/.git",
		"/home/deploy/api/.git",
		"/opt/tool/.git\r",
		"find: '/root': Permission denied",
		"/.git",
		"",
	}, "\n")
	got := parseFind([]byte(out))
	want := []Repo{
		{"/home/deploy/api", "api"},
		{"/opt/tool", "tool"},
		{"/srv/home/deploy/api", "api"},
		{"/var/www/shop", "shop"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("repo %d = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestParseStatus(t *testing.T) {
	out := strings.Join([]string{
		"# branch.oid 1f2e3d",
		"# branch.head main",
		"# branch.upstream origin/main",
		"# branch.ab +2 -3",
		"1 .M N... 100644 100644 100644 aaa aaa src/app file.go",
		"1 A. N... 000000 100644 100644 000 bbb new.txt",
		"2 R. N... 100644 100644 100644 ccc ccc R100 docs/new name.md",
		"docs/old name.md",
		"u UU N... 100644 100644 100644 100644 d1 d2 d3 merge.go",
		"? build/out.log",
		"! ignored.tmp",
		"",
	}, "\x00")
	st := parseStatus([]byte(out))
	if st.Branch != "main" || st.Upstream != "origin/main" || st.Ahead != 2 || st.Behind != 3 {
		t.Fatalf("branch %+v", st)
	}
	want := []Change{
		{Path: "src/app file.go", Index: ".", Worktree: "M"},
		{Path: "new.txt", Index: "A", Worktree: "."},
		{Path: "docs/new name.md", OrigPath: "docs/old name.md", Index: "R", Worktree: "."},
		{Path: "merge.go", Index: "U", Worktree: "U", Conflict: true},
		{Path: "build/out.log", Index: "?", Worktree: "?"},
	}
	if len(st.Changes) != len(want) {
		t.Fatalf("changes %+v", st.Changes)
	}
	for i := range want {
		if st.Changes[i] != want[i] {
			t.Errorf("change %d = %+v, want %+v", i, st.Changes[i], want[i])
		}
	}
}

func TestParseStatusDetachedAndTruncated(t *testing.T) {
	entries := []string{"# branch.oid abc", "# branch.head (detached)"}
	for i := 0; i < maxChanges+5; i++ {
		entries = append(entries, "? f"+strconv.Itoa(i))
	}
	st := parseStatus([]byte(strings.Join(entries, "\x00")))
	if st.Branch != "" || len(st.Changes) != maxChanges || !st.Truncated {
		t.Fatalf("branch %q, %d changes, truncated %v", st.Branch, len(st.Changes), st.Truncated)
	}
}

func TestParseCommit(t *testing.T) {
	c := parseCommit("abc123\tabc\tAn Nguyễn\t1700000000\tfix: a\ttab in subject\n", "\t")
	if c == nil || c.Author != "An Nguyễn" || c.Time != 1700000000 || c.Subject != "fix: a\ttab in subject" {
		t.Fatalf("got %+v", c)
	}
	if parseCommit("", "\t") != nil {
		t.Fatal("empty line is no commit")
	}
}

func TestDiffArgs(t *testing.T) {
	tests := []struct {
		name string
		r    DiffRequest
		want string
	}{
		{"commit", DiffRequest{Commit: "abc1234"}, "show --format= --no-color --no-ext-diff --no-textconv abc1234 --"},
		{"file", DiffRequest{File: "-rf name"}, "diff --no-color --no-ext-diff --no-textconv HEAD -- -rf name"},
		{"untracked", DiffRequest{File: "new.txt", Untracked: true}, "diff --no-index --no-color --no-ext-diff --no-textconv -- /dev/null new.txt"},
		{"bad hash", DiffRequest{Commit: "HEAD; reboot"}, ""},
		{"option hash", DiffRequest{Commit: "--output=/tmp/x"}, ""},
		{"no file", DiffRequest{}, ""},
		{"absolute", DiffRequest{File: "/etc/passwd"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, err := diffArgs(tt.r)
			if tt.want == "" {
				if err == nil {
					t.Fatalf("args %q, want an error", args)
				}
				return
			}
			if err != nil || strings.Join(args, " ") != tt.want {
				t.Fatalf("got %q, %v", strings.Join(args, " "), err)
			}
		})
	}
}

func TestGitError(t *testing.T) {
	tests := []struct {
		stderr string
		exit   int
		want   apperr.Code
	}{
		{"fatal: detected dubious ownership in repository at '/var/www/x'", 128, apperr.GitUnsafeOwner},
		{"fatal: not a git repository (or any of the parent directories): .git", 128, apperr.NotFound},
		{"sh: 1: cd: can't cd to /nope", 2, apperr.NotFound},
		{"", 127, apperr.CmdMissing},
		{"fatal: something else", 128, apperr.CmdFailed},
	}
	for _, tt := range tests {
		var ae *apperr.Error
		err := gitError(audit.Result{Stderr: []byte(tt.stderr), Exit: tt.exit})
		if !errors.As(err, &ae) || ae.Code != tt.want {
			t.Errorf("%q: got %v, want %s", tt.stderr, err, tt.want)
		}
	}
}
