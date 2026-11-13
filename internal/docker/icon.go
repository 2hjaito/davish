package docker

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/pkg/sftp"

	"davish/internal/apperr"
)

// Conns opens a server's SFTP session.
type Conns interface {
	SFTP(serverID string) (*sftp.Client, error)
}

// UseConns lets StackIcon read compose project folders over SFTP.
func (s *Service) UseConns(c Conns) { s.conns = c }

// StackIcon is what a compose project's card shows in place of a letter
// badge: the project's own icon, else the framework its code uses. Both
// empty when neither is found.
type StackIcon struct {
	// Icon is a data: URL of the project's favicon or app icon.
	Icon string `json:"icon,omitempty"`
	// Tech names a logo the frontend knows (lib/ui/techIcons.ts).
	Tech string `json:"tech,omitempty"`
}

// maxIcon caps an icon file; a favicon is a few KB.
const maxIcon = 64 << 10

// maxManifest caps package.json and the like.
const maxManifest = 256 << 10

// Folders searched below the project folder, as code often sits in one.
var appDirs = []string{"", "app", "web", "frontend", "client", "api", "server", "backend"}

// Where frameworks keep the site icon, relative to an app folder, best first.
var iconDirs = []string{"app", "src/app", "public", "static", ""}

var iconNames = []string{
	"icon.svg", "icon.png", "favicon.svg", "favicon.png", "favicon.ico", "apple-touch-icon.png",
}

// files is the part of an SFTP session StackIcon needs, so tests can fake it.
type files interface {
	// ReadDir lists a folder's entry names; an error reads as empty.
	ReadDir(p string) ([]string, error)
	ReadFile(p string, limit int64) ([]byte, error)
}

// StackIcon looks in dir, a compose project's folder from its labels, for
// an icon and for the manifests that name its framework. It only reads,
// a handful of folders and small files; nothing runs on the server.
func (s *Service) StackIcon(_ context.Context, serverID, dir string) (StackIcon, error) {
	if !path.IsAbs(dir) || path.Clean(dir) != dir || strings.ContainsRune(dir, 0) {
		return StackIcon{}, apperr.New(apperr.InvalidInput, "not an absolute folder")
	}
	if s.conns == nil {
		return StackIcon{}, nil
	}
	sc, err := s.conns.SFTP(serverID)
	if err != nil {
		return StackIcon{}, err
	}
	return findStackIcon(sftpFiles{sc}, dir), nil
}

func findStackIcon(fs files, dir string) StackIcon {
	var out StackIcon
	listed := map[string]map[string]bool{}
	list := func(p string) map[string]bool {
		if m, ok := listed[p]; ok {
			return m
		}
		m := map[string]bool{}
		names, _ := fs.ReadDir(p)
		for _, n := range names {
			m[n] = true
		}
		listed[p] = m
		return m
	}
	// Only the app folders that exist, the project folder first.
	top := list(dir)
	var apps []string
	for _, a := range appDirs {
		if a == "" || top[a] {
			apps = append(apps, path.Join(dir, a))
		}
	}
	for _, a := range apps {
		if out.Icon == "" {
			out.Icon = findIcon(fs, a, list)
		}
		if out.Tech == "" {
			out.Tech = findTech(fs, a, list(a))
		}
		if out.Icon != "" && out.Tech != "" {
			break
		}
	}
	return out
}

func findIcon(fs files, app string, list func(string) map[string]bool) string {
	here := list(app)
	for _, d := range iconDirs {
		dir := app
		if d != "" {
			first, _, _ := strings.Cut(d, "/")
			if !here[first] {
				continue
			}
			dir = path.Join(app, d)
		}
		names := list(dir)
		for _, n := range iconNames {
			if !names[n] {
				continue
			}
			data, err := fs.ReadFile(path.Join(dir, n), maxIcon+1)
			if err != nil || len(data) > maxIcon {
				continue
			}
			if url := dataURL(data); url != "" {
				return url
			}
		}
	}
	return ""
}

// dataURL embeds an image whose bytes say what it is; anything else is "".
// An SVG shown through <img> cannot run scripts or load anything.
func dataURL(b []byte) string {
	var mime string
	switch {
	case bytes.HasPrefix(b, []byte("\x89PNG\r\n\x1a\n")):
		mime = "image/png"
	case bytes.HasPrefix(b, []byte{0, 0, 1, 0}):
		mime = "image/x-icon"
	case bytes.HasPrefix(b, []byte{0xFF, 0xD8, 0xFF}):
		mime = "image/jpeg"
	case bytes.HasPrefix(b, []byte("GIF8")):
		mime = "image/gif"
	case len(b) > 12 && bytes.HasPrefix(b, []byte("RIFF")) && string(b[8:12]) == "WEBP":
		mime = "image/webp"
	case isSVG(b):
		mime = "image/svg+xml"
	default:
		return ""
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(b)
}

var svgStart = regexp.MustCompile(`^(?s)\s*(<\?xml[^>]*>\s*)?(<!--.*?-->\s*)*(<!DOCTYPE[^>]*>\s*)?<svg[\s>]`)

func isSVG(b []byte) bool {
	b = bytes.TrimPrefix(b, []byte("\xEF\xBB\xBF"))
	if len(b) > 1024 {
		b = b[:1024]
	}
	return svgStart.Match(b)
}

// Frameworks by the package that marks them, most telling first: Next
// carries React, Nest carries Express.
var nodeTechs = []struct{ pkg, tech string }{
	{"next", "next"},
	{"nuxt", "nuxt"},
	{"@remix-run/node", "remix"},
	{"@nestjs/core", "nest"},
	{"astro", "astro"},
	{"@sveltejs/kit", "svelte"},
	{"@angular/core", "angular"},
	{"hono", "hono"},
	{"fastify", "fastify"},
	{"koa", "koa"},
	{"express", "express"},
	{"solid-js", "solid"},
	{"svelte", "svelte"},
	{"vue", "vue"},
	{"react", "react"},
}

var pyTechs = []struct{ pkg, tech string }{
	{"django", "django"},
	{"fastapi", "fastapi"},
	{"flask", "flask"},
}

// findTech reads the manifests present in app, listed in names.
func findTech(fs files, app string, names map[string]bool) string {
	read := func(n string) []byte {
		if !names[n] {
			return nil
		}
		b, err := fs.ReadFile(path.Join(app, n), maxManifest)
		if err != nil {
			return nil
		}
		return b
	}
	if b := read("package.json"); b != nil {
		var pkg struct {
			Deps    map[string]any `json:"dependencies"`
			DevDeps map[string]any `json:"devDependencies"`
		}
		if json.Unmarshal(b, &pkg) == nil {
			for _, t := range nodeTechs {
				if _, ok := pkg.Deps[t.pkg]; ok {
					return t.tech
				}
				if _, ok := pkg.DevDeps[t.pkg]; ok {
					return t.tech
				}
			}
		}
		if names["bun.lockb"] || names["bun.lock"] {
			return "bun"
		}
		return "node"
	}
	for _, n := range []string{"requirements.txt", "pyproject.toml", "Pipfile"} {
		if b := read(n); b != nil {
			for _, t := range pyTechs {
				if pyDeclares(b, t.pkg) {
					return t.tech
				}
			}
			return "python"
		}
	}
	if b := read("composer.json"); b != nil {
		if bytes.Contains(b, []byte(`"laravel/framework"`)) {
			return "laravel"
		}
		return "php"
	}
	if b := read("Gemfile"); b != nil {
		if railsGem.Match(b) {
			return "rails"
		}
		return "ruby"
	}
	switch {
	case names["go.mod"]:
		return "go"
	case names["Cargo.toml"]:
		return "rust"
	case names["deno.json"] || names["deno.jsonc"]:
		return "deno"
	case names["pom.xml"] || names["build.gradle"] || names["build.gradle.kts"]:
		if b := read("pom.xml"); bytes.Contains(b, []byte("spring-boot")) {
			return "spring"
		}
		if b := read("build.gradle"); bytes.Contains(b, []byte("springframework.boot")) {
			return "spring"
		}
		if b := read("build.gradle.kts"); bytes.Contains(b, []byte("springframework.boot")) {
			return "spring"
		}
	}
	return ""
}

var railsGem = regexp.MustCompile(`(?m)^\s*gem\s+['"]rails['"]`)

// pyDeclares finds pkg as a requirement name: at a line start in
// requirements.txt or Pipfile, or quoted in pyproject.toml.
func pyDeclares(b []byte, pkg string) bool {
	q := regexp.QuoteMeta(pkg)
	// The name ends where a version, extra or marker starts: "flask-cors" is not flask.
	end := `(\s|[=<>~!\[;,"']|$)`
	re := regexp.MustCompile(`(?mi)^\s*` + q + end + `|["']` + q + end)
	return re.Match(b)
}

type sftpFiles struct{ sc *sftp.Client }

func (f sftpFiles) ReadDir(p string) ([]string, error) {
	infos, err := f.sc.ReadDir(p)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(infos))
	for _, fi := range infos {
		names = append(names, fi.Name())
	}
	return names, nil
}

func (f sftpFiles) ReadFile(p string, limit int64) ([]byte, error) {
	fi, err := f.sc.Stat(p)
	if err != nil {
		return nil, err
	}
	if !fi.Mode().IsRegular() {
		return nil, os.ErrInvalid
	}
	h, err := f.sc.Open(p)
	if err != nil {
		return nil, err
	}
	defer func() { _ = h.Close() }()
	return io.ReadAll(io.LimitReader(h, limit))
}
