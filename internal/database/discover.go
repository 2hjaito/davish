package database

import (
	"context"
	"strconv"
	"strings"
)

// Running is a database server Find saw running.
type Running struct {
	Kind string `json:"kind"`
	// Port is where it listens, from its config; 0 when unknown.
	Port int `json:"port"`
	// Bind is MongoDB's bindIp.
	Bind string `json:"bind,omitempty"`
	// Auth is "on" or "off" for MongoDB's authorization, "" when unknown.
	Auth string `json:"auth,omitempty"`
}

// SQLiteFile is a SQLite database Find came across.
type SQLiteFile struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
	// Root is set when only root can read it, through sudo -n.
	Root bool `json:"root"`
}

// Found is what Find saw on the server.
type Found struct {
	Running []Running    `json:"running"`
	SQLite  []SQLiteFile `json:"sqlite"`
	// SQLite3 is set when the sqlite3 client is installed.
	SQLite3 bool `json:"sqlite3"`
}

// findLine only reads: the running processes, /etc/mongod.conf, and the
// files named like SQLite databases in the usual places, kept when they
// start with SQLite's header. It looks through sudo -n when sudo needs no
// password, so files only root can read show too; it stops after 20s and
// 300 files.
const findLine = `S=; sudo -n true 2>/dev/null && S="sudo -n"
echo '#ps'; ps -eo comm= 2>/dev/null | sort -u
echo '#mongod.conf'; cat /etc/mongod.conf 2>/dev/null
echo '#sqlite3'; command -v sqlite3
echo '#files'
$S timeout 20 find /var/lib /opt /srv /home /root -maxdepth 6 \( -path /var/lib/docker -o -path /var/lib/containerd -o -name node_modules -o -name .git -o -name .cache \) -prune -o -type f \( -name '*.db' -o -name '*.sqlite' -o -name '*.sqlite3' \) -size +0 -print 2>/dev/null | head -n 300 | while IFS= read -r f; do
  if [ -r "$f" ]; then w=ok; elif [ -n "$S" ]; then w=root; else continue; fi
  [ "$($S head -c 15 "$f" 2>/dev/null)" = "SQLite format 3" ] && printf '%s\t%s\t%s\n' "$w" "$($S stat -c %s "$f")" "$f"
done
true`

// Find looks for database servers and SQLite files on the server. It
// changes nothing.
func (s *Service) Find(ctx context.Context, serverID string) (Found, error) {
	res, err := s.run.RunInput(ctx, serverID, app, findLine, nil)
	if err != nil {
		return Found{}, err
	}
	return parseFound(string(res.Stdout)), nil
}

var processKinds = map[string]string{"mongod": Mongo, "mysqld": MySQL, "mariadbd": MySQL, "postgres": Postgres}

var defaultPorts = map[string]int{Mongo: 27017, MySQL: 3306, Postgres: 5432}

func parseFound(out string) Found {
	f := Found{Running: []Running{}, SQLite: []SQLiteFile{}}
	sections := map[string][]string{}
	part := ""
	for l := range strings.Lines(out) {
		l = strings.TrimRight(l, "\r\n")
		if strings.HasPrefix(l, "#") && sections[l] == nil && (l == "#ps" || l == "#mongod.conf" || l == "#sqlite3" || l == "#files") {
			part = l
			sections[part] = []string{}
			continue
		}
		sections[part] = append(sections[part], l)
	}
	seen := map[string]bool{}
	for _, p := range sections["#ps"] {
		kind, ok := processKinds[strings.TrimSpace(p)]
		if !ok || seen[kind] {
			continue
		}
		seen[kind] = true
		r := Running{Kind: kind, Port: defaultPorts[kind]}
		if kind == Mongo {
			r.Port, r.Bind, r.Auth = parseMongodConf(sections["#mongod.conf"])
		}
		f.Running = append(f.Running, r)
	}
	for _, l := range sections["#sqlite3"] {
		f.SQLite3 = f.SQLite3 || strings.HasPrefix(l, "/")
	}
	for _, l := range sections["#files"] {
		p := strings.SplitN(l, "\t", 3)
		if len(p) != 3 || !strings.HasPrefix(p[2], "/") {
			continue
		}
		size, _ := strconv.ParseInt(p[1], 10, 64)
		f.SQLite = append(f.SQLite, SQLiteFile{Path: p[2], Size: size, Root: p[0] == "root"})
	}
	return f
}

// parseMongodConf reads net.port, net.bindIp and security.authorization
// from mongod.conf, which is YAML: enough of it for these three keys.
func parseMongodConf(lines []string) (port int, bind, auth string) {
	port, auth = 27017, "off"
	section := ""
	for _, l := range lines {
		if i := strings.IndexByte(l, '#'); i >= 0 {
			l = l[:i]
		}
		if strings.TrimSpace(l) == "" {
			continue
		}
		key, val, _ := strings.Cut(strings.TrimSpace(l), ":")
		val = strings.Trim(strings.TrimSpace(val), `"'`)
		if l[0] != ' ' && l[0] != '\t' {
			section = key
			continue
		}
		switch section + "." + key {
		case "net.port":
			if n, err := strconv.Atoi(val); err == nil {
				port = n
			}
		case "net.bindIp":
			bind = val
		case "security.authorization":
			if val == "enabled" {
				auth = "on"
			}
		}
	}
	if len(lines) == 0 {
		auth = ""
	}
	return port, bind, auth
}
