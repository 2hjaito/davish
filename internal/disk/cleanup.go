package disk

import (
	"bufio"
	"bytes"
	"cmp"
	"context"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"davish/internal/apperr"
	"davish/internal/shellx"
)

// Cleanup is one way to free space, with what it would free.
type Cleanup struct {
	ID    string `json:"id"`
	Bytes int64  `json:"bytes"`
	// Count is how many files it would remove, when that is known.
	Count int `json:"count,omitempty"`
	// Command is exactly what runs; it never holds user input.
	Command string `json:"command"`
	// Root is set when the command needs root: Davish types SudoCommand
	// into a terminal instead of running it.
	Root        bool   `json:"root"`
	SudoCommand string `json:"sudoCommand,omitempty"`
	// Danger is set when it may delete data, not only caches and leftovers.
	Danger bool `json:"danger,omitempty"`
}

// journalKeep is what vacuuming leaves of the journal.
const journalKeep = 100 << 20

// The cleanups, by ID. Each command is fixed text.
var commands = map[string]string{
	"docker-images":     "docker image prune -a -f",
	"docker-containers": "docker container prune -f",
	"docker-cache":      "docker builder prune -a -f",
	"docker-volumes":    "docker volume prune -a -f",
	"docker-logs":       `find /var/lib/docker/containers -name '*-json.log' -size +10M -exec truncate -s 0 {} +`,
	"journal":           "journalctl --vacuum-size=100M",
	"logs":              `find /var/log -xdev -type f \( -name '*.gz' -o -name '*.[0-9]' -o -name '*.old' \) -delete`,
	"apt":               "apt-get clean",
	"dnf":               "dnf clean all",
	"coredumps":         "find /var/lib/systemd/coredump /var/crash -xdev -type f -delete",
	"tmp":               "find /tmp /var/tmp -xdev -type f -atime +7 -delete",
	"snap":              `snap list --all | awk '$6 ~ /disabled/ {print $1, $3}' | while read -r n r; do snap remove "$n" --revision="$r"; done`,
	"home-cache":        `find "$HOME/.cache" -mindepth 1 -delete`,
	"npm":               "npm cache clean --force",
}

// rootCleanups need root unless the user is root.
var rootCleanups = []string{"docker-logs", "journal", "logs", "apt", "dnf", "coredumps", "tmp", "snap"}

// sumFiles prints the total size and count of the files find lists.
const sumFiles = ` -printf '%s\n' 2>/dev/null | awk '{s+=$1;n++} END {print s+0, n+0}'`

// userProbe measures what the login user may clean; it runs as the user
// since its paths are under their home.
const userProbe = `export LC_ALL=C; id -u; echo --docker--; docker system df --format '{{.Type}}\t{{.Reclaimable}}' 2>/dev/null;` +
	` echo --home-cache--; [ -d "$HOME/.cache" ] && du -sk "$HOME/.cache" 2>/dev/null;` +
	` echo --npm--; [ -d "$HOME/.npm/_cacache" ] && du -sk "$HOME/.npm/_cacache" 2>/dev/null;`

// systemProbe measures the rest, as root when sudo needs no password so
// that root-owned folders count; missing tools print nothing.
const systemProbe = `export LC_ALL=C; echo --journal--; journalctl --disk-usage 2>/dev/null;` +
	` echo --logs--; find /var/log -xdev -type f \( -name '*.gz' -o -name '*.[0-9]' -o -name '*.old' \)` + sumFiles + `;` +
	` echo --docker-logs--; find /var/lib/docker/containers -name '*-json.log' -size +10M` + sumFiles + `;` +
	` echo --coredumps--; find /var/lib/systemd/coredump /var/crash -xdev -type f` + sumFiles + `;` +
	` echo --tmp--; find /tmp /var/tmp -xdev -type f -atime +7` + sumFiles + `;` +
	` echo --apt--; [ -d /var/cache/apt/archives ] && du -sk /var/cache/apt/archives 2>/dev/null;` +
	` echo --dnf--; [ -d /var/cache/dnf ] && du -sk /var/cache/dnf 2>/dev/null;` +
	` echo --snap--; command -v snap >/dev/null && snap list --all 2>/dev/null | awk '$6 ~ /disabled/ {print "/var/lib/snapd/snaps/" $1 "_" $3 ".snap"}' | xargs -r du -ck 2>/dev/null | tail -n 1; true`

// probeLine measures every cleanup in one command.
var probeLine = userProbe + ` if sudo -n true 2>/dev/null; then sudo -n sh -c ` + shellx.Quote(systemProbe) +
	`; else sh -c ` + shellx.Quote(systemProbe) + `; fi`

// withSudo prefixes sudo, or runs the whole line under it when it is more
// than one command, so every part of a pipeline or loop gets root.
func withSudo(line string) string {
	if strings.ContainsAny(line, "|;&") {
		return "sudo sh -c " + shellx.Quote(line)
	}
	return "sudo " + line
}

// Cleanups lists what could be cleaned, largest first; those that would
// free nothing are left out.
func (s *Service) Cleanups(ctx context.Context, serverID string) ([]Cleanup, error) {
	res, err := s.run.RunLine(ctx, serverID, app, probeLine)
	if err != nil {
		return nil, err
	}
	return parseCleanups(res.Stdout), nil
}

func parseCleanups(out []byte) []Cleanup {
	sections := map[string][]string{}
	cur := "uid"
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "--") && strings.HasSuffix(line, "--") && len(line) > 4 {
			cur = strings.Trim(line, "-")
			continue
		}
		if line != "" {
			sections[cur] = append(sections[cur], line)
		}
	}
	root := len(sections["uid"]) > 0 && sections["uid"][0] == "0"
	var cs []Cleanup
	add := func(id string, bytes int64, count int) {
		if bytes <= 0 {
			return
		}
		c := Cleanup{
			ID: id, Bytes: bytes, Count: count, Command: commands[id],
			Root: slices.Contains(rootCleanups, id) && !root, Danger: id == "docker-volumes",
		}
		if c.Root {
			c.SudoCommand = withSudo(c.Command)
		}
		cs = append(cs, c)
	}

	// Docker answered, so this user may run it.
	dockerIDs := map[string]string{
		"Images": "docker-images", "Containers": "docker-containers",
		"Build Cache": "docker-cache", "Local Volumes": "docker-volumes",
	}
	for _, l := range sections["docker"] {
		typ, rec, _ := strings.Cut(l, "\t")
		if id, ok := dockerIDs[typ]; ok {
			add(id, parseSize(rec), 0)
		}
	}
	if j := sections["journal"]; len(j) > 0 {
		if m := journalRe.FindStringSubmatch(j[0]); m != nil {
			add("journal", parseSize(m[1])-journalKeep, 0)
		}
	}
	// "bytes count" from sumFiles.
	for _, id := range []string{"logs", "docker-logs", "coredumps", "tmp"} {
		if l := sections[id]; len(l) > 0 {
			f := strings.Fields(l[0])
			if len(f) == 2 {
				n, _ := strconv.ParseInt(f[0], 10, 64)
				c, _ := strconv.Atoi(f[1])
				add(id, n, c)
			}
		}
	}
	// "KB<tab>path" from du -sk, or "KB<tab>total" from du -ck.
	for _, id := range []string{"apt", "dnf", "home-cache", "npm", "snap"} {
		if l := sections[id]; len(l) > 0 {
			kb, _, _ := strings.Cut(l[0], "\t")
			n, _ := strconv.ParseInt(kb, 10, 64)
			// An empty cache still holds a few KB of lock and partial dirs.
			if n > 1024 {
				add(id, n<<10, 0)
			}
		}
	}
	slices.SortStableFunc(cs, func(a, b Cleanup) int { return cmp.Compare(b.Bytes, a.Bytes) })
	return cs
}

var journalRe = regexp.MustCompile(`take up ([\d.]+\s*[KMGTP]?i?B?)`)

var sizeRe = regexp.MustCompile(`^([\d.]+)\s*([kKMGTP]?)(i?)B?`)

// parseSize reads docker's "1.2GB (40%)" or journalctl's "120.0M".
func parseSize(s string) int64 {
	m := sizeRe.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return 0
	}
	v, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0
	}
	// docker uses decimal units; journalctl's single letters are binary.
	base := 1000.0
	if m[3] == "i" || (m[2] != "" && !strings.Contains(s, "B")) {
		base = 1024
	}
	exp := strings.Index("KMGTP", strings.ToUpper(m[2])) + 1
	for range exp {
		v *= base
	}
	return int64(v)
}

// Clean runs a cleanup the user confirmed. command must be the one shown,
// so a confirmation cannot be replayed for another command.
func (s *Service) Clean(ctx context.Context, serverID, id, command string) error {
	want, ok := commands[id]
	if !ok || command != want {
		return apperr.New(apperr.InvalidInput, "cleanup does not match what was confirmed")
	}
	// A prune of years of images can outlast the default deadline.
	ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()
	res, err := s.run.RunLine(ctx, serverID, app, want)
	if err != nil {
		return err
	}
	if res.Exit != 0 {
		msg := strings.TrimSpace(string(res.Stderr))
		if strings.Contains(msg, "ermission denied") {
			return apperr.New(apperr.CmdNeedsRoot, msg)
		}
		return apperr.New(apperr.CmdFailed, msg)
	}
	return nil
}
