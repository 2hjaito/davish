package logs

import (
	"context"
	"errors"
	"testing"

	"davish/internal/apperr"
	"davish/internal/audit"
)

func TestSearchLine(t *testing.T) {
	const tail = ` | tail -n 500 | cut -b 1-4000`
	tests := []struct {
		name    string
		src     Source
		q       SearchQuery
		viaGrep bool
		want    string
	}{
		{
			"file",
			Source{Kind: KindFile, Target: "/var/log/app.log"},
			SearchQuery{Query: "timeout"},
			false,
			`{ timeout 60 grep -F -i -n -e timeout -- /var/log/app.log; echo "exit:$?" >&2; }` + tail,
		},
		{
			"file, match case, quoted",
			Source{Kind: KindFile, Target: "/srv/my app/out.log"},
			SearchQuery{Query: "it's -v", MatchCase: true},
			false,
			`{ timeout 60 grep -F -n -e 'it'\''s -v' -- '/srv/my app/out.log'; echo "exit:$?" >&2; }` + tail,
		},
		{
			"journal unit",
			Source{Kind: KindJournal, Target: "nginx.service"},
			SearchQuery{Query: "a.b"},
			false,
			`{ timeout 60 journalctl -u nginx.service --no-pager -o short-iso -g '\Qa.b\E' --case-sensitive=false -n 500; echo "exit:$?" >&2; }` + tail,
		},
		{
			"whole journal through grep",
			Source{Kind: KindJournal},
			SearchQuery{Query: "oom", MatchCase: true},
			true,
			`{ timeout 60 journalctl --no-pager -o short-iso; echo "exit:$?" >&2; } | grep -F -e oom` + tail,
		},
		{
			"docker",
			Source{Kind: KindDocker, Target: "web-1"},
			SearchQuery{Query: "ERROR"},
			false,
			`{ timeout 60 docker logs --timestamps web-1 2>&1; echo "exit:$?" >&2; } | grep -F -i -e ERROR` + tail,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := searchLine(tt.src, tt.q, tt.viaGrep)
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
	tests := []struct {
		src Source
		q   string
	}{
		{Source{Kind: KindFile, Target: "relative.log"}, "x"},
		{Source{Kind: KindFile, Target: "/var/log/../x"}, "x"},
		{Source{Kind: KindJournal, Target: "-x.service"}, "x"},
		{Source{Kind: KindDocker, Target: "--privileged"}, "x"},
		{Source{Kind: KindFile, Target: "/var/log/a"}, " "},
		{Source{Kind: KindFile, Target: "/var/log/a"}, "a\nb"},
		{Source{Kind: "pipe", Target: "/x"}, "x"},
	}
	for _, tt := range tests {
		if _, err := searchLine(tt.src, SearchQuery{Query: tt.q}, false); err == nil {
			t.Errorf("searchLine(%+v, %q) accepted", tt.src, tt.q)
		}
	}
}

func TestPCRELiteral(t *testing.T) {
	tests := map[string]string{
		"a.b":      `\Qa.b\E`,
		`x\Ey`:     `\Qx\E\\E\Qy\E`,
		"(?i)boom": `\Q(?i)boom\E`,
	}
	for in, want := range tests {
		if got := pcreLiteral(in); got != want {
			t.Errorf("pcreLiteral(%q) = %q; want %q", in, got, want)
		}
	}
}

func TestAroundLine(t *testing.T) {
	got, err := aroundLine("/var/log/syslog", 101, 50)
	if err != nil {
		t.Fatal(err)
	}
	want := `sed -n -e '101,150p;150q' -- /var/log/syslog | cut -b 1-4000`
	if got != want {
		t.Errorf("got %s", got)
	}
	for _, r := range [][2]int{{0, 10}, {1, 0}, {1, AroundLimit + 1}} {
		if _, err := aroundLine("/var/log/syslog", r[0], r[1]); err == nil {
			t.Errorf("range %v accepted", r)
		}
	}
}

func TestParseSearch(t *testing.T) {
	got := parseSearch(KindFile, []byte("12:first\n13:has: colon\nnot numbered\n"))
	want := []Line{{12, "first"}, {13, "has: colon"}, {0, "not numbered"}}
	if len(got.Lines) != len(want) || got.Truncated {
		t.Fatalf("got %+v", got)
	}
	for i := range want {
		if got.Lines[i] != want[i] {
			t.Errorf("line %d = %+v; want %+v", i, got.Lines[i], want[i])
		}
	}
	j := parseSearch(KindJournal, []byte("-- No entries --\n2026-11-16T10:00:00+0000 host x[1]: boom\xff\n"))
	if len(j.Lines) != 1 || j.Lines[0].Text != "2026-11-16T10:00:00+0000 host x[1]: boom�" {
		t.Errorf("journal = %+v", j.Lines)
	}
}

type fakeRunner struct {
	results []audit.Result
	lines   []string
}

func (f *fakeRunner) RunLine(_ context.Context, _, _, line string) (audit.Result, error) {
	f.lines = append(f.lines, line)
	r := f.results[0]
	f.results = f.results[1:]
	return r, nil
}

func TestSearchFallsBackToGrep(t *testing.T) {
	run := &fakeRunner{results: []audit.Result{
		{Stderr: []byte("Compiled without pattern matching support\nexit:1\n")},
		{Stdout: []byte("2026-11-16 boom\n"), Stderr: []byte("Hint: You are currently not seeing messages from other users and the system.\nexit:0\n")},
	}}
	s := NewSearcher(run)
	got, err := s.Search(context.Background(), "srv", Source{Kind: KindJournal}, SearchQuery{Query: "boom"}, "t1")
	if err != nil {
		t.Fatal(err)
	}
	if len(run.lines) != 2 || len(got.Lines) != 1 || !got.Restricted || got.Command != run.lines[1] {
		t.Errorf("got %+v, ran %q", got, run.lines)
	}
}

func TestSearchFailure(t *testing.T) {
	run := &fakeRunner{results: []audit.Result{
		{Stderr: []byte("grep: /var/log/secure: Permission denied\nexit:2\n")},
	}}
	_, err := NewSearcher(run).Search(context.Background(), "srv", Source{Kind: KindFile, Target: "/var/log/secure"}, SearchQuery{Query: "x"}, "")
	var ae *apperr.Error
	if !errors.As(err, &ae) || ae.Code != apperr.CmdNeedsRoot {
		t.Errorf("err = %v", err)
	}
}
