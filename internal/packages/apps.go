package packages

import (
	"context"
	"encoding/json"
	"regexp"
	"slices"
	"strings"

	"davish/internal/apperr"
)

// Kinds of programs, to group them in the UI.
const (
	KindRuntime = "runtime" // languages and their package tools
	KindServer  = "server"  // daemons: web, database, cache, containers
	KindTool    = "tool"
)

// known is one program Davish looks for.
type known struct {
	id, label, kind string
	// bin is the command, versionArg what prints its version.
	bin, versionArg string
	configs         []string
	// units are the systemd units it usually runs as, most likely first.
	units []string
}

// programs are looked for in this order, which is also the display order.
var programs = []known{
	{"nginx", "nginx", KindServer, "nginx", "-v", []string{"/etc/nginx/nginx.conf"}, []string{"nginx.service"}},
	{"apache", "Apache", KindServer, "apache2", "-v", []string{"/etc/apache2/apache2.conf"}, []string{"apache2.service"}},
	{"httpd", "Apache", KindServer, "httpd", "-v", []string{"/etc/httpd/conf/httpd.conf"}, []string{"httpd.service"}},
	{"mysql", "MySQL / MariaDB", KindServer, "mysqld", "--version", []string{"/etc/mysql/my.cnf", "/etc/my.cnf"}, []string{"mysql.service", "mariadb.service", "mysqld.service"}},
	{"mariadb", "MariaDB", KindServer, "mariadbd", "--version", []string{"/etc/mysql/my.cnf", "/etc/my.cnf"}, []string{"mariadb.service"}},
	{"postgres", "PostgreSQL", KindServer, "psql", "--version", nil, []string{"postgresql.service"}},
	{"redis", "Redis", KindServer, "redis-server", "--version", []string{"/etc/redis/redis.conf", "/etc/redis.conf"}, []string{"redis-server.service", "redis.service"}},
	{"mongodb", "MongoDB", KindServer, "mongod", "--version", []string{"/etc/mongod.conf"}, []string{"mongod.service"}},
	{"docker", "Docker", KindServer, "docker", "--version", []string{"/etc/docker/daemon.json"}, []string{"docker.service"}},
	{"node", "Node.js", KindRuntime, "node", "-v", nil, nil},
	{"npm", "npm", KindRuntime, "npm", "-v", nil, nil},
	{"yarn", "Yarn", KindRuntime, "yarn", "-v", nil, nil},
	{"pnpm", "pnpm", KindRuntime, "pnpm", "-v", nil, nil},
	{"php", "PHP", KindRuntime, "php", "-v", nil, nil},
	{"composer", "Composer", KindRuntime, "composer", "--version", nil, nil},
	{"python", "Python", KindRuntime, "python3", "--version", nil, nil},
	{"pip", "pip", KindRuntime, "pip3", "--version", nil, nil},
	{"go", "Go", KindRuntime, "go", "version", nil, nil},
	{"java", "Java", KindRuntime, "java", "-version", nil, nil},
	{"ruby", "Ruby", KindRuntime, "ruby", "-v", nil, nil},
	{"git", "Git", KindTool, "git", "--version", nil, nil},
	{"certbot", "Certbot", KindTool, "certbot", "--version", []string{"/etc/letsencrypt/cli.ini"}, []string{"certbot.timer"}},
}

// App is a program found on the server.
type App struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Kind    string `json:"kind"`
	Version string `json:"version"`
	Path    string `json:"path"`
	// Configs are its config files that exist.
	Configs []string `json:"configs"`
	// Units are the systemd units it may run as; the UI opens the first
	// one Services lists.
	Units []string `json:"units"`
	// Versions lists the versions a manager installed: nvm for Node.js,
	// pyenv for Python; php-fpm pools for PHP.
	Versions []string `json:"versions"`
}

// probeLine looks for every known program in one command: path and first
// line of its version, then config files that exist, nvm and pyenv
// versions and php-fpm binaries. It is a constant built from programs.
var probeLine = buildProbe()

func buildProbe() string {
	var pairs, configs []string
	for _, p := range programs {
		pairs = append(pairs, p.bin+":"+p.versionArg)
		configs = append(configs, p.configs...)
	}
	slices.Sort(configs)
	configs = slices.Compact(configs)
	return `PATH="$PATH:/usr/local/sbin:/usr/sbin:/sbin"; ` +
		`for x in ` + strings.Join(pairs, " ") + `; do b="${x%%:*}"; a="${x#*:}"; ` +
		`p="$(command -v "$b" 2>/dev/null)" || continue; ` +
		`v="$(timeout 5 "$b" "$a" </dev/null 2>&1 | head -n 1)"; printf 'bin\t%s\t%s\t%s\n' "$b" "$p" "$v"; done; ` +
		`for f in ` + strings.Join(configs, " ") + `; do [ -e "$f" ] && printf 'cfg\t%s\n' "$f"; done; ` +
		`for d in "$HOME"/.nvm/versions/node/*/; do [ -d "$d" ] && printf 'nvm\t%s\n' "$(basename "$d")"; done; ` +
		`for d in "$HOME"/.pyenv/versions/*/; do [ -d "$d" ] && printf 'pyenv\t%s\n' "$(basename "$d")"; done; ` +
		`for b in /usr/sbin/php-fpm*; do [ -x "$b" ] && printf 'fpm\t%s\n' "$(basename "$b")"; done; true`
}

// Apps finds the known programs installed on the server.
func (s *Service) Apps(ctx context.Context, serverID string) ([]App, error) {
	out, err := s.read(ctx, serverID, probeLine)
	if err != nil {
		return nil, err
	}
	return parseProbe(out), nil
}

var versionNumber = regexp.MustCompile(`\d+(?:\.\d+)+(?:[-+~][0-9A-Za-z.+~-]*)?|\d+`)

func parseProbe(out []byte) []App {
	found := map[string][3]string{} // bin: path, raw version line
	exists := map[string]bool{}
	var nvm, pyenv, fpm []string
	for _, l := range lines(out) {
		f := strings.Split(l, "\t")
		switch {
		case f[0] == "bin" && len(f) >= 4:
			found[f[1]] = [3]string{f[2], f[3]}
		case f[0] == "cfg" && len(f) == 2:
			exists[f[1]] = true
		case f[0] == "nvm" && len(f) == 2:
			nvm = append(nvm, strings.TrimPrefix(f[1], "v"))
		case f[0] == "pyenv" && len(f) == 2:
			pyenv = append(pyenv, f[1])
		case f[0] == "fpm" && len(f) == 2:
			fpm = append(fpm, f[1])
		}
	}
	apps := []App{}
	for _, p := range programs {
		hit, ok := found[p.bin]
		if !ok {
			continue
		}
		a := App{ID: p.id, Label: p.label, Kind: p.kind, Path: hit[0], Version: versionOf(p.id, hit[1]),
			Configs: []string{}, Units: append([]string{}, p.units...), Versions: []string{}}
		for _, c := range p.configs {
			if exists[c] {
				a.Configs = append(a.Configs, c)
			}
		}
		switch p.id {
		case "node":
			a.Versions = append(a.Versions, nvm...)
		case "python":
			a.Versions = append(a.Versions, pyenv...)
		case "php":
			// php-fpm8.3 runs as php8.3-fpm on Debian, php-fpm elsewhere.
			for _, b := range fpm {
				a.Versions = append(a.Versions, b)
				if v, ok := strings.CutPrefix(b, "php-fpm"); ok && v != "" {
					a.Units = append(a.Units, "php"+v+"-fpm.service")
				} else {
					a.Units = append(a.Units, "php-fpm.service")
				}
			}
		}
		apps = append(apps, a)
	}
	// MariaDB answers to mysqld too; keep one entry.
	if slices.ContainsFunc(apps, func(a App) bool { return a.ID == "mariadb" }) {
		apps = slices.DeleteFunc(apps, func(a App) bool { return a.ID == "mysql" && strings.Contains(found["mysqld"][1], "MariaDB") })
	}
	return apps
}

// versionOf takes the version number out of a program's version line. Go
// prints "go version go1.22.2 linux/amd64", so its "go" prefix goes first.
func versionOf(id, line string) string {
	if id == "go" {
		line = strings.ReplaceAll(line, "go1", "1")
	}
	if id == "pip" {
		line, _, _ = strings.Cut(line, " from ")
	}
	return versionNumber.FindString(line)
}

// NpmGlobal is a package installed with npm -g.
type NpmGlobal struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// npmGlobalsLine lists the global npm packages as JSON.
const npmGlobalsLine = `npm ls -g --depth=0 --json 2>/dev/null`

// NpmGlobals lists the packages installed globally with npm.
func (s *Service) NpmGlobals(ctx context.Context, serverID string) ([]NpmGlobal, error) {
	out, err := s.read(ctx, serverID, npmGlobalsLine)
	if err != nil {
		return nil, err
	}
	return parseNpmGlobals(out)
}

func parseNpmGlobals(out []byte) ([]NpmGlobal, error) {
	var tree struct {
		Dependencies map[string]struct {
			Version string `json:"version"`
		} `json:"dependencies"`
	}
	if len(strings.TrimSpace(string(out))) == 0 {
		return []NpmGlobal{}, nil
	}
	if err := json.Unmarshal(out, &tree); err != nil {
		return nil, apperr.Wrap(apperr.CmdFailed, "npm printed something unexpected", err)
	}
	pkgs := []NpmGlobal{}
	for name, d := range tree.Dependencies {
		pkgs = append(pkgs, NpmGlobal{Name: name, Version: d.Version})
	}
	slices.SortFunc(pkgs, func(a, b NpmGlobal) int { return strings.Compare(a.Name, b.Name) })
	return pkgs, nil
}
