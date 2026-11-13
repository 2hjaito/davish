package docker

import (
	"context"
	"errors"
	"path"
	"sort"
	"strings"
	"testing"
)

// fakeFiles is a folder tree: file path → content. Folders are implied.
type fakeFiles struct {
	files map[string]string
	reads []string
}

func (f *fakeFiles) ReadDir(p string) ([]string, error) {
	seen := map[string]bool{}
	for k := range f.files {
		rest, ok := strings.CutPrefix(k, p+"/")
		if !ok {
			continue
		}
		name, _, _ := strings.Cut(rest, "/")
		seen[name] = true
	}
	if len(seen) == 0 {
		return nil, errors.New("no such folder")
	}
	names := make([]string, 0, len(seen))
	for n := range seen {
		names = append(names, n)
	}
	sort.Strings(names)
	return names, nil
}

func (f *fakeFiles) ReadFile(p string, limit int64) ([]byte, error) {
	f.reads = append(f.reads, path.Base(p))
	c, ok := f.files[p]
	if !ok {
		return nil, errors.New("no such file")
	}
	if int64(len(c)) > limit {
		c = c[:limit]
	}
	return []byte(c), nil
}

const png = "\x89PNG\r\n\x1a\nrest"

func TestStackIconFavicon(t *testing.T) {
	fs := &fakeFiles{files: map[string]string{
		"/srv/shop/docker-compose.yml":           "services: {}",
		"/srv/shop/web/package.json":             `{"dependencies":{"next":"15","react":"19"}}`,
		"/srv/shop/web/src/app/favicon.ico":      "\x00\x00\x01\x00ico",
		"/srv/shop/web/public/favicon.png":       png,
		"/srv/shop/web/public/not-an-icon.txt":   "x",
		"/srv/shop/api/package.json":             `{"dependencies":{"express":"4"}}`,
		"/srv/shop/api/public/favicon.svg":       "<svg/>",
		"/srv/shop/node_modules/x/package.json":  "{}",
		"/srv/shop/web/node_modules/a/icon.png":  png,
		"/srv/shop/web/app-unrelated/readme.txt": "",
	}}
	got := findStackIcon(fs, "/srv/shop")
	// web comes before api in appDirs; in web, src/app (Next) before public.
	if !strings.HasPrefix(got.Icon, "data:image/x-icon;base64,") {
		t.Fatalf("icon = %.40q", got.Icon)
	}
	if got.Tech != "next" {
		t.Fatalf("tech = %q, want next (it carries react)", got.Tech)
	}
}

func TestStackIconTechOnly(t *testing.T) {
	cases := map[string]map[string]string{
		"nest":    {"/p/package.json": `{"dependencies":{"@nestjs/core":"10","express":"4"}}`},
		"express": {"/p/server/package.json": `{"dependencies":{"express":"4"}}`},
		"vue":     {"/p/package.json": `{"devDependencies":{"vue":"3"}}`},
		"node":    {"/p/package.json": `{"name":"x"}`},
		"bun":     {"/p/package.json": `{}`, "/p/bun.lockb": "b"},
		"django":  {"/p/requirements.txt": "Django==5.0\npsycopg[binary]\n"},
		"fastapi": {"/p/pyproject.toml": "dependencies = [\"fastapi>=0.110\", \"uvicorn\"]"},
		"python":  {"/p/requirements.txt": "requests\nflask-cors-not-flask\n"},
		"laravel": {"/p/composer.json": `{"require":{"laravel/framework":"^11"}}`},
		"rails":   {"/p/Gemfile": "source 'https://rubygems.org'\ngem \"rails\", \"~> 7\"\n"},
		"go":      {"/p/backend/go.mod": "module x"},
		"rust":    {"/p/Cargo.toml": "[package]"},
		"spring":  {"/p/pom.xml": "<artifactId>spring-boot-starter-web</artifactId>"},
		"":        {"/p/docker-compose.yml": "services: {}"},
	}
	for want, files := range cases {
		got := findStackIcon(&fakeFiles{files: files}, "/p")
		if got.Tech != want || got.Icon != "" {
			t.Errorf("%v: got %+v, want tech %q", files, got, want)
		}
	}
}

func TestStackIconRejects(t *testing.T) {
	big := png + strings.Repeat("x", maxIcon)
	fs := &fakeFiles{files: map[string]string{
		"/p/public/favicon.png": big,                        // too large
		"/p/public/favicon.ico": "<html>not an icon</html>", // wrong bytes
		"/p/static/icon.svg":    "<?xml version=\"1.0\"?>\n<!-- logo -->\n<svg xmlns=\"http://www.w3.org/2000/svg\"/>",
	}}
	got := findStackIcon(fs, "/p")
	if !strings.HasPrefix(got.Icon, "data:image/svg+xml;base64,") {
		t.Fatalf("want the svg after skipping the bad ones, got %.40q", got.Icon)
	}
}

func TestStackIconReadsLittle(t *testing.T) {
	// Only listed files are read: no guessing paths over the network.
	fs := &fakeFiles{files: map[string]string{"/p/compose.yml": "", "/p/README.md": ""}}
	_ = findStackIcon(fs, "/p")
	if len(fs.reads) != 0 {
		t.Fatalf("read %v", fs.reads)
	}
}

func TestDataURLKinds(t *testing.T) {
	for in, want := range map[string]string{
		png:                         "image/png",
		"\xFF\xD8\xFFjpeg":          "image/jpeg",
		"GIF89a..":                  "image/gif",
		"RIFF\x00\x00\x00\x00WEBPx": "image/webp",
		"\xEF\xBB\xBF  <svg viewBox=\"0 0 1 1\">": "image/svg+xml",
		"<!DOCTYPE svg><svg>":                     "image/svg+xml",
		"<html><svg></svg></html>":                "",
		"plain text":                              "",
	} {
		got := dataURL([]byte(in))
		if (want == "") != (got == "") || (want != "" && !strings.HasPrefix(got, "data:"+want+";")) {
			t.Errorf("%q → %.30q, want %s", in, got, want)
		}
	}
}

func TestStackIconValidatesDir(t *testing.T) {
	s := New(nil)
	for _, dir := range []string{"", "relative/dir", "/a/../b", "/a/"} {
		if _, err := s.StackIcon(context.Background(), "srv", dir); err == nil {
			t.Errorf("%q accepted", dir)
		}
	}
	// No SFTP wired: nothing found, no error.
	if got, err := s.StackIcon(context.Background(), "srv", "/srv/app"); err != nil || got != (StackIcon{}) {
		t.Fatalf("got %+v, %v", got, err)
	}
}
