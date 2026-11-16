package fs

import (
	"bytes"
	"context"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"davish/internal/apperr"
	"davish/internal/shellx"
)

// Search modes.
const (
	SearchName    = "name"
	SearchContent = "content"
)

// SearchLimit is the most hits a search returns; past it the result says
// it was cut.
const SearchLimit = 500

// searchTimeout bounds find and grep on the server, so a search of a huge
// tree ends even when nobody presses Stop.
const searchTimeout = 30 * time.Second

// searchBytes caps what the server sends back. find and grep stop on
// SIGPIPE once head has read this much, and the runner never holds more.
const searchBytes = 512 << 10

// snippetContext is how many characters of a matching line are kept on
// each side of the match, so a minified file does not fill the result.
const snippetContext = 80

// Folders a search never enters. Name search prunes them only from the
// top of the tree; content search refuses to start inside them, because
// grep reading /proc or /dev can block or never end.
var virtualDirs = []string{"/proc", "/sys", "/dev", "/run"}

// Folders content search skips by name anywhere in the tree.
var skippedDirs = []string{".git", "node_modules"}

// SearchQuery is what to look for and where.
type SearchQuery struct {
	Dir   string `json:"dir"`
	Query string `json:"query"`
	Mode  string `json:"mode"`
	// MatchCase turns off the default case-insensitive match.
	MatchCase bool `json:"matchCase"`
}

// SearchHit is one result: a path, and for content search the line.
type SearchHit struct {
	Path string `json:"path"`
	Line int    `json:"line,omitempty"`
	Text string `json:"text,omitempty"`
}

// SearchResult is what a search found.
type SearchResult struct {
	Hits []SearchHit `json:"hits"`
	// Truncated is set when there were more than SearchLimit hits.
	Truncated bool `json:"truncated"`
	// Denied counts the places the account could not read.
	Denied int `json:"denied"`
	// TimedOut is set when the server-side timeout ended the search.
	TimedOut bool `json:"timedOut"`
	// Command is the command line that ran, as the audit log shows it.
	Command string `json:"command"`
}

// Search looks for files under q.Dir by name, or for lines in them by
// content. token names the search for CancelSearch. It only reads.
func (s *Service) Search(ctx context.Context, serverID string, q SearchQuery, token string) (SearchResult, error) {
	line, err := searchLine(q)
	if err != nil {
		return SearchResult{}, err
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer s.trackSearch(token, cancel)()
	res, err := s.run.RunLine(ctx, serverID, "files", line)
	if err != nil {
		return SearchResult{}, err
	}
	out := parseSearch(q.Mode, res.Stdout, res.Stderr)
	out.Command = line
	if out.exit != 0 && out.exit != 1 && out.exit != 124 && len(out.Hits) == 0 && out.Denied == 0 {
		// find or grep refused to run (bad pattern, missing option).
		return SearchResult{}, apperr.New(apperr.CmdFailed, firstLine(res.Stderr))
	}
	return out.SearchResult, nil
}

// CancelSearch stops a running Search; find or grep ends with its session
// or, at the latest, at its own timeout.
func (s *Service) CancelSearch(token string) {
	s.mu.Lock()
	cancel := s.searches[token]
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (s *Service) trackSearch(token string, cancel context.CancelFunc) func() {
	if token == "" {
		return func() {}
	}
	s.mu.Lock()
	if s.searches == nil {
		s.searches = map[string]context.CancelFunc{}
	}
	s.searches[token] = cancel
	s.mu.Unlock()
	return func() {
		s.mu.Lock()
		delete(s.searches, token)
		s.mu.Unlock()
	}
}

// searchLine builds the command for q. The tool's exit status goes to
// stderr as "exit:N", since sh has no pipefail to report it through head.
func searchLine(q SearchQuery) (string, error) {
	if err := CheckPath(q.Dir); err != nil {
		return "", err
	}
	query := strings.TrimSpace(q.Query)
	if query == "" || strings.ContainsAny(query, "\x00\n\r") {
		return "", apperr.New(apperr.InvalidInput, "search text must be one non-empty line")
	}
	secs := strconv.Itoa(int(searchTimeout / time.Second))
	var tool shellx.Command
	switch q.Mode {
	case SearchName:
		tool = findCommand(secs, q.Dir, query, q.MatchCase)
	case SearchContent:
		if underVirtual(q.Dir) {
			return "", apperr.New(apperr.InvalidInput, "content search cannot run in this folder")
		}
		tool = grepCommand(secs, q.Dir, query, q.MatchCase)
	default:
		return "", apperr.New(apperr.InvalidInput, "unknown search mode")
	}
	return "{ " + tool.String() + `; echo "exit:$?" >&2; } | ` +
		shellx.Cmd("head", "-c", strconv.Itoa(searchBytes)).String(), nil
}

// findCommand matches names with -iname (or -name), as "*query*" unless
// the query is already a glob. Paths are NUL-terminated, so names with
// newlines come back whole.
func findCommand(secs, dir, query string, matchCase bool) shellx.Command {
	pattern := query
	if !strings.ContainsAny(query, "*?[") {
		pattern = "*" + query + "*"
	}
	nameTest := "-iname"
	if matchCase {
		nameTest = "-name"
	}
	// -mindepth is a global option and goes first; the start folder
	// itself is not a result.
	args := []string{secs, "find", dir, "-mindepth", "1"}
	if dir == "/" {
		args = append(args, "(")
		for i, v := range virtualDirs {
			if i > 0 {
				args = append(args, "-o")
			}
			args = append(args, "-path", v)
		}
		args = append(args, ")", "-prune", "-o")
	}
	args = append(args, nameTest, pattern, "-print0")
	return shellx.Cmd("timeout", args...)
}

// grepCommand prints, for each file, the file name, a NUL, the line number
// and the text around the match. -o with an ERE that takes up to
// snippetContext characters on each side keeps long lines short. Binary
// files, devices and FIFOs are skipped; links are not followed (-r).
func grepCommand(secs, dir, query string, matchCase bool) shellx.Command {
	ctx := strconv.Itoa(snippetContext)
	pattern := ".{0," + ctx + "}" + quoteERE(query) + ".{0," + ctx + "}"
	args := []string{secs, "grep", "-r", "-I", "-H", "-n", "-o", "-Z", "-E", "-D", "skip", "-m", "3"}
	if !matchCase {
		args = append(args, "-i")
	}
	for _, d := range skippedDirs {
		args = append(args, "--exclude-dir="+d)
	}
	args = append(args, "-e", pattern, "--", dir)
	return shellx.Cmd("timeout", args...)
}

// ereSpecial are the characters with a meaning in a POSIX extended regex.
var ereSpecial = regexp.MustCompile(`[.\[\]()*+?{}|^$\\]`)

// quoteERE makes s match itself literally in an extended regex.
func quoteERE(s string) string {
	return ereSpecial.ReplaceAllString(s, `\$0`)
}

func underVirtual(dir string) bool {
	if dir == "/" {
		return true
	}
	for _, v := range virtualDirs {
		if dir == v || strings.HasPrefix(dir, v+"/") {
			return true
		}
	}
	return false
}

type parsed struct {
	SearchResult
	exit int
}

// parseSearch reads the NUL-separated output of find or grep. A record
// cut by the byte cap is dropped; the cap, or more than SearchLimit hits,
// marks the result truncated.
func parseSearch(mode string, stdout, stderr []byte) parsed {
	out := parsed{SearchResult: SearchResult{Hits: []SearchHit{}}, exit: -1}
	full := len(stdout) >= searchBytes
	if mode == SearchName {
		parts := bytes.Split(stdout, []byte{0})
		// The last part follows the last NUL: empty, or cut by the cap.
		parts = parts[:len(parts)-1]
		for _, p := range parts {
			if len(p) == 0 {
				continue
			}
			out.Hits = append(out.Hits, SearchHit{Path: string(p)})
		}
	} else {
		out.Hits = parseGrep(stdout)
	}
	if len(out.Hits) > SearchLimit {
		out.Hits = out.Hits[:SearchLimit]
		out.Truncated = true
	}
	if full {
		out.Truncated = true
	}
	for _, l := range strings.Split(string(stderr), "\n") {
		switch {
		case strings.HasPrefix(l, "exit:"):
			if n, err := strconv.Atoi(strings.TrimPrefix(l, "exit:")); err == nil {
				out.exit = n
			}
		case strings.Contains(l, "Permission denied"):
			out.Denied++
		}
	}
	out.TimedOut = out.exit == 124
	return out
}

// parseGrep reads "path\0line:text\n" records. The path may hold any
// byte but NUL; the text holds no newline.
func parseGrep(b []byte) []SearchHit {
	hits := []SearchHit{}
	for len(b) > 0 {
		nul := bytes.IndexByte(b, 0)
		if nul < 0 {
			break
		}
		p := string(b[:nul])
		rest := b[nul+1:]
		nl := bytes.IndexByte(rest, '\n')
		if nl < 0 {
			break // cut by the byte cap
		}
		rec := rest[:nl]
		b = rest[nl+1:]
		num, text, ok := bytes.Cut(rec, []byte{':'})
		if !ok {
			continue
		}
		n, err := strconv.Atoi(string(num))
		if err != nil {
			continue
		}
		hits = append(hits, SearchHit{Path: p, Line: n, Text: cleanText(text)})
	}
	return hits
}

// cleanText makes server bytes safe to show: valid UTF-8, no control
// characters but tabs, trimmed.
func cleanText(b []byte) string {
	s := strings.ToValidUTF8(string(b), "�")
	s = strings.Map(func(r rune) rune {
		if r == '\t' {
			return ' '
		}
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, s)
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) > 2*snippetContext+200 {
		s = string([]rune(s)[:2*snippetContext+200]) + "…"
	}
	return s
}

func firstLine(b []byte) string {
	for _, l := range strings.Split(string(b), "\n") {
		l = strings.TrimSpace(l)
		if l != "" && !strings.HasPrefix(l, "exit:") {
			return cleanText([]byte(l))
		}
	}
	return "search failed"
}
