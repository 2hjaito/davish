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
)

// Cleanup is one way to free space, with what it would free.
type Cleanup struct {
	ID    string `json:"id"`
	Bytes int64  `json:"bytes"`
	// Count is how many files it would remove, when that is known.
	Count int `json:"count,omitempty"`
	// Command is exactly what runs; it never holds user input.
	Command string `json:"command"`
	// Root is set when the command needs root: Davish types it with sudo
	// into a terminal instead of running it.
	Root bool `json:"root"`
}

// journalKeep is what vacuuming leaves of the journal.
const journalKeep = 100 << 20

// The cleanups, by ID. Each command is fixed text.
var commands = map[string]string{
	"docker":  "docker system prune -f",
	"journal": "journalctl --vacuum-size=100M",
	"logs":    `find /var/log -xdev -type f \( -name '*.gz' -o -name '*.[0-9]' -o -name '*.old' \) -delete`,
	"apt":     "apt-get clean",
	"dnf":     "dnf clean all",
}

// probeLine measures every cleanup in one command; missing tools print
// nothing in their section.
const probeLine = `id -u; echo --docker--; docker system df --format '{{.Type}}\t{{.Reclaimable}}' 2>/dev/null;` +
	` echo --journal--; journalctl --disk-usage 2>/dev/null;` +
	` echo --logs--; find /var/log -xdev -type f \( -name '*.gz' -o -name '*.[0-9]' -o -name '*.old' \) -printf '%s\n' 2>/dev/null | awk '{s+=$1;n++} END {print s+0, n+0}';` +
	` echo --apt--; [ -d /var/cache/apt/archives ] && du -sk /var/cache/apt/archives 2>/dev/null;` +
	` echo --dnf--; [ -d /var/cache/dnf ] && du -sk /var/cache/dnf 2>/dev/null; true`

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
	add := func(id string, bytes int64, count int, needsRoot bool) {
		if bytes <= 0 {
			return
		}
		cs = append(cs, Cleanup{ID: id, Bytes: bytes, Count: count, Command: commands[id], Root: needsRoot && !root})
	}

	// Docker answered, so this user may run it.
	var docker int64
	for _, l := range sections["docker"] {
		_, rec, _ := strings.Cut(l, "\t")
		docker += parseSize(rec)
	}
	add("docker", docker, 0, false)

	if j := sections["journal"]; len(j) > 0 {
		if m := journalRe.FindStringSubmatch(j[0]); m != nil {
			add("journal", parseSize(m[1])-journalKeep, 0, true)
		}
	}
	if l := sections["logs"]; len(l) > 0 {
		f := strings.Fields(l[0])
		if len(f) == 2 {
			n, _ := strconv.ParseInt(f[0], 10, 64)
			c, _ := strconv.Atoi(f[1])
			add("logs", n, c, true)
		}
	}
	for _, id := range []string{"apt", "dnf"} {
		if l := sections[id]; len(l) > 0 {
			kb, _, _ := strings.Cut(l[0], "\t")
			n, _ := strconv.ParseInt(kb, 10, 64)
			// An empty cache still holds a few KB of lock and partial dirs.
			if n > 1024 {
				add(id, n<<10, 0, true)
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
