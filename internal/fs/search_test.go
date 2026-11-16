package fs

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"davish/internal/apperr"
	"davish/internal/audit"
)

func TestSearchLine(t *testing.T) {
	tests := []struct {
		name string
		q    SearchQuery
		want string
	}{
		{
			"name, case-insensitive",
			SearchQuery{Dir: "/etc", Query: "conf", Mode: SearchName},
			`{ timeout 30 find /etc -mindepth 1 -iname '*conf*' -print0; echo "exit:$?" >&2; } | head -c 524288`,
		},
		{
			"name, glob kept, match case",
			SearchQuery{Dir: "/srv/my app", Query: "*.log", Mode: SearchName, MatchCase: true},
			`{ timeout 30 find '/srv/my app' -mindepth 1 -name '*.log' -print0; echo "exit:$?" >&2; } | head -c 524288`,
		},
		{
			"name from the root prunes virtual folders",
			SearchQuery{Dir: "/", Query: "nginx", Mode: SearchName},
			`{ timeout 30 find / -mindepth 1 '(' -path /proc -o -path /sys -o -path /dev -o -path /run ')' -prune -o -iname '*nginx*' -print0; echo "exit:$?" >&2; } | head -c 524288`,
		},
		{
			"content, quotes the pattern",
			SearchQuery{Dir: "/var/www", Query: "it's a.b", Mode: SearchContent},
			`{ timeout 30 grep -r -I -H -n -o -Z -E -D skip -m 3 -i --exclude-dir=.git --exclude-dir=node_modules -e '.{0,80}it'\''s a\.b.{0,80}' -- /var/www; echo "exit:$?" >&2; } | head -c 524288`,
		},
		{
			"content, a leading dash is not an option",
			SearchQuery{Dir: "/home/u", Query: "-rf", Mode: SearchContent, MatchCase: true},
			`{ timeout 30 grep -r -I -H -n -o -Z -E -D skip -m 3 --exclude-dir=.git --exclude-dir=node_modules -e '.{0,80}-rf.{0,80}' -- /home/u; echo "exit:$?" >&2; } | head -c 524288`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := searchLine(tt.q)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("got\n%s\nwant\n%s", got, tt.want)
			}
		})
	}
}

func TestSearchLineRejects(t *testing.T) {
	tests := []SearchQuery{
		{Dir: "etc", Query: "x", Mode: SearchName},
		{Dir: "/etc/../root", Query: "x", Mode: SearchName},
		{Dir: "/etc", Query: "  ", Mode: SearchName},
		{Dir: "/etc", Query: "a\nb", Mode: SearchName},
		{Dir: "/etc", Query: "x", Mode: "regex"},
		{Dir: "/", Query: "x", Mode: SearchContent},
		{Dir: "/proc", Query: "x", Mode: SearchContent},
		{Dir: "/dev/shm", Query: "x", Mode: SearchContent},
	}
	for _, q := range tests {
		_, err := searchLine(q)
		var ae *apperr.Error
		if !errors.As(err, &ae) || ae.Code != apperr.InvalidInput {
			t.Errorf("searchLine(%+v) = %v; want INVALID_INPUT", q, err)
		}
	}
}

func TestQuoteERE(t *testing.T) {
	tests := map[string]string{
		"plain":       "plain",
		"a.b*c":       `a\.b\*c`,
		"(x|y)":       `\(x\|y\)`,
		"[0-9]{2}":    `\[0-9\]\{2\}`,
		`^$\`:         `\^\$\\`,
		"tiếng việt?": `tiếng việt\?`,
	}
	for in, want := range tests {
		if got := quoteERE(in); got != want {
			t.Errorf("quoteERE(%q) = %q; want %q", in, got, want)
		}
	}
}

func TestParseSearchNames(t *testing.T) {
	stdout := []byte("/etc/nginx\x00/etc/a b.conf\x00/etc/new\nline\x00/etc/tiếng.conf\x00")
	stderr := []byte("find: '/etc/ssl/private': Permission denied\nfind: '/etc/x': Permission denied\nexit:1\n")
	got := parseSearch(SearchName, stdout, stderr)
	want := []string{"/etc/nginx", "/etc/a b.conf", "/etc/new\nline", "/etc/tiếng.conf"}
	if len(got.Hits) != len(want) {
		t.Fatalf("hits = %+v", got.Hits)
	}
	for i, w := range want {
		if got.Hits[i].Path != w {
			t.Errorf("hit %d = %q; want %q", i, got.Hits[i].Path, w)
		}
	}
	if got.Denied != 2 || got.exit != 1 || got.Truncated || got.TimedOut {
		t.Errorf("got %+v", got)
	}
}

func TestParseSearchCapAndTimeout(t *testing.T) {
	var b bytes.Buffer
	for b.Len() < searchBytes {
		b.WriteString("/srv/file\x00")
	}
	out := b.Bytes()[:searchBytes] // cut mid-record, like head -c
	got := parseSearch(SearchName, out, []byte("exit:124\n"))
	if len(got.Hits) != SearchLimit || !got.Truncated || !got.TimedOut {
		t.Errorf("hits=%d truncated=%v timedOut=%v", len(got.Hits), got.Truncated, got.TimedOut)
	}
}

func TestParseGrep(t *testing.T) {
	stdout := []byte("/srv/app/main.go\x0012:\tlisten(\":8080\")\n" +
		"/srv/odd\nname.txt\x003:a:b:c\n" +
		"/srv/bad.txt\x00x:not a number\n" +
		"/srv/bin\x007:ctl\x01char \xff\n" +
		"/srv/cut\x009:no newline")
	got := parseGrep(stdout)
	want := []SearchHit{
		{Path: "/srv/app/main.go", Line: 12, Text: `listen(":8080")`},
		{Path: "/srv/odd\nname.txt", Line: 3, Text: "a:b:c"},
		{Path: "/srv/bin", Line: 7, Text: "ctlchar �"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("hit %d = %+v; want %+v", i, got[i], want[i])
		}
	}
}

func TestCleanTextCaps(t *testing.T) {
	long := strings.Repeat("é", 1000)
	got := cleanText([]byte(long))
	if !strings.HasSuffix(got, "…") || len([]rune(got)) != 2*snippetContext+201 {
		t.Errorf("len = %d", len([]rune(got)))
	}
}

// blockingRunner answers only when the search is canceled.
type blockingRunner struct{ started chan struct{} }

func (b blockingRunner) RunLine(ctx context.Context, _, _, _ string) (audit.Result, error) {
	close(b.started)
	<-ctx.Done()
	return audit.Result{}, ctx.Err()
}

func TestCancelSearch(t *testing.T) {
	run := blockingRunner{started: make(chan struct{})}
	s := New(nil, run, nil)
	done := make(chan error, 1)
	go func() {
		_, err := s.Search(context.Background(), "srv", SearchQuery{Dir: "/srv", Query: "x", Mode: SearchContent}, "tok")
		done <- err
	}()
	<-run.started
	s.CancelSearch("other") // another token does nothing
	s.CancelSearch("tok")
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("err = %v; want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("search did not stop")
	}
	s.mu.Lock()
	left := len(s.searches)
	s.mu.Unlock()
	if left != 0 {
		t.Errorf("%d searches still tracked", left)
	}
}
