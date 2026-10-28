// Package sshconfig reads the user's ~/.ssh/config (never writes it) and
// turns its concrete hosts into servers to import.
package sshconfig

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
)

// Host is one concrete Host alias with the settings ssh would use for it.
type Host struct {
	Alias        string `json:"alias"`
	HostName     string `json:"hostName"`
	User         string `json:"user"`
	Port         int    `json:"port"`
	IdentityFile string `json:"identityFile,omitempty"`
	// ProxyJump is kept as written: "a,b" or "user@host:port", "" for none.
	ProxyJump string `json:"proxyJump,omitempty"`
}

// block is one Host section: its patterns and settings in file order.
type block struct {
	patterns []string
	settings [][2]string // lowercase key, value
}

// maxIncludeDepth matches ssh's own limit on nested Include.
const maxIncludeDepth = 16

// Load parses ~/.ssh/config under home. A missing file is an empty list.
// user is the local login, ssh's default for hosts without User.
func Load(home, user string) ([]Host, error) {
	var blocks []block
	err := parseFile(filepath.Join(home, ".ssh", "config"), home, 0, &blocks)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return resolve(blocks, home, user), nil
}

func parseFile(path, home string, depth int, blocks *[]block) error {
	if depth > maxIncludeDepth {
		return fmt.Errorf("ssh config: Include nested too deeply at %s", path)
	}
	f, err := os.Open(path) //nolint:gosec // the user's own ssh config
	if err != nil {
		return err
	}
	defer f.Close() //nolint:errcheck // read-only

	// Settings before the first Host apply to every host, like "Host *".
	if len(*blocks) == 0 {
		*blocks = append(*blocks, block{patterns: []string{"*"}})
	}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		key, val := splitLine(sc.Text())
		switch key {
		case "":
			continue
		case "host":
			*blocks = append(*blocks, block{patterns: strings.Fields(val)})
		case "match":
			// Match conditions need runtime state; treat the section as never matching.
			*blocks = append(*blocks, block{})
		case "include":
			for _, pat := range strings.Fields(val) {
				pat = expandHome(pat, home)
				if !filepath.IsAbs(pat) {
					pat = filepath.Join(home, ".ssh", pat)
				}
				matches, _ := filepath.Glob(pat) // bad pattern: nothing included, like ssh
				for _, m := range matches {
					if err := parseFile(m, home, depth+1, blocks); err != nil && !errors.Is(err, fs.ErrNotExist) {
						return err
					}
				}
			}
		default:
			last := &(*blocks)[len(*blocks)-1]
			last.settings = append(last.settings, [2]string{key, val})
		}
	}
	if err := sc.Err(); err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	return nil
}

// splitLine returns the lowercase keyword and its value, "" for a blank
// line or comment. "Key value" and "Key=value" are both accepted.
func splitLine(line string) (string, string) {
	line = strings.TrimSpace(line)
	if line == "" || line[0] == '#' {
		return "", ""
	}
	i := strings.IndexAny(line, " \t=")
	if i < 0 {
		return strings.ToLower(line), ""
	}
	key := strings.ToLower(line[:i])
	val := strings.TrimLeft(line[i:], " \t")
	val = strings.TrimPrefix(val, "=")
	val = strings.Trim(strings.TrimSpace(val), `"`)
	return key, val
}

// resolve applies ssh's first-value-wins rule to every concrete alias.
func resolve(blocks []block, home, user string) []Host {
	var hosts []Host
	seen := map[string]bool{}
	for _, b := range blocks {
		for _, alias := range b.patterns {
			if seen[alias] || strings.ContainsAny(alias, "*?!") {
				continue
			}
			seen[alias] = true
			hosts = append(hosts, hostFor(alias, blocks, home, user))
		}
	}
	return hosts
}

func hostFor(alias string, blocks []block, home, user string) Host {
	h := Host{Alias: alias}
	set := map[string]string{}
	for _, b := range blocks {
		if !matches(b.patterns, alias) {
			continue
		}
		for _, kv := range b.settings {
			if _, ok := set[kv[0]]; !ok {
				set[kv[0]] = kv[1]
			}
		}
	}
	h.HostName = strings.ReplaceAll(set["hostname"], "%h", alias)
	if h.HostName == "" {
		h.HostName = alias
	}
	h.User = set["user"]
	if h.User == "" {
		h.User = user
	}
	h.Port, _ = strconv.Atoi(set["port"])
	if h.Port == 0 {
		h.Port = 22
	}
	if id := set["identityfile"]; id != "" && !strings.EqualFold(id, "none") {
		id = strings.ReplaceAll(id, "%d", home) // ~ is kept; conn expands it
		if !strings.Contains(id, "%") {         // other tokens need ssh's runtime state
			h.IdentityFile = id
		}
	}
	if pj := set["proxyjump"]; !strings.EqualFold(pj, "none") {
		h.ProxyJump = pj
	}
	return h
}

// matches reports whether alias matches a Host line: any positive pattern
// matches and no negated one does.
func matches(patterns []string, alias string) bool {
	ok := false
	for _, p := range patterns {
		neg := strings.HasPrefix(p, "!")
		if m, _ := filepath.Match(strings.TrimPrefix(p, "!"), alias); m {
			if neg {
				return false
			}
			ok = true
		}
	}
	return ok
}

func expandHome(p, home string) string {
	if p == "~" {
		return home
	}
	if strings.HasPrefix(p, "~/") {
		return filepath.Join(home, p[2:])
	}
	return p
}

// Hop is one step of a ProxyJump chain: a config alias, or an explicit
// [user@]host[:port] that is not one.
type Hop struct {
	Alias string
	User  string
	Host  string
	Port  int
}

// ParseJump splits a ProxyJump value into hops, first hop first.
func ParseJump(v string) ([]Hop, error) {
	var hops []Hop
	for _, part := range strings.Split(v, ",") {
		part = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(part), "ssh://"))
		if part == "" {
			return nil, fmt.Errorf("empty ProxyJump hop in %q", v)
		}
		h := Hop{Alias: part}
		if at := strings.LastIndex(part, "@"); at >= 0 {
			h.User, part = part[:at], part[at+1:]
		}
		h.Host = part
		if i := strings.LastIndex(part, ":"); i >= 0 && !strings.Contains(part[:i], ":") {
			p, err := strconv.Atoi(part[i+1:])
			if err != nil {
				return nil, fmt.Errorf("bad port in ProxyJump hop %q", h.Alias)
			}
			h.Host, h.Port = part[:i], p
		}
		hops = append(hops, h)
	}
	return hops, nil
}

// LoadDefault is Load for the current OS user.
func LoadDefault() ([]Host, string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, "", fmt.Errorf("locate home dir: %w", err)
	}
	name := ""
	if u, err := user.Current(); err == nil {
		name = u.Username
	}
	hosts, err := Load(home, name)
	return hosts, name, err
}
