//go:build integration

package logs_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"davish/internal/audit"
	"davish/internal/logs"
	"davish/internal/shellx"
	"davish/internal/testsshd"
)

func searcher(t *testing.T) (*logs.Searcher, *audit.Runner, *audit.Log, string) {
	t.Helper()
	m, id := testsshd.Connect(t)
	log := audit.New(&recorder{lines: map[string][]string{}, ends: map[string]logs.End{}})
	run := audit.NewRunner(m, log)
	return logs.NewSearcher(run), run, log, id
}

func TestSearchBigFileAndPage(t *testing.T) {
	s, run, log, id := searcher(t)
	ctx := context.Background()
	file := "/tmp/davish-search-" + strings.ReplaceAll(t.Name(), "/", "-") + ".log"
	// 300 000 lines, one "needle" every 1000, the last at line 300000.
	gen := `awk 'BEGIN { for (i = 1; i <= 300000; i++) { if (i % 1000 == 0) print "line " i " needle here"; else print "line " i " ordinary" } }' > ` + shellx.Quote(file)
	if _, err := run.RunLine(ctx, id, "test", gen); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = run.RunLine(context.Background(), id, "test", shellx.Cmd("rm", "-f", "--", file).String())
	})
	before := len(log.Entries())

	start := time.Now()
	got, err := s.Search(ctx, id, logs.Source{Kind: logs.KindFile, Target: file}, logs.SearchQuery{Query: "NEEDLE"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Lines) != 300 || got.Truncated {
		t.Fatalf("lines = %d truncated = %v", len(got.Lines), got.Truncated)
	}
	if last := got.Lines[len(got.Lines)-1]; last.Number != 300000 || last.Text != "line 300000 needle here" {
		t.Errorf("last = %+v", last)
	}
	t.Logf("searched 300k lines in %v", time.Since(start))

	page, err := s.Around(ctx, id, file, 299951, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Lines) != 50 || !page.EOF || page.Lines[0].Number != 299951 || page.Lines[0].Text != "line 299951 ordinary" {
		t.Errorf("page = %d lines, eof %v, first %+v", len(page.Lines), page.EOF, page.Lines[0])
	}

	// Only the two reads were recorded.
	if n := len(log.Entries()) - before; n != 2 {
		t.Errorf("audit grew by %d, want 2", n)
	}
}

func TestSearchJournal(t *testing.T) {
	s, _, _, id := searcher(t)
	// The test user cannot read the system journal; the search still runs
	// and says the journal is restricted, or finds nothing.
	got, err := s.Search(context.Background(), id, logs.Source{Kind: logs.KindJournal}, logs.SearchQuery{Query: "zz-never-zz"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Lines) != 0 {
		t.Errorf("lines = %+v", got.Lines)
	}
}

func TestSearchUnreadableFile(t *testing.T) {
	s, _, _, id := searcher(t)
	_, err := s.Search(context.Background(), id, logs.Source{Kind: logs.KindFile, Target: "/etc/shadow"}, logs.SearchQuery{Query: "root"}, "")
	if err == nil {
		t.Fatal("reading /etc/shadow succeeded")
	}
}
