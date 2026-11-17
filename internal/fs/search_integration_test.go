//go:build integration

package fs_test

import (
	"context"
	"slices"
	"testing"

	"davish/internal/fs"
	"davish/internal/shellx"
)

func TestSearchNamesAndContent(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	e.run(t, "cd "+shellx.Quote(e.dir)+` && mkdir -p 'my dir/deep' locked .git node_modules &&
		printf 'listen 8080;\nserver_name tiếng.vn;\n' > 'my dir/site.conf' &&
		printf 'nothing here\n' > 'my dir/deep/other.CONF' &&
		printf 'listen 9090;\n' > locked/hidden.conf && chmod 000 locked &&
		printf 'listen 1;\n' > .git/config.conf && printf 'listen 2;\n' > node_modules/x.conf &&
		touch "$(printf 'odd\nname.conf')" && head -c 2048 /dev/urandom > blob.bin`)
	t.Cleanup(func() { e.run(t, shellx.Cmd("chmod", "755", e.path("locked")).String()) })

	got, err := e.svc.Search(ctx, e.id, fs.SearchQuery{Dir: e.dir, Query: "conf", Mode: fs.SearchName}, "")
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, h := range got.Hits {
		paths = append(paths, h.Path)
	}
	for _, want := range []string{e.path("my dir/site.conf"), e.path("my dir/deep/other.CONF"), e.path("odd\nname.conf")} {
		if !slices.Contains(paths, want) {
			t.Errorf("name search missed %q in %q", want, paths)
		}
	}
	if got.Denied == 0 {
		t.Errorf("denied = 0, want the locked folder counted")
	}

	got, err = e.svc.Search(ctx, e.id, fs.SearchQuery{Dir: e.dir, Query: "LISTEN", Mode: fs.SearchContent}, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Hits) != 1 || got.Hits[0].Path != e.path("my dir/site.conf") || got.Hits[0].Line != 1 || got.Hits[0].Text != "listen 8080;" {
		t.Errorf("content hits = %+v (.git, node_modules and binaries must be skipped)", got.Hits)
	}

	got, err = e.svc.Search(ctx, e.id, fs.SearchQuery{Dir: e.dir, Query: "tiếng", Mode: fs.SearchContent, MatchCase: true}, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Hits) != 1 || got.Hits[0].Line != 2 {
		t.Errorf("unicode hits = %+v", got.Hits)
	}

	got, err = e.svc.Search(ctx, e.id, fs.SearchQuery{Dir: e.dir, Query: "no such thing", Mode: fs.SearchContent}, "")
	if err != nil || len(got.Hits) != 0 {
		t.Errorf("no match: %+v, %v", got, err)
	}
}
